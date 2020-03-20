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

package azsb

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"knative.dev/eventing/pkg/adapter"
	"knative.dev/pkg/source"

	azsbus "github.com/Azure/azure-service-bus-go"
	cloudevents "github.com/cloudevents/sdk-go/legacy"
)

func MockEventSender(ctx context.Context, event cloudevents.Event) (context.Context, *cloudevents.Event, error) {
	return ctx, &event, nil
}

func MockEventResponse(ctx context.Context, err error, msg *azsbus.Message) error {
	return nil
}

func TestAdapterStartFailure(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	ac := &adapterConfig{
		Topic:            "TestTopic",
		Subcription:      "TestSubscription",
		Name:             "TestConfig",
		ConnectionString: "TestConnection",
		EnvConfig: adapter.EnvConfig{
			SinkURI:   "http://sink-name",
			Sink:      "http://sink-name",
			Namespace: "default",
		},
	}
	a := &Adapter{
		config: ac,
	}

	_ = a.Start(make(chan struct{}))
}

func TestPostMessage_ServeHTTP(t *testing.T) {
	aTimestamp := time.Now()
	statsReporter, _ := source.NewStatsReporter()

	testCases := map[string]struct {
		sink    func(http.ResponseWriter, *http.Request)
		msg     *azsbus.Message
		isError bool
	}{
		"happy-path": {
			sink: sinkAccepted,
			msg: &azsbus.Message{
				ContentType: "application/json",
				Data:        []byte("{\"msg\": \"this is a test message\"}"),
				Label:       "Message Label",
				SystemProperties: &azsbus.SystemProperties{
					EnqueuedTime: &aTimestamp,
				},
			},
		},
		"cloud-event-error": {
			sink: sinkRejected,
			msg: &azsbus.Message{
				ContentType: cloudevents.ApplicationCloudEventsJSON,
				Data:        []byte("{\"msg\": \"this is a test message\"}"),
				Label:       "Message Label",
				SystemProperties: &azsbus.SystemProperties{
					EnqueuedTime: &aTimestamp,
				},
			},
			isError: true,
		},
		"cloud-event-happy": {
			sink: sinkAccepted,
			msg: &azsbus.Message{
				ContentType: cloudevents.ApplicationCloudEventsJSON,
				Data:        []byte("{\"type\": \"com.example.someevent\", \"specversion\": \"1.0\", \"source\": \"mycontext\", \"datacontenttype\": \"application/json\"}"),
				Label:       "Message Label",
				SystemProperties: &azsbus.SystemProperties{
					EnqueuedTime: &aTimestamp,
				},
			},
		},
	}
	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {
			msgID, err := uuid.NewUUID()
			if err != nil {
				t.Error("failed to generate uuid:", err)
			}
			tc.msg.ID = msgID.String()
			h := &fakeHandler{
				handler: tc.sink,
			}
			sinkServer := httptest.NewServer(h)
			defer sinkServer.Close()

			ac := &adapterConfig{
				Topic:            "TestTopic",
				Subcription:      "TestSubscription",
				Name:             "TestConfig",
				ConnectionString: "TestConnection",
				EnvConfig: adapter.EnvConfig{
					SinkURI:   sinkServer.URL,
					Sink:      sinkServer.URL,
					Namespace: "default",
				},
			}
			a := &Adapter{
				config:        ac,
				logger:        zap.NewNop(),
				reporter:      statsReporter,
				ctx:           context.TODO(),
				EventResponse: MockEventResponse,
			}

			if err := a.initClient(); err != nil {
				t.Errorf("failed to create cloudevent client, %v", err)
			}

			err = a.ProcessEvent(context.TODO(), tc.msg)

			if tc.isError && err == nil {
				t.Errorf("expected error but got nil")
			} else if !tc.isError && err != nil {
				t.Errorf("unexpected error, got %v", err)
			}
		})
	}
}

type fakeHandler struct {
	body []byte

	handler func(http.ResponseWriter, *http.Request)
}

func (h *fakeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "can not read body", http.StatusBadRequest)
		return
	}
	h.body = body

	defer r.Body.Close()
	h.handler(w, r)
}

func sinkAccepted(writer http.ResponseWriter, req *http.Request) {
	writer.WriteHeader(http.StatusOK)
}

func sinkRejected(writer http.ResponseWriter, _ *http.Request) {
	writer.WriteHeader(http.StatusRequestTimeout)
}
