package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	eventingduck "knative.dev/eventing/pkg/apis/duck/v1alpha1"
	"knative.dev/pkg/apis"
	duckv1alpha1 "knative.dev/pkg/apis/duck/v1alpha1"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AZServicebusChannel is a resource representing a AZServicebus Channel.
type AZServicebusChannel struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of the Channel.
	Spec AZServicebusChannelSpec `json:"spec,omitempty"`

	// Status represents the current state of the AZServicebusChannel. This data may be out of
	// date.
	// +optional
	Status AZServicebusChannelStatus `json:"status,omitempty"`
}

// Check that Channel can be validated, can be defaulted, and has immutable fields.
var _ apis.Validatable = (*AZServicebusChannel)(nil)
var _ apis.Defaultable = (*AZServicebusChannel)(nil)
var _ runtime.Object = (*AZServicebusChannel)(nil)

// AZServicebusChannelSpec defines the specification for a AZServicebusChannel.
type AZServicebusChannelSpec struct {
	// EnablePartition enables partitioning.
	EnablePartition bool `json:"enablePartition"`

	// MaxTopicSize size of the Topic.
	MaxTopicSize int16 `json:"maxTopicSize"`

	// AZServicebusChannel conforms to Duck type Subscribable.
	Subscribable *eventingduck.Subscribable `json:"subscribable,omitempty"`
}

// AZServicebusChannelStatus represents the current state of a AZServicebusChannel.
type AZServicebusChannelStatus struct {
	// inherits duck/v1beta1 Status, which currently provides:
	// * ObservedGeneration - the 'Generation' of the Service that was last processed by the controller.
	// * Conditions - the latest available observations of a resource's current state.
	duckv1beta1.Status `json:",inline"`

	// AZServicebus is Addressable. It currently exposes the endpoint as a
	// fully-qualified DNS name which will distribute traffic over the
	// provided targets from inside the cluster.
	//
	// It generally has the form {channel}.{namespace}.svc.{cluster domain name}
	duckv1alpha1.AddressStatus `json:",inline"`

	// Subscribers is populated with the statuses of each of the Channelable's subscribers.
	eventingduck.SubscribableTypeStatus `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AZServicebusChannelList is a collection of AZServicebusChannels.
type AZServicebusChannelList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AZServicebusChannel `json:"items"`
}

// GetGroupVersionKind returns GroupVersionKind for AZServicebusChannels
func (c *AZServicebusChannel) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("AZServicebusChannel")
}
