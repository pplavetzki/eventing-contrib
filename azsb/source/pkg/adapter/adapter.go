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
	"fmt"
	"strings"

	"github.com/cloudevents/sdk-go/pkg/cloudevents/client"
	"go.uber.org/zap"
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
	Topic            string `envconfig:"AZSB_TOPIC" required:"true"`
	Subcription      string `envconfig:"AZSB_SUBSCRIPTION" required:"true"`
	Name             string `envconfig:"NAME" required:"true"`
	ConnectionString string `envconfig:"AZSB_CONNECTION_STRING" required:"true"`
	KeyType          string `envconfig:"KEY_TYPE" required:"false"`
}

// NewEnvConfig config
func NewEnvConfig() adapter.EnvConfigAccessor {
	return &adapterConfig{}
}

type messageSessionHandler struct {
	MessageSession *azsbus.MessageSession
}

// Adapter struct
type Adapter struct {
	config   *adapterConfig
	ceClient client.Client
	reporter source.StatsReporter
	logger   *zap.Logger
	ctx      context.Context
}

// NewAdapter adapter struct
func NewAdapter(ctx context.Context, processed adapter.EnvConfigAccessor, ceClient client.Client, reporter source.StatsReporter) adapter.Adapter {
	logger := logging.FromContext(ctx).Desugar()
	config := processed.(*adapterConfig)

	return &Adapter{
		ctx:      ctx,
		config:   config,
		ceClient: ceClient,
		reporter: reporter,
		logger:   logger,
	}
}

// Start starts the adapter process
func (a *Adapter) Start(stopCh <-chan struct{}) error {
	a.logger.Info("Starting with config: ",
		zap.String("Topic", a.config.Topic),
		zap.String("SinkURI", a.config.SinkURI),
		zap.String("Name", a.config.Name),
		zap.String("Namespace", a.config.Namespace),
	)

	ns, err := azsbus.NewNamespace(azsbus.NamespaceWithConnectionString(a.config.ConnectionString))
	if err != nil {
		panic(err)
	}
	t, err := ns.NewTopic(a.config.Topic)
	if err != nil {
		a.logger.Error("Getting the Topic from Azure Service Bus failed: ", zap.Error(err))
		return err
	}
	sub, err := t.NewSubscription(a.config.Subcription)
	if err != nil {
		a.logger.Error("Failed to retrieve the newly created Azure Service Bus subscription: ", zap.Error(err))
		return err
	}

	return a.subscriber(a.ctx, sub, stopCh)
}

func (a *Adapter) messageHandler(ctx context.Context) azsbus.HandlerFunc {
	return func(eventCtx context.Context, msg *azsbus.Message) error {
		var err error

		event := cloudevents.NewEvent(cloudevents.VersionV03)

		if strings.Contains(msg.ContentType, "application/cloudevents+json") {
			err = json.Unmarshal(msg.Data, &event)
		} else {
			// Check if payload is a valid json
			if !json.Valid(msg.Data) {
				return fmt.Errorf("json is malformed") // Message is malformed, commit the offset so it won't be reprocessed
			}

			sourceURL := sourcesv1alpha1.AzsbEventSource(a.config.Namespace, a.config.Name, a.config.Topic)
			a.logger.Info("here is the source url", zap.String("sourceUrl", sourceURL))
			event.SetID(msg.ID)
			event.SetTime(*msg.SystemProperties.EnqueuedTime)
			event.SetType(sourcesv1alpha1.AzsbEventType)
			event.SetSource(sourceURL)
			event.SetSubject(msg.Label)
			event.SetDataContentType(cloudevents.ApplicationJSON)

			err = event.SetData(msg.Data)
		}

		if err != nil {
			return msg.Abandon(ctx) // Message is malformed, commit the offset so it won't be reprocessed
		}

		// Check before writing log since event.String() allocates and uses a lot of time
		if ce := a.logger.Check(zap.DebugLevel, "debugging"); ce != nil {
			a.logger.Debug("Sending cloud event", zap.String("event", event.String()))
		}

		rctx, resp, err := a.ceClient.Send(a.ctx, event)

		if err != nil {
			a.logger.Info(err.Error())
			a.logger.Info("Error while sending the message", zap.Error(err))
			if resp != nil {
				for k, e := range resp.FieldErrors {
					a.logger.Info("resp field error", zap.Any(k, e))
				}
			}
			return msg.Abandon(ctx) // Error while sending, don't commit offset
		}

		reportArgs := &source.ReportArgs{
			Namespace:     a.config.Namespace,
			EventSource:   event.Source(),
			EventType:     event.Type(),
			Name:          a.config.Name,
			ResourceGroup: resourceGroup,
		}

		_ = a.reporter.ReportEventCount(reportArgs, cloudevents.HTTPTransportContextFrom(rctx).StatusCode)

		return msg.Complete(ctx)
	}
}

func (a *Adapter) subscriber(ctx context.Context, subscription *azsbus.Subscription, stopCh <-chan struct{}) error {

	go func() {
		err := subscription.Receive(ctx, a.messageHandler(ctx))
		if err != nil {
			a.logger.Error("failed to receive message.", zap.Error(err))
		}
	}()

	for {
		select {
		case <-stopCh:
			a.logger.Info("Shutting down...")
			return nil
		}
	}
}
