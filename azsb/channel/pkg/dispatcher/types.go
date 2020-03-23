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
	"fmt"

	eventingchannels "knative.dev/eventing/pkg/channel"

	eventingduck "knative.dev/eventing/pkg/apis/duck/v1alpha1"
)

type subscriptionReference struct {
	UID       string
	Namespace string
	Name      string
	TopicURL  string
	ReplyURL  string
	Delivery  eventingchannels.DeliveryOptions
}

func newSubscriptionReference(spec eventingduck.SubscriberSpec) subscriptionReference {
	return subscriptionReference{
		UID:      string(spec.UID),
		TopicURL: spec.SubscriberURI.String(),
		ReplyURL: spec.ReplyURI.String(),
	}
}

func (r *subscriptionReference) String() string {
	return fmt.Sprintf("%s", r.UID)
}
