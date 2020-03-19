package resources

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// MakeDispatcherService creates the Kafka dispatcher service
func MakeDispatcherService(namespace string) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      dispatcherName,
			Namespace: namespace,
			Labels:    dispatcherLabels,
		},
		Spec: corev1.ServiceSpec{
			Selector: dispatcherLabels,
			Ports: []corev1.ServicePort{
				{
					Name:       "http-dispatcher",
					Protocol:   corev1.ProtocolTCP,
					Port:       80,
					TargetPort: intstr.IntOrString{IntVal: 8080},
				},
			},
		},
	}
}
