package v1alpha1

import (
	"context"
)

// SetDefaults sets the channel defaults
func (c *AZServicebusChannel) SetDefaults(ctx context.Context) {
	c.Spec.SetDefaults(ctx)
}

// SetDefaults sets the spec defaults
func (cs *AZServicebusChannelSpec) SetDefaults(ctx context.Context) {
	if cs.MaxTopicSize == 0 {
		cs.MaxTopicSize = 1024
	}
}
