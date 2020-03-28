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

package controller

import (
	"context"
	"fmt"
	"testing"

	clientgotesting "k8s.io/client-go/testing"
	"knative.dev/eventing/pkg/reconciler"
	"knative.dev/eventing/pkg/utils"

	duckv1alpha1 "knative.dev/pkg/apis/duck/v1alpha1"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/kmeta"
	logtesting "knative.dev/pkg/logging/testing"
	. "knative.dev/pkg/reconciler/testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"

	"knative.dev/eventing-contrib/azsb/channel/pkg/apis/messaging/v1alpha1"
	fakeclientset "knative.dev/eventing-contrib/azsb/channel/pkg/client/injection/client/fake"
	"knative.dev/eventing-contrib/azsb/channel/pkg/reconciler/controller/resources"
	reconciletesting "knative.dev/eventing-contrib/azsb/channel/pkg/reconciler/testing"

	bus "github.com/Azure/azure-service-bus-go"
)

const (
	systemNS              = "knative-eventing"
	testNS                = "test-namespace"
	azcName               = "test-azsb"
	dispatcherDeployName  = "test-deployment"
	dispatcherServName    = "test-service"
	channelServiceAddress = "test-azsb-kn-channel.test-namespace.svc.cluster.local"
)

var (
	trueVal = true
	// deletionTime is used when objects are marked as deleted. Rfc3339Copy()
	// truncates to seconds to match the loss of precision during serialization.
	deletionTime = metav1.Now().Rfc3339Copy()
)

func init() {
	// Add types to scheme
	_ = v1alpha1.AddToScheme(scheme.Scheme)
	_ = duckv1alpha1.AddToScheme(scheme.Scheme)
}

func TestAllCases(t *testing.T) {
	ncKey := testNS + "/" + azcName
	table := TableTest{
		{
			Name: "bad workqueue key",
			// Make sure Reconcile handles bad keys.
			Key: "too/many/parts",
		},
		{
			Name: "key not found",
			// Make sure Reconcile handles good keys that don't exist.
			Key: "foo/not-found",
		},
		{
			Name: "deleting",
			Key:  ncKey,
			Objects: []runtime.Object{
				reconciletesting.NewAZServicebusChannel(azcName, testNS,
					reconciletesting.WithAzsbInitChannelConditions,
					reconciletesting.WithAZServicebusChannelDeleted)},
			WantErr: false,
			WantEvents: []string{
				Eventf(corev1.EventTypeNormal, channelReconciled, "AZServicebusChannel reconciled"),
			},
		},
		{
			Name: "deployment does not exist",
			Key:  ncKey,
			Objects: []runtime.Object{
				reconciletesting.NewAZServicebusChannel(azcName, testNS),
			},
			WantErr: true,
			WantStatusUpdates: []clientgotesting.UpdateActionImpl{{
				Object: reconciletesting.NewAZServicebusChannel(azcName, testNS,
					reconciletesting.WithAzsbInitChannelConditions,
					reconciletesting.WithAZServicebusChannelTopicReady(),
					reconciletesting.WithAzServicebusChannelConfigReady(),
					reconciletesting.WithAZServicebusChannelDeploymentNotReady("DispatcherDeploymentDoesNotExist", "Dispatcher Deployment does not exist")),
			}},
			WantPatches: []clientgotesting.PatchActionImpl{
				patchFinalizers(testNS, azcName),
			},
			WantEvents: []string{
				Eventf(corev1.EventTypeWarning, channelReconcileFailed, "AZServicebusChannel reconciliation failed: deployment.apps \"test-deployment\" not found"),
			},
		},
		{
			Name: "Service does not exist",
			Key:  ncKey,
			Objects: []runtime.Object{
				makeReadyDeployment(),
				reconciletesting.NewAZServicebusChannel(azcName, testNS),
			},
			WantErr: true,
			WantStatusUpdates: []clientgotesting.UpdateActionImpl{{
				Object: reconciletesting.NewAZServicebusChannel(azcName, testNS,
					reconciletesting.WithAzsbInitChannelConditions,
					reconciletesting.WithAZServicebusChannelDeploymentReady(),
					reconciletesting.WithAZServicebusChannelTopicReady(),
					reconciletesting.WithAzServicebusChannelConfigReady(),
					reconciletesting.WithAZServicebusChannelServiceNotReady("DispatcherServiceDoesNotExist", "Dispatcher Service does not exist")),
			}},
			WantPatches: []clientgotesting.PatchActionImpl{
				patchFinalizers(testNS, azcName),
			},
			WantEvents: []string{
				Eventf(corev1.EventTypeWarning, channelReconcileFailed, "AZServicebusChannel reconciliation failed: service \"test-service\" not found"),
			},
		},
		{
			Name: "Endpoints do not exist",
			Key:  ncKey,
			Objects: []runtime.Object{
				makeReadyDeployment(),
				makeService(),
				reconciletesting.NewAZServicebusChannel(azcName, testNS),
			},
			WantErr: true,
			WantStatusUpdates: []clientgotesting.UpdateActionImpl{{
				Object: reconciletesting.NewAZServicebusChannel(azcName, testNS,
					reconciletesting.WithAzsbInitChannelConditions,
					reconciletesting.WithAZServicebusChannelDeploymentReady(),
					reconciletesting.WithAZServicebusChannelServiceReady(),
					reconciletesting.WithAZServicebusChannelTopicReady(),
					reconciletesting.WithAzServicebusChannelConfigReady(),
					reconciletesting.WithAZServicebusChannelEndpointsNotReady("DispatcherEndpointsDoesNotExist", "Dispatcher Endpoints does not exist"),
				),
			}},
			WantPatches: []clientgotesting.PatchActionImpl{
				patchFinalizers(testNS, azcName),
			},
			WantEvents: []string{
				Eventf(corev1.EventTypeWarning, channelReconcileFailed, "AZServicebusChannel reconciliation failed: endpoints \"test-service\" not found"),
			},
		},
		{
			Name: "Endpoints not ready",
			Key:  ncKey,
			Objects: []runtime.Object{
				makeReadyDeployment(),
				makeService(),
				makeEmptyEndpoints(),
				reconciletesting.NewAZServicebusChannel(azcName, testNS),
			},
			WantErr: true,
			WantStatusUpdates: []clientgotesting.UpdateActionImpl{{
				Object: reconciletesting.NewAZServicebusChannel(azcName, testNS,
					reconciletesting.WithAzsbInitChannelConditions,
					reconciletesting.WithAZServicebusChannelDeploymentReady(),
					reconciletesting.WithAZServicebusChannelServiceReady(),
					reconciletesting.WithAZServicebusChannelTopicReady(),
					reconciletesting.WithAzServicebusChannelConfigReady(),
					reconciletesting.WithAZServicebusChannelEndpointsNotReady("DispatcherEndpointsNotReady", "There are no endpoints ready for Dispatcher service"),
				),
			}},
			WantPatches: []clientgotesting.PatchActionImpl{
				patchFinalizers(testNS, azcName),
			},
			WantEvents: []string{
				Eventf(corev1.EventTypeWarning, channelReconcileFailed, "AZServicebusChannel reconciliation failed: there are no endpoints ready for Dispatcher service test-service"),
			},
		},
		{
			Name: "Works, creates new channel",
			Key:  ncKey,
			Objects: []runtime.Object{
				makeReadyDeployment(),
				makeService(),
				makeReadyEndpoints(),
				reconciletesting.NewAZServicebusChannel(azcName, testNS),
			},
			WantErr: false,
			WantCreates: []runtime.Object{
				makeChannelService(reconciletesting.NewAZServicebusChannel(azcName, testNS)),
			},
			WantStatusUpdates: []clientgotesting.UpdateActionImpl{{
				Object: reconciletesting.NewAZServicebusChannel(azcName, testNS,
					reconciletesting.WithAzsbInitChannelConditions,
					reconciletesting.WithAZServicebusChannelDeploymentReady(),
					reconciletesting.WithAZServicebusChannelServiceReady(),
					reconciletesting.WithAZServicebusChannelEndpointsReady(),
					reconciletesting.WithAZServicebusChannelChannelServiceReady(),
					reconciletesting.WithAZServicebusChannelTopicReady(),
					reconciletesting.WithAzServicebusChannelConfigReady(),
					reconciletesting.WithAZServicebusChannelAddress(channelServiceAddress),
				),
			}},
			WantPatches: []clientgotesting.PatchActionImpl{
				patchFinalizers(testNS, azcName),
			},
			WantEvents: []string{
				Eventf(corev1.EventTypeNormal, channelReconciled, "AZServicebusChannel reconciled"),
			},
		},
		{
			Name: "Works, channel exists",
			Key:  ncKey,
			Objects: []runtime.Object{
				makeReadyDeployment(),
				makeService(),
				makeReadyEndpoints(),
				reconciletesting.NewAZServicebusChannel(azcName, testNS),
				makeChannelService(reconciletesting.NewAZServicebusChannel(azcName, testNS)),
			},
			WantErr: false,
			WantStatusUpdates: []clientgotesting.UpdateActionImpl{{
				Object: reconciletesting.NewAZServicebusChannel(azcName, testNS,
					reconciletesting.WithAzsbInitChannelConditions,
					reconciletesting.WithAZServicebusChannelDeploymentReady(),
					reconciletesting.WithAZServicebusChannelServiceReady(),
					reconciletesting.WithAZServicebusChannelEndpointsReady(),
					reconciletesting.WithAZServicebusChannelChannelServiceReady(),
					reconciletesting.WithAZServicebusChannelTopicReady(),
					reconciletesting.WithAzServicebusChannelConfigReady(),
					reconciletesting.WithAZServicebusChannelAddress(channelServiceAddress),
				),
			}},
			WantPatches: []clientgotesting.PatchActionImpl{
				patchFinalizers(testNS, azcName),
			},
			WantEvents: []string{
				Eventf(corev1.EventTypeNormal, channelReconciled, "AZServicebusChannel reconciled"),
			},
		},
		{
			Name: "channel exists, not owned by us",
			Key:  ncKey,
			Objects: []runtime.Object{
				makeReadyDeployment(),
				makeService(),
				makeReadyEndpoints(),
				reconciletesting.NewAZServicebusChannel(azcName, testNS),
				makeChannelServiceNotOwnedByUs(reconciletesting.NewAZServicebusChannel(azcName, testNS)),
			},
			WantErr: true,
			WantStatusUpdates: []clientgotesting.UpdateActionImpl{{
				Object: reconciletesting.NewAZServicebusChannel(azcName, testNS,
					reconciletesting.WithAzsbInitChannelConditions,
					reconciletesting.WithAZServicebusChannelDeploymentReady(),
					reconciletesting.WithAZServicebusChannelServiceReady(),
					reconciletesting.WithAZServicebusChannelEndpointsReady(),
					reconciletesting.WithAZServicebusChannelChannelServiceReady(),
					reconciletesting.WithAZServicebusChannelTopicReady(),
					reconciletesting.WithAzServicebusChannelConfigReady(),
					reconciletesting.WithAZServicebusChannelChannelServicetNotReady("ChannelServiceFailed", "Channel Service failed: AZServicebusChannel: test-namespace/test-azsb does not own Service: \"test-azsb-kn-channel\""),
				),
			}},
			WantPatches: []clientgotesting.PatchActionImpl{
				patchFinalizers(testNS, azcName),
			},
			WantEvents: []string{
				Eventf(corev1.EventTypeWarning, "ChannelReconcileFailed", "AZServicebusChannel reconciliation failed: AZServicebusChannel: test-namespace/test-azsb does not own Service: \"test-azsb-kn-channel\""),
			},
		},
		{
			Name: "channel does not exist, fails to create",
			Key:  ncKey,
			Objects: []runtime.Object{
				makeReadyDeployment(),
				makeService(),
				makeReadyEndpoints(),
				reconciletesting.NewAZServicebusChannel(azcName, testNS),
			},
			WantErr: true,
			WithReactors: []clientgotesting.ReactionFunc{
				InduceFailure("create", "Services"),
			},
			WantStatusUpdates: []clientgotesting.UpdateActionImpl{{
				Object: reconciletesting.NewAZServicebusChannel(azcName, testNS,
					reconciletesting.WithAzsbInitChannelConditions,
					reconciletesting.WithAZServicebusChannelDeploymentReady(),
					reconciletesting.WithAZServicebusChannelServiceReady(),
					reconciletesting.WithAZServicebusChannelEndpointsReady(),
					reconciletesting.WithAZServicebusChannelChannelServiceReady(),
					reconciletesting.WithAZServicebusChannelTopicReady(),
					reconciletesting.WithAzServicebusChannelConfigReady(),
					reconciletesting.WithAZServicebusChannelChannelServicetNotReady("ChannelServiceFailed", "Channel Service failed: inducing failure for create services"),
				),
			}},
			WantPatches: []clientgotesting.PatchActionImpl{
				patchFinalizers(testNS, azcName),
			},
			WantCreates: []runtime.Object{
				makeChannelService(reconciletesting.NewAZServicebusChannel(azcName, testNS)),
			},
			WantEvents: []string{
				Eventf(corev1.EventTypeWarning, channelReconcileFailed, "AZServicebusChannel reconciliation failed: inducing failure for create services"),
			},
		},
	}
	defer logtesting.ClearAll()

	table.Test(t, reconciletesting.MakeFactoryWithContext(func(ctx context.Context, listers *reconciletesting.Listers) controller.Reconciler {
		ce := "Endpoint=sb://example.servicebus.windows.net/;SharedAccessKeyName=ExamplePolicy;SharedAccessKey=XXXXXXXXXXXXXXXXXXXXXXXXXXXXXX="
		ns, _ := bus.NewNamespace(bus.NamespaceWithConnectionString(ce))
		return &Reconciler{
			Base:                     reconciler.NewBase(ctx, controllerAgentName, configmap.NewStaticWatcher()),
			dispatcherNamespace:      testNS,
			dispatcherDeploymentName: dispatcherDeployName,
			dispatcherServiceName:    dispatcherServName,
			azsbchannelLister:        listers.GetAzsbChannelLister(),
			azServicebusClientSet:    fakeclientset.Get(ctx),
			// TODO fix
			azsbchannelInformer: nil,
			deploymentLister:    listers.GetDeploymentLister(),
			serviceLister:       listers.GetServiceLister(),
			endpointsLister:     listers.GetEndpointsLister(),
			azServicebusManager: ns,
		}
	}))
}

func makeDeployment() *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNS,
			Name:      dispatcherDeployName,
		},
		Status: appsv1.DeploymentStatus{},
	}
}

func makeReadyDeployment() *appsv1.Deployment {
	d := makeDeployment()
	d.Status.Conditions = []appsv1.DeploymentCondition{{Type: appsv1.DeploymentAvailable, Status: corev1.ConditionTrue}}
	return d
}

func makeService() *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNS,
			Name:      dispatcherServName,
		},
	}
}

func makeChannelService(nc *v1alpha1.AZServicebusChannel) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNS,
			Name:      fmt.Sprintf("%s-kn-channel", azcName),
			Labels: map[string]string{
				resources.MessagingRoleLabel: resources.MessagingRole,
			},
			OwnerReferences: []metav1.OwnerReference{
				*kmeta.NewControllerRef(nc),
			},
		},
		Spec: corev1.ServiceSpec{
			Type:         corev1.ServiceTypeExternalName,
			ExternalName: fmt.Sprintf("%s.%s.svc.%s", dispatcherServName, testNS, utils.GetClusterDomainName()),
		},
	}
}

func makeChannelServiceNotOwnedByUs(nc *v1alpha1.AZServicebusChannel) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNS,
			Name:      fmt.Sprintf("%s-kn-channel", azcName),
			Labels: map[string]string{
				resources.MessagingRoleLabel: resources.MessagingRole,
			},
		},
		Spec: corev1.ServiceSpec{
			Type:         corev1.ServiceTypeExternalName,
			ExternalName: fmt.Sprintf("%s.%s.svc.%s", dispatcherServName, testNS, utils.GetClusterDomainName()),
		},
	}
}

func makeEmptyEndpoints() *corev1.Endpoints {
	return &corev1.Endpoints{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Endpoints",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNS,
			Name:      dispatcherServName,
		},
	}
}

func makeReadyEndpoints() *corev1.Endpoints {
	e := makeEmptyEndpoints()
	e.Subsets = []corev1.EndpointSubset{{Addresses: []corev1.EndpointAddress{{IP: "1.1.1.1"}}}}
	return e
}

func patchFinalizers(namespace, name string) clientgotesting.PatchActionImpl {
	action := clientgotesting.PatchActionImpl{}
	action.Name = name
	action.Namespace = namespace
	patch := `{"metadata":{"finalizers":["` + finalizerName + `"],"resourceVersion":""}}`
	action.Patch = []byte(patch)
	return action
}

func patchRemoveFinalizers(namespace, name string) clientgotesting.PatchActionImpl {
	action := clientgotesting.PatchActionImpl{}
	action.Name = name
	action.Namespace = namespace
	patch := `{"metadata":{"finalizers":[],"resourceVersion":""}}`
	action.Patch = []byte(patch)
	return action
}
