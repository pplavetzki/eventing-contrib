package dispatcher

import (
	"fmt"

	eventingduck "knative.dev/eventing/pkg/apis/duck/v1alpha1"
)

type subscriptionReference struct {
	UID       string
	Namespace string
	Name      string
	TopicURL  string
	ReplyURL  string
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
