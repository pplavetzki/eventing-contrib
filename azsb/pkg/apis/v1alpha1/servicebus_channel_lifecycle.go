package v1alpha1

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/apis/duck/v1alpha1"
)

var azc = apis.NewLivingConditionSet(
	AZServicebusChannelConditionTopicReady,
	AZServicebusChannelConditionDispatcherReady,
	AZServicebusChannelConditionServiceReady,
	AZServicebusChannelConditionEndpointsReady,
	AZServicebusChannelConditionAddressable,
	AZServicebusChannelConditionChannelServiceReady,
	AZServicebusChannelConditionConfigReady)

const (
	// AZServicebusChannelConditionReady has status True when all subconditions below have been set to True.
	AZServicebusChannelConditionReady = apis.ConditionReady

	// AZServicebusChannelConditionDispatcherReady has status True when a Dispatcher deployment is ready
	// Keyed off appsv1.DeploymentAvailable, which means minimum available replicas required are up
	// and running for at least minReadySeconds.
	AZServicebusChannelConditionDispatcherReady apis.ConditionType = "DispatcherReady"

	// AZServicebusChannelConditionServiceReady has status True when a k8s Service is ready. This
	// basically just means it exists because there's no meaningful status in Service. See Endpoints
	// below.
	AZServicebusChannelConditionServiceReady apis.ConditionType = "ServiceReady"

	// AZServicebusChannelConditionEndpointsReady has status True when a k8s Service Endpoints are backed
	// by at least one endpoint.
	AZServicebusChannelConditionEndpointsReady apis.ConditionType = "EndpointsReady"

	// AZServicebusChannelConditionAddressable has status true when this AZServicebusChannel meets
	// the Addressable contract and has a non-empty hostname.
	AZServicebusChannelConditionAddressable apis.ConditionType = "Addressable"

	// AZServicebusChannelConditionChannelServiceReady has status True when a k8s Service representing the channel is ready.
	// Because this uses ExternalName, there are no endpoints to check.
	AZServicebusChannelConditionChannelServiceReady apis.ConditionType = "ChannelServiceReady"

	// AZServicebusChannelConditionTopicReady has status True when the AZ Service Bus topic to use by the channel exists.
	AZServicebusChannelConditionTopicReady apis.ConditionType = "TopicReady"

	// AZServicebusChannelConditionConfigReady has status True when the AZ Service Bus configuration to use by the channel exists and is valid
	// (ie. the connection has been established).
	AZServicebusChannelConditionConfigReady apis.ConditionType = "ConfigurationReady"
)

// GetCondition returns the condition currently associated with the given type, or nil.
func (cs *AZServicebusChannelStatus) GetCondition(t apis.ConditionType) *apis.Condition {
	return azc.Manage(cs).GetCondition(t)
}

// IsReady returns true if the resource is ready overall.
func (cs *AZServicebusChannelStatus) IsReady() bool {
	return azc.Manage(cs).IsHappy()
}

// InitializeConditions sets relevant unset conditions to Unknown state.
func (cs *AZServicebusChannelStatus) InitializeConditions() {
	azc.Manage(cs).InitializeConditions()
}

// SetAddress sets the address (as part of Addressable contract) and marks the correct condition.
func (cs *AZServicebusChannelStatus) SetAddress(url *apis.URL) {
	if cs.Address == nil {
		cs.Address = &v1alpha1.Addressable{}
	}
	if url != nil {
		cs.Address.Hostname = url.Host
		cs.Address.URL = url
		azc.Manage(cs).MarkTrue(AZServicebusChannelConditionAddressable)
	} else {
		cs.Address.Hostname = ""
		cs.Address.URL = nil
		azc.Manage(cs).MarkFalse(AZServicebusChannelConditionAddressable, "EmptyHostname", "hostname is the empty string")
	}
}

// MarkDispatcherFailed marks field
func (cs *AZServicebusChannelStatus) MarkDispatcherFailed(reason, messageFormat string, messageA ...interface{}) {
	azc.Manage(cs).MarkFalse(AZServicebusChannelConditionDispatcherReady, reason, messageFormat, messageA...)
}

// MarkDispatcherUnknown dispatch unknown
func (cs *AZServicebusChannelStatus) MarkDispatcherUnknown(reason, messageFormat string, messageA ...interface{}) {
	azc.Manage(cs).MarkUnknown(AZServicebusChannelConditionDispatcherReady, reason, messageFormat, messageA...)
}

// PropagateDispatcherStatus status
// TODO: Unify this with the ones from Eventing. Say: Broker, Trigger.
func (cs *AZServicebusChannelStatus) PropagateDispatcherStatus(ds *appsv1.DeploymentStatus) {
	for _, cond := range ds.Conditions {
		if cond.Type == appsv1.DeploymentAvailable {
			if cond.Status == corev1.ConditionTrue {
				azc.Manage(cs).MarkTrue(AZServicebusChannelConditionDispatcherReady)
			} else if cond.Status == corev1.ConditionFalse {
				cs.MarkDispatcherFailed("DispatcherDeploymentFalse", "The status of Dispatcher Deployment is False: %s : %s", cond.Reason, cond.Message)
			} else if cond.Status == corev1.ConditionUnknown {
				cs.MarkDispatcherUnknown("DispatcherDeploymentUnknown", "The status of Dispatcher Deployment is Unknown: %s : %s", cond.Reason, cond.Message)
			}
		}
	}
}

// MarkServiceFailed service failed
func (cs *AZServicebusChannelStatus) MarkServiceFailed(reason, messageFormat string, messageA ...interface{}) {
	azc.Manage(cs).MarkFalse(AZServicebusChannelConditionServiceReady, reason, messageFormat, messageA...)
}

// MarkServiceUnknown unknown
func (cs *AZServicebusChannelStatus) MarkServiceUnknown(reason, messageFormat string, messageA ...interface{}) {
	azc.Manage(cs).MarkUnknown(AZServicebusChannelConditionServiceReady, reason, messageFormat, messageA...)
}

// MarkServiceTrue true
func (cs *AZServicebusChannelStatus) MarkServiceTrue() {
	azc.Manage(cs).MarkTrue(AZServicebusChannelConditionServiceReady)
}

// MarkChannelServiceFailed failed
func (cs *AZServicebusChannelStatus) MarkChannelServiceFailed(reason, messageFormat string, messageA ...interface{}) {
	azc.Manage(cs).MarkFalse(AZServicebusChannelConditionChannelServiceReady, reason, messageFormat, messageA...)
}

// MarkChannelServiceTrue service true
func (cs *AZServicebusChannelStatus) MarkChannelServiceTrue() {
	azc.Manage(cs).MarkTrue(AZServicebusChannelConditionChannelServiceReady)
}

// MarkEndpointsFailed failed
func (cs *AZServicebusChannelStatus) MarkEndpointsFailed(reason, messageFormat string, messageA ...interface{}) {
	azc.Manage(cs).MarkFalse(AZServicebusChannelConditionEndpointsReady, reason, messageFormat, messageA...)
}

// MarkEndpointsTrue ture
func (cs *AZServicebusChannelStatus) MarkEndpointsTrue() {
	azc.Manage(cs).MarkTrue(AZServicebusChannelConditionEndpointsReady)
}

// MarkTopicTrue true
func (cs *AZServicebusChannelStatus) MarkTopicTrue() {
	azc.Manage(cs).MarkTrue(AZServicebusChannelConditionTopicReady)
}

// MarkTopicFailed fialed
func (cs *AZServicebusChannelStatus) MarkTopicFailed(reason, messageFormat string, messageA ...interface{}) {
	azc.Manage(cs).MarkFalse(AZServicebusChannelConditionTopicReady, reason, messageFormat, messageA...)
}

// MarkConfigTrue true
func (cs *AZServicebusChannelStatus) MarkConfigTrue() {
	azc.Manage(cs).MarkTrue(AZServicebusChannelConditionConfigReady)
}

// MarkConfigFailed failed
func (cs *AZServicebusChannelStatus) MarkConfigFailed(reason, messageFormat string, messageA ...interface{}) {
	azc.Manage(cs).MarkFalse(AZServicebusChannelConditionConfigReady, reason, messageFormat, messageA...)
}
