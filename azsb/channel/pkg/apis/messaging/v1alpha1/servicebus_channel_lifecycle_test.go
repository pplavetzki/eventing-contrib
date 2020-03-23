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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	eventingduck "knative.dev/eventing/pkg/apis/duck/v1alpha1"
	"knative.dev/pkg/apis"
	duckv1alpha1 "knative.dev/pkg/apis/duck/v1alpha1"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
)

var condReady = apis.Condition{
	Type:   AZServicebusChannelConditionReady,
	Status: corev1.ConditionTrue,
}

var condDispatcherReady = apis.Condition{
	Type:   AZServicebusChannelConditionDispatcherReady,
	Status: corev1.ConditionTrue,
}

var condDispatcherNotReady = apis.Condition{
	Type:   AZServicebusChannelConditionDispatcherReady,
	Status: corev1.ConditionFalse,
}

var condDispatcherServiceReady = apis.Condition{
	Type:   AZServicebusChannelConditionServiceReady,
	Status: corev1.ConditionTrue,
}

var condDispatcherEndpointsReady = apis.Condition{
	Type:   AZServicebusChannelConditionEndpointsReady,
	Status: corev1.ConditionTrue,
}

var condDispatcherAddressable = apis.Condition{
	Type:   AZServicebusChannelConditionAddressable,
	Status: corev1.ConditionTrue,
}

var deploymentConditionReady = appsv1.DeploymentCondition{
	Type:   appsv1.DeploymentAvailable,
	Status: corev1.ConditionTrue,
}

var deploymentConditionNotReady = appsv1.DeploymentCondition{
	Type:   appsv1.DeploymentAvailable,
	Status: corev1.ConditionFalse,
}

var deploymentStatusReady = &appsv1.DeploymentStatus{Conditions: []appsv1.DeploymentCondition{deploymentConditionReady}}
var deploymentStatusNotReady = &appsv1.DeploymentStatus{Conditions: []appsv1.DeploymentCondition{deploymentConditionNotReady}}

var ignoreAllButTypeAndStatus = cmpopts.IgnoreFields(
	apis.Condition{},
	"LastTransitionTime", "Message", "Reason", "Severity")

var ignoreLastTransitionTime = cmpopts.IgnoreFields(apis.Condition{}, "LastTransitionTime")

func TestChannelGetCondition(t *testing.T) {
	tests := []struct {
		name      string
		cs        *AZServicebusChannelStatus
		condQuery apis.ConditionType
		want      *apis.Condition
	}{{
		name: "single condition",
		cs: &AZServicebusChannelStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{
					condReady,
				},
			},
		},
		condQuery: apis.ConditionReady,
		want:      &condReady,
	}, {
		name: "unknown condition",
		cs: &AZServicebusChannelStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{
					condReady,
					condDispatcherNotReady,
				},
			},
		},
		condQuery: apis.ConditionType("foo"),
		want:      nil,
	}}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.cs.GetCondition(test.condQuery)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("unexpected condition (-want, +got) = %v", diff)
			}
		})
	}
}

func TestChannelInitializeConditions(t *testing.T) {
	tests := []struct {
		name string
		cs   *AZServicebusChannelStatus
		want *AZServicebusChannelStatus
	}{{
		name: "empty",
		cs:   &AZServicebusChannelStatus{},
		want: &AZServicebusChannelStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{{
					Type:   AZServicebusChannelConditionAddressable,
					Status: corev1.ConditionUnknown,
				}, {
					Type:   AZServicebusChannelConditionChannelServiceReady,
					Status: corev1.ConditionUnknown,
				}, {
					Type:   AZServicebusChannelConditionConfigReady,
					Status: corev1.ConditionUnknown,
				}, {
					Type:   AZServicebusChannelConditionDispatcherReady,
					Status: corev1.ConditionUnknown,
				}, {
					Type:   AZServicebusChannelConditionEndpointsReady,
					Status: corev1.ConditionUnknown,
				}, {
					Type:   AZServicebusChannelConditionReady,
					Status: corev1.ConditionUnknown,
				}, {
					Type:   AZServicebusChannelConditionServiceReady,
					Status: corev1.ConditionUnknown,
				}, {
					Type:   AZServicebusChannelConditionTopicReady,
					Status: corev1.ConditionUnknown,
				}},
			},
		},
	}, {
		name: "one false",
		cs: &AZServicebusChannelStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{{
					Type:   AZServicebusChannelConditionDispatcherReady,
					Status: corev1.ConditionFalse,
				}},
			},
		},
		want: &AZServicebusChannelStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{{
					Type:   AZServicebusChannelConditionAddressable,
					Status: corev1.ConditionUnknown,
				}, {
					Type:   AZServicebusChannelConditionChannelServiceReady,
					Status: corev1.ConditionUnknown,
				}, {
					Type:   AZServicebusChannelConditionConfigReady,
					Status: corev1.ConditionUnknown,
				}, {
					Type:   AZServicebusChannelConditionDispatcherReady,
					Status: corev1.ConditionFalse,
				}, {
					Type:   AZServicebusChannelConditionEndpointsReady,
					Status: corev1.ConditionUnknown,
				}, {
					Type:   AZServicebusChannelConditionReady,
					Status: corev1.ConditionUnknown,
				}, {
					Type:   AZServicebusChannelConditionServiceReady,
					Status: corev1.ConditionUnknown,
				}, {
					Type:   AZServicebusChannelConditionTopicReady,
					Status: corev1.ConditionUnknown,
				}},
			},
		},
	}, {
		name: "one true",
		cs: &AZServicebusChannelStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{{
					Type:   AZServicebusChannelConditionDispatcherReady,
					Status: corev1.ConditionTrue,
				}},
			},
		},
		want: &AZServicebusChannelStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{{
					Type:   AZServicebusChannelConditionAddressable,
					Status: corev1.ConditionUnknown,
				}, {
					Type:   AZServicebusChannelConditionChannelServiceReady,
					Status: corev1.ConditionUnknown,
				}, {
					Type:   AZServicebusChannelConditionConfigReady,
					Status: corev1.ConditionUnknown,
				}, {
					Type:   AZServicebusChannelConditionDispatcherReady,
					Status: corev1.ConditionTrue,
				}, {
					Type:   AZServicebusChannelConditionEndpointsReady,
					Status: corev1.ConditionUnknown,
				}, {
					Type:   AZServicebusChannelConditionReady,
					Status: corev1.ConditionUnknown,
				}, {
					Type:   AZServicebusChannelConditionServiceReady,
					Status: corev1.ConditionUnknown,
				}, {
					Type:   AZServicebusChannelConditionTopicReady,
					Status: corev1.ConditionUnknown,
				}},
			},
		},
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.cs.InitializeConditions()
			if diff := cmp.Diff(test.want, test.cs, ignoreAllButTypeAndStatus); diff != "" {
				t.Errorf("unexpected conditions (-want, +got) = %v", diff)
			}
		})
	}
}

func TestChannelIsReady(t *testing.T) {
	tests := []struct {
		name                    string
		markServiceReady        bool
		markChannelServiceReady bool
		markConfigurationReady  bool
		setAddress              bool
		markEndpointsReady      bool
		markTopicReady          bool
		wantReady               bool
		dispatcherStatus        *appsv1.DeploymentStatus
	}{{
		name:                    "all happy",
		markServiceReady:        true,
		markChannelServiceReady: true,
		markConfigurationReady:  true,
		markEndpointsReady:      true,
		dispatcherStatus:        deploymentStatusReady,
		setAddress:              true,
		markTopicReady:          true,
		wantReady:               true,
	}, {
		name:                    "service not ready",
		markServiceReady:        false,
		markChannelServiceReady: false,
		markConfigurationReady:  true,
		markEndpointsReady:      true,
		dispatcherStatus:        deploymentStatusReady,
		setAddress:              true,
		markTopicReady:          true,
		wantReady:               false,
	}, {
		name:                    "endpoints not ready",
		markServiceReady:        true,
		markChannelServiceReady: false,
		markConfigurationReady:  true,
		markEndpointsReady:      false,
		dispatcherStatus:        deploymentStatusReady,
		setAddress:              true,
		markTopicReady:          true,
		wantReady:               false,
	}, {
		name:                    "deployment not ready",
		markServiceReady:        true,
		markConfigurationReady:  true,
		markEndpointsReady:      true,
		markChannelServiceReady: false,
		dispatcherStatus:        deploymentStatusNotReady,
		setAddress:              true,
		markTopicReady:          true,
		wantReady:               false,
	}, {
		name:                    "address not set",
		markServiceReady:        true,
		markConfigurationReady:  true,
		markChannelServiceReady: false,
		markEndpointsReady:      true,
		dispatcherStatus:        deploymentStatusReady,
		setAddress:              false,
		markTopicReady:          true,
		wantReady:               false,
	}, {
		name:                    "channel service not ready",
		markServiceReady:        true,
		markConfigurationReady:  true,
		markChannelServiceReady: false,
		markEndpointsReady:      true,
		dispatcherStatus:        deploymentStatusReady,
		setAddress:              true,
		markTopicReady:          true,
		wantReady:               false,
	}, {
		name:                    "topic not ready",
		markServiceReady:        true,
		markConfigurationReady:  true,
		markChannelServiceReady: true,
		markEndpointsReady:      true,
		dispatcherStatus:        deploymentStatusReady,
		setAddress:              true,
		markTopicReady:          false,
		wantReady:               false,
	}}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cs := &AZServicebusChannelStatus{}
			cs.InitializeConditions()
			if test.markServiceReady {
				cs.MarkServiceTrue()
			} else {
				cs.MarkServiceFailed("NotReadyService", "testing")
			}
			if test.markChannelServiceReady {
				cs.MarkChannelServiceTrue()
			} else {
				cs.MarkChannelServiceFailed("NotReadyChannelService", "testing")
			}
			if test.markConfigurationReady {
				cs.MarkConfigTrue()
			} else {
				cs.MarkConfigFailed("NotReadyConfiguration", "testing")
			}
			if test.setAddress {
				cs.SetAddress(&apis.URL{Scheme: "http", Host: "foo.bar"})
			}
			if test.markEndpointsReady {
				cs.MarkEndpointsTrue()
			} else {
				cs.MarkEndpointsFailed("NotReadyEndpoints", "testing")
			}
			if test.dispatcherStatus != nil {
				cs.PropagateDispatcherStatus(test.dispatcherStatus)
			} else {
				cs.MarkDispatcherFailed("NotReadyDispatcher", "testing")
			}
			if test.markTopicReady {
				cs.MarkTopicTrue()
			} else {
				cs.MarkTopicFailed("NotReadyTopic", "testing")
			}
			got := cs.IsReady()
			if test.wantReady != got {
				t.Errorf("unexpected readiness: want %v, got %v", test.wantReady, got)
			}
		})
	}
}

func TestAzsbChannelStatus_SetAddressable(t *testing.T) {
	testCases := map[string]struct {
		url  *apis.URL
		want *AZServicebusChannelStatus
	}{
		"empty string": {
			want: &AZServicebusChannelStatus{
				Status: duckv1beta1.Status{
					Conditions: []apis.Condition{
						{
							Type:   AZServicebusChannelConditionAddressable,
							Status: corev1.ConditionFalse,
						},
						// Note that Ready is here because when the condition is marked False, duck
						// automatically sets Ready to false.
						{
							Type:   AZServicebusChannelConditionReady,
							Status: corev1.ConditionFalse,
						},
					},
				},
				AddressStatus:          duckv1alpha1.AddressStatus{Address: &duckv1alpha1.Addressable{}},
				SubscribableTypeStatus: eventingduck.SubscribableTypeStatus{},
			},
		},
		"has domain": {
			url: &apis.URL{Scheme: "http", Host: "test-domain"},
			want: &AZServicebusChannelStatus{
				AddressStatus: duckv1alpha1.AddressStatus{
					Address: &duckv1alpha1.Addressable{
						Addressable: duckv1beta1.Addressable{
							URL: &apis.URL{
								Scheme: "http",
								Host:   "test-domain",
							},
						},
						Hostname: "test-domain",
					},
				},
				Status: duckv1beta1.Status{
					Conditions: []apis.Condition{{
						Type:   AZServicebusChannelConditionAddressable,
						Status: corev1.ConditionTrue,
					}, {
						// Ready unknown comes from other dependent conditions via MarkTrue.
						Type:   AZServicebusChannelConditionReady,
						Status: corev1.ConditionUnknown,
					}},
				},
			},
		},
	}
	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {
			cs := &AZServicebusChannelStatus{}
			cs.SetAddress(tc.url)
			if diff := cmp.Diff(tc.want, cs, ignoreAllButTypeAndStatus); diff != "" {
				t.Errorf("unexpected conditions (-want, +got) = %v", diff)
			}
		})
	}
}
