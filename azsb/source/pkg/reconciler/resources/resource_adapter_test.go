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

package resources

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/eventing-contrib/azsb/source/pkg/apis/sources/v1alpha1"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
)

func TestMakeReceiveAdapter(t *testing.T) {
	args := &ReceiveAdapterArgs{
		Image:         "test-image:v1alpha1",
		LoggingConfig: "logging-config",
		MetricsConfig: "metrics-config",
		Sink:          "test-sink",
		Labels:        map[string]string{"key1": "label1"},
		Source: &v1alpha1.AzsbSource{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "source-test-namespace",
				Name:      "source-test-name-",
			},
			Spec: v1alpha1.AzsbSourceSpec{
				ServiceAccountName: "test-service-account",
				Topic:              "topic1",
				Subscription:       "subscription",
				ConnectionString: v1alpha1.SecretValueFromSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						Key: "myKey",
					},
				},
				Sink: &duckv1beta1.Destination{
					Ref: &corev1.ObjectReference{
						Name:       "test-sink",
						Kind:       "Sink",
						APIVersion: "duck.knative.dev/v1",
					},
				},
			},
		},
	}

	dev := MakeReceiveAdapter(args)
	copy := dev.DeepCopy()

	if dev.Name != copy.Name {
		t.Errorf("Expected name: %s, but got name: %s", dev.Name, copy.Name)
	}
	if dev.Labels["key1"] != copy.Labels["key1"] {
		t.Errorf("Expected key1: %s, but got key1: %s", dev.Labels["key1"], copy.Labels["key1"])
	}
}
