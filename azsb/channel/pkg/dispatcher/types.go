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
