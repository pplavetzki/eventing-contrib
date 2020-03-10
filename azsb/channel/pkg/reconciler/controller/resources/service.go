package resources

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/eventing-contrib/azsb/channel/pkg/apis/messaging/v1alpha1"
	"knative.dev/eventing/pkg/utils"
	"knative.dev/pkg/kmeta"
)

const (
	portName   = "http"
	portNumber = 80
	// MessagingRoleLabel role label
	MessagingRoleLabel = "messaging.knative.dev/role"
	// MessagingRole MessagingRole
	MessagingRole = "az-servicebus-channel"
)

// ServiceOption can be used to optionally modify the K8s service in MakeK8sService.
type ServiceOption func(*corev1.Service) error

// MakeExternalServiceAddress external service
func MakeExternalServiceAddress(namespace, service string) string {
	return fmt.Sprintf("%s.%s.svc.%s", service, namespace, utils.GetClusterDomainName())
}

// MakeChannelServiceName service name
func MakeChannelServiceName(name string) string {
	return fmt.Sprintf("%s-kn-channel", name)
}

// ExternalService is a functional option for MakeK8sService to create a K8s service of type ExternalName
// pointing to the specified service in a namespace.
func ExternalService(namespace, service string) ServiceOption {
	return func(svc *corev1.Service) error {
		// TODO this overrides the current serviceSpec. Is this correct?
		svc.Spec = corev1.ServiceSpec{
			Type:         corev1.ServiceTypeExternalName,
			ExternalName: MakeExternalServiceAddress(namespace, service),
		}
		return nil
	}
}

// MakeK8sService creates a new K8s Service for a Channel resource. It also sets the appropriate
// OwnerReferences on the resource so handleObject can discover the Channel resource that 'owns' it.
// As well as being garbage collected when the Channel is deleted.
func MakeK8sService(azsb *v1alpha1.AZServicebusChannel, opts ...ServiceOption) (*corev1.Service, error) {
	// Add annotations
	svc := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      MakeChannelServiceName(azsb.ObjectMeta.Name),
			Namespace: azsb.Namespace,
			Labels: map[string]string{
				MessagingRoleLabel: MessagingRole,
			},
			OwnerReferences: []metav1.OwnerReference{
				*kmeta.NewControllerRef(azsb),
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:     portName,
					Protocol: corev1.ProtocolTCP,
					Port:     portNumber,
				},
			},
		},
	}
	for _, opt := range opts {
		if err := opt(svc); err != nil {
			return nil, err
		}
	}
	return svc, nil
}
