package v1alpha1

import (
	"context"
	"fmt"

	"knative.dev/pkg/apis"
)

// Validate validates the channel
func (c *AZServicebusChannel) Validate(ctx context.Context) *apis.FieldError {
	return c.Spec.Validate(ctx).ViaField("spec")
}

// Validate validates the channel spec
func (cs *AZServicebusChannelSpec) Validate(ctx context.Context) *apis.FieldError {
	var errs *apis.FieldError

	if cs.MaxTopicSize <= 0 {
		fe := apis.ErrInvalidValue(cs.MaxTopicSize, "maxTopicSize")
		errs = errs.Also(fe)
	}

	if cs.Subscribable != nil {
		for i, subscriber := range cs.Subscribable.Subscribers {
			if subscriber.ReplyURI == nil && subscriber.SubscriberURI == nil {
				fe := apis.ErrMissingField("replyURI", "subscriberURI")
				fe.Details = "expected at least one of, got none"
				errs = errs.Also(fe.ViaField(fmt.Sprintf("subscriber[%d]", i)).ViaField("subscribable"))
			}
		}
	}
	return errs
}
