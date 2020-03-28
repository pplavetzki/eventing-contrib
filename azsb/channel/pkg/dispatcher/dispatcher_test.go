/*
Copyright 2020 The Knative Authors

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

package dispatcher

import (
	"context"
	"testing"

	"go.uber.org/zap"
	eventingchannels "knative.dev/eventing/pkg/channel"
	"knative.dev/eventing/pkg/kncloudevents"
)

func TestInvalidConnectionString(t *testing.T) {
	ce := "Endpoint=sb://example.servicebus.windows.net/;SharedAccessKeyName=;SharedAccessKey="
	_, err := getNewSasInstance(ce)
	if err == nil {
		t.Errorf("Expected error want %s, got %s", "\"key \"SharedAccessKeyName\" must not be empty\"", err)
	}
}

func TestSubscribeError(t *testing.T) {

	sr := subscriptionReference{
		UID:       "xidf-ie20",
		Name:      "subscription-test",
		Namespace: "default",
		ReplyURL:  "http://testing.com",
		TopicURL:  "http://testing.com",
	}
	ce := "Endpoint=sb://example.servicebus.windows.net/;SharedAccessKeyName=ExamplePolicy;SharedAccessKey=XXXXXXXXXXXXXXXXXXXXXXXXXXXXXX="
	ns, err := getNewSasInstance(ce)
	if err != nil {
		t.Error("did not expect error here as we mocked a valid connection string", err)
	}

	ss := &SubscriptionsSupervisor{
		logger:             zap.NewNop(),
		dispatcher:         eventingchannels.NewEventDispatcher(zap.NewNop()),
		connectionEndpoint: ce,
		subscriptions:      make(SubscriptionChannelMapping),
		azNamespace:        ns,
	}

	channelRef := eventingchannels.ChannelReference{
		Name:      "test-channel",
		Namespace: "test-ns",
	}

	_, err = ss.subscribe(context.TODO(), channelRef, sr)
	if err == nil {
		t.Errorf("Expected error want %s, got %s", "error creating consumer", err)
	}
}

func TestSubscribeOk(t *testing.T) {

	sr := subscriptionReference{
		UID:       "xidf-ie20",
		Name:      "subscription-test",
		Namespace: "default",
		ReplyURL:  "http://testing.com",
		TopicURL:  "http://testing.com",
	}
	ce := "Endpoint=sb://example.servicebus.windows.net/;SharedAccessKeyName=ExamplePolicy;SharedAccessKey=XXXXXXXXXXXXXXXXXXXXXXXXXXXXXX="
	ns, err := getNewSasInstance(ce)
	if err != nil {
		t.Error("did not expect error here as we mocked a valid connection string", err)
	}

	ss := &SubscriptionsSupervisor{
		logger:             zap.NewNop(),
		dispatcher:         eventingchannels.NewEventDispatcher(zap.NewNop()),
		connectionEndpoint: ce,
		subscriptions:      make(SubscriptionChannelMapping),
		azNamespace:        ns,
	}

	channelRef := eventingchannels.ChannelReference{
		Name:      "test-channel",
		Namespace: "test-ns",
	}

	_, err = ss.subscribe(context.TODO(), channelRef, sr)
	if err == nil {
		t.Errorf("Expected error want %s, got %s", "error creating consumer", err)
	}
}

func TestNewDispatcher(t *testing.T) {
	kargs := &kncloudevents.ConnectionArgs{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 1000,
	}
	ce := "Endpoint=sb://example.servicebus.windows.net/;SharedAccessKeyName=ExamplePolicy;SharedAccessKey=XXXXXXXXXXXXXXXXXXXXXXXXXXXXXX="
	_, err := NewDispatcher(ce, zap.NewNop(), kargs)

	if err != nil {
		t.Errorf("Expected that we do not have an error, but we have %s", err)
	}
}
