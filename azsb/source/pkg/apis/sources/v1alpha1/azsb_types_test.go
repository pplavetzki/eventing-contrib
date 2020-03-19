package v1alpha1

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
)

func TestAzsbSource_GetGroupVersionKind(t *testing.T) {
	src := AzsbSource{}
	gvk := src.GetGroupVersionKind()

	if gvk.Kind != "AzsbSource" {
		t.Errorf("Should be 'AzsbSource'.")
	}
}

func TestAzsbSource_SetSinkURI(t *testing.T) {
	src := AzsbSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "default",
		},
		Spec: AzsbSourceSpec{
			Sink: &duckv1beta1.Destination{
				URI: &apis.URL{
					Host: "http://example.com",
				},
			},
		},
		Status: AzsbSourceStatus{
			Status: duckv1.Status{
				Conditions: duckv1.Conditions{},
			},
			SinkURI: "http://example.com",
		},
	}
	updated := src.DeepCopy()
	updated.Labels = map[string]string{"hello": "world"}
	updated.Status = AzsbSourceStatus{
		SinkURI: "http://sample.com",
	}
	t.Log("successfully ran ok!\n")
}
