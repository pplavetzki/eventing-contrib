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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"knative.dev/eventing/pkg/apis/duck"
	"knative.dev/pkg/apis"
)

const (
	// AzsbConditionReady has status True when the AzsbSource is ready to send events.
	AzsbConditionReady = apis.ConditionReady

	// AzsbConditionSinkProvided has status True when the AzsbSource has been configured with a sink target.
	AzsbConditionSinkProvided apis.ConditionType = "SinkProvided"

	// AzsbConditionDeployed has status True when the AzsbSource has had it's receive adapter deployment created.
	AzsbConditionDeployed apis.ConditionType = "Deployed"

	// AzsbConditionEventTypesProvided has status True when the AzsbSource has been configured with event types.
	AzsbConditionEventTypesProvided apis.ConditionType = "EventTypesProvided"

	// AzsbConditionResources is True when the resources listed for the AzsbSource have been properly
	// parsed and match specified syntax for resource quantities
	AzsbConditionResources apis.ConditionType = "ResourcesCorrect"
)

// AzsbSourceCondSet condition set
var AzsbSourceCondSet = apis.NewLivingConditionSet(
	AzsbConditionSinkProvided,
	AzsbConditionDeployed)

// GetCondition retrieves the condition
func (s *AzsbSourceStatus) GetCondition(t apis.ConditionType) *apis.Condition {
	return AzsbSourceCondSet.Manage(s).GetCondition(t)
}

// IsReady returns true if the resource is ready overall.
func (s *AzsbSourceStatus) IsReady() bool {
	return AzsbSourceCondSet.Manage(s).IsHappy()
}

// InitializeConditions sets relevant unset conditions to Unknown state.
func (s *AzsbSourceStatus) InitializeConditions() {
	AzsbSourceCondSet.Manage(s).InitializeConditions()
}

// MarkSink sets the condition that the source has a sink configured.
func (s *AzsbSourceStatus) MarkSink(uri string) {
	s.SinkURI = uri
	if len(uri) > 0 {
		AzsbSourceCondSet.Manage(s).MarkTrue(AzsbConditionSinkProvided)
	} else {
		AzsbSourceCondSet.Manage(s).MarkUnknown(AzsbConditionSinkProvided, "SinkEmpty", "Sink has resolved to empty.%s", "")
	}
}

// MarkSinkWarnRefDeprecated sets the condition that the source has a sink configured and warns ref is deprecated.
func (s *AzsbSourceStatus) MarkSinkWarnRefDeprecated(uri string) {
	s.SinkURI = uri
	if len(uri) > 0 {
		c := apis.Condition{
			Type:     AzsbConditionSinkProvided,
			Status:   corev1.ConditionTrue,
			Severity: apis.ConditionSeverityError,
			Message:  "Using deprecated object ref fields when specifying spec.sink. Update to spec.sink.ref. These will be removed in a future release.",
		}
		AzsbSourceCondSet.Manage(s).SetCondition(c)
	} else {
		AzsbSourceCondSet.Manage(s).MarkUnknown(AzsbConditionSinkProvided, "SinkEmpty", "Sink has resolved to empty.%s", "")
	}
}

// MarkNoSink sets the condition that the source does not have a sink configured.
func (s *AzsbSourceStatus) MarkNoSink(reason, messageFormat string, messageA ...interface{}) {
	AzsbSourceCondSet.Manage(s).MarkFalse(AzsbConditionSinkProvided, reason, messageFormat, messageA...)
}

// DeploymentIsAvailable deploy is ready
func DeploymentIsAvailable(d *appsv1.DeploymentStatus, def bool) bool {
	// Check if the Deployment is available.
	for _, cond := range d.Conditions {
		if cond.Type == appsv1.DeploymentAvailable {
			return cond.Status == "True"
		}
	}
	return def
}

// MarkDeployed sets the condition that the source has been deployed.
func (s *AzsbSourceStatus) MarkDeployed(d *appsv1.Deployment) {
	if duck.DeploymentIsAvailable(&d.Status, false) {
		AzsbSourceCondSet.Manage(s).MarkTrue(AzsbConditionDeployed)
	} else {
		// I don't know how to propagate the status well, so just give the name of the Deployment
		// for now.
		AzsbSourceCondSet.Manage(s).MarkFalse(AzsbConditionDeployed, "DeploymentUnavailable", "The Deployment '%s' is unavailable.", d.Name)
	}
}

// MarkDeploying sets the condition that the source is deploying.
func (s *AzsbSourceStatus) MarkDeploying(reason, messageFormat string, messageA ...interface{}) {
	AzsbSourceCondSet.Manage(s).MarkUnknown(AzsbConditionDeployed, reason, messageFormat, messageA...)
}

// MarkNotDeployed sets the condition that the source has not been deployed.
func (s *AzsbSourceStatus) MarkNotDeployed(reason, messageFormat string, messageA ...interface{}) {
	AzsbSourceCondSet.Manage(s).MarkFalse(AzsbConditionDeployed, reason, messageFormat, messageA...)
}

// MarkEventTypes sets the condition that the source has created its event types.
func (s *AzsbSourceStatus) MarkEventTypes() {
	AzsbSourceCondSet.Manage(s).MarkTrue(AzsbConditionEventTypesProvided)
}

// MarkNoEventTypes sets the condition that the source does not its event types configured.
func (s *AzsbSourceStatus) MarkNoEventTypes(reason, messageFormat string, messageA ...interface{}) {
	AzsbSourceCondSet.Manage(s).MarkFalse(AzsbConditionEventTypesProvided, reason, messageFormat, messageA...)
}

// MarkResourcesCorrect sets the condition that the source does not its event types configured.
func (s *AzsbSourceStatus) MarkResourcesCorrect() {
	AzsbSourceCondSet.Manage(s).MarkTrue(AzsbConditionResources)
}

// MarkResourcesIncorrect sets the condition that the source does not its event types configured.
func (s *AzsbSourceStatus) MarkResourcesIncorrect(reason, messageFormat string, messageA ...interface{}) {
	AzsbSourceCondSet.Manage(s).MarkFalse(AzsbConditionResources, reason, messageFormat, messageA...)
}
