package resources

import (
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"knative.dev/pkg/system"
)

var (
	serviceAccountName = "az-servicebus-ch-dispatcher"
	dispatcherName     = "az-servicebus-ch-dispatcher"
	dispatcherLabels   = map[string]string{
		"messaging.knative.dev/channel": "az-servicebus-channel",
		"messaging.knative.dev/role":    "dispatcher",
	}
)

// DispatcherArgs args
type DispatcherArgs struct {
	DispatcherNamespace string
	Image               string
}

// MakeDispatcher generates the dispatcher deployment for the AZ Servicebus channel
func MakeDispatcher(args DispatcherArgs) *v1.Deployment {
	replicas := int32(1)

	return &v1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployments",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      dispatcherName,
			Namespace: args.DispatcherNamespace,
		},
		Spec: v1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: dispatcherLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: dispatcherLabels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: serviceAccountName,
					Containers: []corev1.Container{
						{
							Name:  "dispatcher",
							Image: args.Image,
							Env:   makeEnv(),
							Ports: []corev1.ContainerPort{{
								Name:          "metrics",
								ContainerPort: 9090,
							}},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "config-kafka",
									MountPath: "/etc/config-kafka",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "config-kafka",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "config-kafka",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func makeEnv() []corev1.EnvVar {
	return []corev1.EnvVar{{
		Name:  system.NamespaceEnvKey,
		Value: system.Namespace(),
	}, {
		Name:  "METRICS_DOMAIN",
		Value: "knative.dev/eventing",
	}, {
		Name:  "CONFIG_LOGGING_NAME",
		Value: "config-logging",
	}}
}
