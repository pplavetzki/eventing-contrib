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

package testing

import (
	"context"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	duckv1alpha1 "knative.dev/eventing/pkg/apis/duck/v1alpha1"
	"knative.dev/pkg/apis"

	"knative.dev/eventing-contrib/azsb/channel/pkg/apis/messaging/v1alpha1"
)

// AzsbChannelOption enables further configuration of a AZServicebusChannel.
type AzsbChannelOption func(*v1alpha1.AZServicebusChannel)

// NewAZServicebusChannel creates an AZServicebusChannel with AzsbChannelOptions.
func NewAZServicebusChannel(name, namespace string, ncopt ...AzsbChannelOption) *v1alpha1.AZServicebusChannel {
	nc := &v1alpha1.AZServicebusChannel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1alpha1.AZServicebusChannelSpec{},
	}
	for _, opt := range ncopt {
		opt(nc)
	}
	nc.SetDefaults(context.Background())
	return nc
}

// WithReady marks a AZServicebusChannel as being ready
// The dispatcher reconciler does not set the ready status, instead the controller reconciler does
// For testing, we need to be able to set the status to ready
func WithReady(nc *v1alpha1.AZServicebusChannel) {
	cs := apis.NewLivingConditionSet()
	cs.Manage(&nc.Status).MarkTrue(v1alpha1.AZServicebusChannelConditionReady)
}

// WithNotReady status condition
func WithNotReady(reason, messageFormat string) AzsbChannelOption {
	return func(nc *v1alpha1.AZServicebusChannel) {
		cs := apis.NewLivingConditionSet()
		cs.Manage(&nc.Status).MarkFalse(v1alpha1.AZServicebusChannelConditionReady, reason, messageFormat)
	}
}

// WithAzsbInitChannelConditions status condition
func WithAzsbInitChannelConditions(nc *v1alpha1.AZServicebusChannel) {
	nc.Status.InitializeConditions()
}

// WithAZServicebusChannelFinalizer status condition
func WithAZServicebusChannelFinalizer(nc *v1alpha1.AZServicebusChannel) {
	nc.Finalizers = []string{"az-servicebus-ch-dispatcher"}
}

// WithAZServicebusChannelDeleted status condition
func WithAZServicebusChannelDeleted(nc *v1alpha1.AZServicebusChannel) {
	deleteTime := metav1.NewTime(time.Unix(1e9, 0))
	nc.ObjectMeta.SetDeletionTimestamp(&deleteTime)
}

// WithAZServicebusChannelTopicReady topic is ready
func WithAZServicebusChannelTopicReady() AzsbChannelOption {
	return func(nc *v1alpha1.AZServicebusChannel) {
		nc.Status.MarkTopicTrue()
	}
}

// WithAzServicebusChannelConfigReady configuration ready
func WithAzServicebusChannelConfigReady() AzsbChannelOption {
	return func(nc *v1alpha1.AZServicebusChannel) {
		nc.Status.MarkConfigTrue()
	}
}

// WithAZServicebusChannelDeploymentNotReady status condition
func WithAZServicebusChannelDeploymentNotReady(reason, message string) AzsbChannelOption {
	return func(nc *v1alpha1.AZServicebusChannel) {
		nc.Status.MarkDispatcherFailed(reason, message)
	}
}

// WithAZServicebusChannelDeploymentReady status condition
func WithAZServicebusChannelDeploymentReady() AzsbChannelOption {
	return func(nc *v1alpha1.AZServicebusChannel) {
		nc.Status.PropagateDispatcherStatus(&appsv1.DeploymentStatus{Conditions: []appsv1.DeploymentCondition{{Type: appsv1.DeploymentAvailable, Status: corev1.ConditionTrue}}})
	}
}

// WithAZServicebusChannelServiceNotReady status condition
func WithAZServicebusChannelServiceNotReady(reason, message string) AzsbChannelOption {
	return func(nc *v1alpha1.AZServicebusChannel) {
		nc.Status.MarkServiceFailed(reason, message)
	}
}

// WithAZServicebusChannelServiceReady status condition
func WithAZServicebusChannelServiceReady() AzsbChannelOption {
	return func(nc *v1alpha1.AZServicebusChannel) {
		nc.Status.MarkServiceTrue()
	}
}

// WithAZServicebusChannelChannelServicetNotReady status condition
func WithAZServicebusChannelChannelServicetNotReady(reason, message string) AzsbChannelOption {
	return func(nc *v1alpha1.AZServicebusChannel) {
		nc.Status.MarkChannelServiceFailed(reason, message)
	}
}

// WithAZServicebusChannelChannelServiceReady status condition
func WithAZServicebusChannelChannelServiceReady() AzsbChannelOption {
	return func(nc *v1alpha1.AZServicebusChannel) {
		nc.Status.MarkChannelServiceTrue()
	}
}

// WithAZServicebusChannelEndpointsNotReady status condition
func WithAZServicebusChannelEndpointsNotReady(reason, message string) AzsbChannelOption {
	return func(nc *v1alpha1.AZServicebusChannel) {
		nc.Status.MarkEndpointsFailed(reason, message)
	}
}

// WithAZServicebusChannelEndpointsReady status condition
func WithAZServicebusChannelEndpointsReady() AzsbChannelOption {
	return func(nc *v1alpha1.AZServicebusChannel) {
		nc.Status.MarkEndpointsTrue()
	}
}

// WithAZServicebusChannelSubscribers status condition
func WithAZServicebusChannelSubscribers(t *testing.T, subscriberURI string) AzsbChannelOption {
	s, err := apis.ParseURL(subscriberURI)
	if err != nil {
		t.Errorf("cannot parse url: %v", err)
	}
	return func(nc *v1alpha1.AZServicebusChannel) {
		nc.Spec.Subscribable = &duckv1alpha1.Subscribable{
			Subscribers: []duckv1alpha1.SubscriberSpec{
				{
					UID:               "",
					Generation:        0,
					SubscriberURI:     s,
					ReplyURI:          nil,
					DeadLetterSinkURI: nil,
				},
			},
		}
	}
}

// WithAZServicebusChannelSubscribableStatus status condition
func WithAZServicebusChannelSubscribableStatus(ready corev1.ConditionStatus, message string) AzsbChannelOption {
	return func(nc *v1alpha1.AZServicebusChannel) {
		nc.Status.SubscribableStatus = &duckv1alpha1.SubscribableStatus{
			Subscribers: []duckv1alpha1.SubscriberStatus{
				{
					Ready:   ready,
					Message: message,
				},
			},
		}
	}
}

// WithAZServicebusChannelAddress status condition
func WithAZServicebusChannelAddress(a string) AzsbChannelOption {
	return func(nc *v1alpha1.AZServicebusChannel) {
		nc.Status.SetAddress(&apis.URL{
			Scheme: "http",
			Host:   a,
		})
	}
}

// WithAzsbFinalizer adds finalizer
func WithAzsbFinalizer(finalizerName string) AzsbChannelOption {
	return func(nc *v1alpha1.AZServicebusChannel) {
		finalizers := sets.NewString(nc.Finalizers...)
		finalizers.Insert(finalizerName)
		nc.SetFinalizers(finalizers.List())
	}
}
