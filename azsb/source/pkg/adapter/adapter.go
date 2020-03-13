/*
Copyright 2019 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package azsb

import (
	"context"
	"encoding/json"
	"strings"

	"go.uber.org/zap"
	"gopkg.in/jcmturner/gokrb5.v7/client"
	"knative.dev/eventing/pkg/adapter"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/source"

	azsbus "github.com/Azure/azure-service-bus-go"
	cloudevents "github.com/cloudevents/sdk-go"
	sourcesv1alpha1 "knative.dev/eventing-contrib/azsb/source/pkg/apis/sources/v1alpha1"
)

const (
	resourceGroup = "azsbsources.sources.knative.dev"
)

type adapterConfig struct {
	adapter.EnvConfig
	Topics           string `envconfig:"AZSB_TOPICS" required:"true"`
	Name             string `envconfig:"NAME" required:"true"`
	ConnectionString string `envconfig:"AZSB_CONNECTION_STRING" required:"true"`
	KeyType          string `envconfig:"KEY_TYPE" required:"false"`
}

// NewEnvConfig config
func NewEnvConfig() adapter.EnvConfigAccessor {
	return &adapterConfig{}
}

// Adapter struct
type Adapter struct {
	config        *adapterConfig
	ceClient      client.Client
	reporter      source.StatsReporter
	logger        *zap.Logger
	keyTypeMapper func([]byte) interface{}
}

// NewAdapter adapter struct
func NewAdapter(ctx context.Context, processed adapter.EnvConfigAccessor, ceClient client.Client, reporter source.StatsReporter) adapter.Adapter {
	logger := logging.FromContext(ctx).Desugar()
	config := processed.(*adapterConfig)

	return &Adapter{
		config:        config,
		ceClient:      ceClient,
		reporter:      reporter,
		logger:        logger,
		keyTypeMapper: getKeyTypeMapper(config.KeyType),
	}
}

// Start starts the adapter process
func (a *Adapter) Start(stopCh <-chan struct{}) error {
	a.logger.Info("Starting with config: ",
		zap.String("Topics", a.config.Topics),
		zap.String("ConnectionString", a.config.ConnectionString),
		zap.String("SinkURI", a.config.SinkURI),
		zap.String("Name", a.config.Name),
		zap.String("Namespace", a.config.Namespace),
	)

	ns, err := azsbus.NewNamespace(azsbus.NamespaceWithConnectionString(connection))...)
	if err != nil {
		panic(err)
	}
	t, err := ns.NewTopic(strings.Join(a.config.Topics, ","))
	if err != nil {
		s.logger.Error("Getting the Topic from Azure Service Bus failed: ", zap.Error(err))
		return err
	}
	sub, err := t.NewSubscription(subscription.Name)
	if err != nil {
		s.logger.Error("Failed to retrieve the newly created Azure Service Bus subscription: ", zap.Error(err))
		return nil, err
	}
	s.logger.Info("after the context in subscribe")
	go func() {
		err = sub.Receive(ctx, messageHandler())
		if err != nil {
			log.Println(err)
		}
	}()
}

func (a *Adapter) Handle(ctx context.Context, msg *azsbus.Message) (bool, error) {
	var err error
	event := cloudevents.NewEvent(cloudevents.VersionV1)

	if strings.Contains(msg.ContentType, "application/cloudevents+json") {
		err = json.Unmarshal(msg.Value, &event)
	} else {
		// Check if payload is a valid json
		if !json.Valid(msg.Data) {
			return true, nil // Message is malformed, commit the offset so it won't be reprocessed
		}

		event.SetID(msg.ID)
		event.SetTime(msg.TTL)
		event.SetType(sourcesv1alpha1.AzsbEventType)
		event.SetSource(sourcesv1alpha1.AzsbEventSource(a.config.Namespace, a.config.Name, msg.Topic))
		event.SetSubject(msg.Label)
		event.SetDataContentType(cloudevents.ApplicationJSON)

		// dumpKafkaMetaToEvent(&event, a.keyTypeMapper, msg)

		err = event.SetData(msg.Data)
	}

	if err != nil {
		return true, err // Message is malformed, commit the offset so it won't be reprocessed
	}

	// Check before writing log since event.String() allocates and uses a lot of time
	if ce := a.logger.Check(zap.DebugLevel, "debugging"); ce != nil {
		a.logger.Debug("Sending cloud event", zap.String("event", event.String()))
	}

	rctx, _, err := a.ceClient.Send(ctx, event)

	if err != nil {
		a.logger.Debug("Error while sending the message", zap.Error(err))
		return false, err // Error while sending, don't commit offset
	}

	reportArgs := &source.ReportArgs{
		Namespace:     a.config.Namespace,
		EventSource:   event.Source(),
		EventType:     event.Type(),
		Name:          a.config.Name,
		ResourceGroup: resourceGroup,
	}

	_ = a.reporter.ReportEventCount(reportArgs, cloudevents.HTTPTransportContextFrom(rctx).StatusCode)
	return true, nil
}
