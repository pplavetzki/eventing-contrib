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

package v1alpha1

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/webhook/resourcesemantics"

	eventingduck "knative.dev/eventing/pkg/apis/duck/v1alpha1"
	"knative.dev/eventing/pkg/apis/eventing"
	"knative.dev/pkg/apis"
)

func TestAzsbChannelValidation(t *testing.T) {
	aURL, _ := apis.ParseURL("http://example.com/")

	testCases := map[string]struct {
		cr   resourcesemantics.GenericCRD
		want *apis.FieldError
	}{
		"empty spec": {
			cr: &AZServicebusChannel{
				Spec: AZServicebusChannelSpec{},
			},
			want: func() *apis.FieldError {
				var errs *apis.FieldError
				fe := apis.ErrInvalidValue(0, "spec.maxTopicSize")
				errs = errs.Also(fe)
				return errs
			}(),
		},
		"negative maxTopicSize": {
			cr: &AZServicebusChannel{
				Spec: AZServicebusChannelSpec{
					EnablePartition: true,
					MaxTopicSize:    -10,
				},
			},
			want: func() *apis.FieldError {
				fe := apis.ErrInvalidValue(-10, "spec.maxTopicSize")
				return fe
			}(),
		},
		"valid subscribers array": {
			cr: &AZServicebusChannel{
				Spec: AZServicebusChannelSpec{
					EnablePartition: true,
					MaxTopicSize:    10,
					Subscribable: &eventingduck.Subscribable{
						Subscribers: []eventingduck.SubscriberSpec{{
							SubscriberURI: aURL,
							ReplyURI:      aURL,
						}},
					}},
			},
			want: nil,
		},
		"empty subscriber at index 1": {
			cr: &AZServicebusChannel{
				Spec: AZServicebusChannelSpec{
					EnablePartition: true,
					MaxTopicSize:    10,
					Subscribable: &eventingduck.Subscribable{
						Subscribers: []eventingduck.SubscriberSpec{{
							SubscriberURI: aURL,
							ReplyURI:      aURL,
						}, {}},
					}},
			},
			want: func() *apis.FieldError {
				fe := apis.ErrMissingField("spec.subscribable.subscriber[1].replyURI", "spec.subscribable.subscriber[1].subscriberURI")
				fe.Details = "expected at least one of, got none"
				return fe
			}(),
		},
		"two empty subscribers": {
			cr: &AZServicebusChannel{
				Spec: AZServicebusChannelSpec{
					EnablePartition: true,
					MaxTopicSize:    10,
					Subscribable: &eventingduck.Subscribable{
						Subscribers: []eventingduck.SubscriberSpec{{}, {}},
					},
				},
			},
			want: func() *apis.FieldError {
				var errs *apis.FieldError
				fe := apis.ErrMissingField("spec.subscribable.subscriber[0].replyURI", "spec.subscribable.subscriber[0].subscriberURI")
				fe.Details = "expected at least one of, got none"
				errs = errs.Also(fe)
				fe = apis.ErrMissingField("spec.subscribable.subscriber[1].replyURI", "spec.subscribable.subscriber[1].subscriberURI")
				fe.Details = "expected at least one of, got none"
				errs = errs.Also(fe)
				return errs
			}(),
		},
		"invalid scope annotation": {
			cr: &AZServicebusChannel{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						eventing.ScopeAnnotationKey: "notvalid",
					},
				},
				Spec: AZServicebusChannelSpec{
					EnablePartition: true,
					MaxTopicSize:    10,
				},
			},
			want: func() *apis.FieldError {
				fe := apis.ErrInvalidValue("notvalid", "metadata.annotations.[eventing.knative.dev/scope]")
				fe.Details = "expected either 'cluster' or 'namespace'"
				return fe
			}(),
		},
	}

	for n, test := range testCases {
		t.Run(n, func(t *testing.T) {
			got := test.cr.Validate(context.Background())
			if diff := cmp.Diff(test.want.Error(), got.Error()); diff != "" {
				t.Errorf("%s: validate (-want, +got) = %v", n, diff)
			}
		})
	}
}