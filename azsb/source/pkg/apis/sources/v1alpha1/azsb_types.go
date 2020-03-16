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

package v1alpha1

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
	"knative.dev/pkg/kmeta"
)

// AzsbSource primary datatype for knative source
// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// AzsbSource is the Schema for the AzsbSource API.
// +k8s:openapi-gen=true
type AzsbSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AzsbSourceSpec   `json:"spec,omitempty"`
	Status AzsbSourceStatus `json:"status,omitempty"`
}

// Check that KafkaSource can be validated and can be defaulted.
var _ runtime.Object = (*AzsbSource)(nil)
var _ kmeta.OwnerRefable = (*AzsbSource)(nil)

// AzsbRequestsSpec requests for cpu/memory
type AzsbRequestsSpec struct {
	ResourceCPU    string `json:"cpu,omitempty"`
	ResourceMemory string `json:"memory,omitempty"`
}

// AzsbLimitsSpec limits for cpu/memory
type AzsbLimitsSpec struct {
	ResourceCPU    string `json:"cpu,omitempty"`
	ResourceMemory string `json:"memory,omitempty"`
}

// AzsbResourceSpec resource wrapper
type AzsbResourceSpec struct {
	Requests AzsbRequestsSpec `json:"requests,omitempty"`
	Limits   AzsbLimitsSpec   `json:"limits,omitempty"`
}

const (
	// AzsbEventType is the Azure Service Bus CloudEvent type.
	AzsbEventType = "dev.knative.azsb.event"
	// AzsbKeyTypeLabel label
	AzsbKeyTypeLabel = "azsbsources.sources.knative.dev/key-type"
)

// AzsbKeyTypeAllowed allowed key types
var AzsbKeyTypeAllowed = []string{"string", "int", "float", "byte-array"}

// AzsbSourceSpec AzsbSource spec
type AzsbSourceSpec struct {
	// Topic to consume messages from
	// +required
	Topic string `json:"topic"`

	// Subscription name of the subscription to the topic
	// +required
	Subscription string `json:"subscription"`

	// ConnectionString credentials to connect to the topics
	// +required
	ConnectionString SecretValueFromSource `json:"connectionString"`

	// Sink is a reference to an object that will resolve to a domain name to use as the sink.
	// +optional
	Sink *duckv1beta1.Destination `json:"sink,omitempty"`

	// ServiceAccoutName is the name of the ServiceAccount that will be used to run the Receive
	// Adapter Deployment.
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// Resource limits and Request specifications of the Receive Adapter Deployment
	Resources AzsbResourceSpec `json:"resources,omitempty"`
}

// AzsbEventSource returns the Kafka CloudEvent source.
func AzsbEventSource(namespace, azsbSourceName, topic string) string {
	return fmt.Sprintf("/apis/v1/namespaces/%s/azsbsources/%s#%s", namespace, azsbSourceName, topic)
}

// AzsbSourceStatus status for the source
type AzsbSourceStatus struct {
	// inherits duck/v1alpha1 Status, which currently provides:
	// * ObservedGeneration - the 'Generation' of the Service that was last processed by the controller.
	// * Conditions - the latest available observations of a resource's current state.
	duckv1.Status `json:",inline"`

	// SinkURI is the current active sink URI that has been configured for the AzsbSource.
	// +optional
	SinkURI string `json:"sinkUri,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AzsbSourceList contains a list of AzsbSources.
type AzsbSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AzsbSource `json:"items"`
}

// SecretValueFromSource represents the source of a secret value
type SecretValueFromSource struct {
	// The Secret key to select from.
	SecretKeyRef *corev1.SecretKeySelector `json:"secretKeyRef,omitempty"`
}

// GetGroupVersionKind this proides version of type
func (s *AzsbSource) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("AzsbSource")
}
