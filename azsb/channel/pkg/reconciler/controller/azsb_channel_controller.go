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
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	azsbus "github.com/Azure/azure-service-bus-go"
	"github.com/kelseyhightower/envconfig"
	"knative.dev/eventing-contrib/azsb/channel/pkg/apis/messaging/v1alpha1"
	azsbClientset "knative.dev/eventing-contrib/azsb/channel/pkg/client/clientset/versioned"
	azsbScheme "knative.dev/eventing-contrib/azsb/channel/pkg/client/clientset/versioned/scheme"
	azservicebuschannelinjection "knative.dev/eventing-contrib/azsb/channel/pkg/client/injection/client"

	"knative.dev/eventing-contrib/azsb/channel/pkg/client/injection/informers/messaging/v1alpha1/azservicebuschannel"
	listers "knative.dev/eventing-contrib/azsb/channel/pkg/client/listers/messaging/v1alpha1"
	"knative.dev/eventing-contrib/azsb/channel/pkg/reconciler"
	"knative.dev/eventing-contrib/azsb/channel/pkg/reconciler/controller/resources"

	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
	"knative.dev/eventing/pkg/reconciler/names"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/client/injection/kube/informers/apps/v1/deployment"
	"knative.dev/pkg/client/injection/kube/informers/core/v1/endpoints"
	"knative.dev/pkg/client/injection/kube/informers/core/v1/secret"
	"knative.dev/pkg/client/injection/kube/informers/core/v1/service"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/system"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	appsv1listers "k8s.io/client-go/listers/apps/v1"
	corev1listers "k8s.io/client-go/listers/core/v1"
)

const (
	// ReconcilerName is the name of the reconciler.
	ReconcilerName = "AZServicebusChannels"

	// controllerAgentName is the string used by this controller to identify
	// itself when creating events.
	controllerAgentName = "az-servicebus-ch-controller"

	finalizerName = controllerAgentName

	// Name of the corev1.Events emitted from the reconciliation process.
	channelReconciled                = "ChannelReconciled"
	channelReconcileFailed           = "ChannelReconcileFailed"
	channelUpdateStatusFailed        = "ChannelUpdateStatusFailed"
	dispatcherDeploymentCreated      = "DispatcherDeploymentCreated"
	dispatcherDeploymentUpdated      = "DispatcherDeploymentUpdated"
	dispatcherDeploymentFailed       = "DispatcherDeploymentFailed"
	dispatcherDeploymentUpdateFailed = "DispatcherDeploymentUpdateFailed"
	dispatcherServiceCreated         = "DispatcherServiceCreated"
	dispatcherServiceFailed          = "DispatcherServiceFailed"

	dispatcherDeploymentName = "az-servicebus-ch-dispatcher"
	dispatcherServiceName    = "az-servicebus-ch-dispatcher"
)

func init() {
	// Add run types to the default Kubernetes Scheme so Events can be
	// logged for run types.
	_ = azsbScheme.AddToScheme(scheme.Scheme)
}

// Reconciler reconciles AZ Service Bus Channels.
type Reconciler struct {
	*reconciler.Base

	topicMux                 sync.Mutex
	dispatcherNamespace      string
	dispatcherDeploymentName string
	dispatcherServiceName    string

	azServicebusManager *azsbus.Namespace

	azServicebusClientSet azsbClientset.Interface
	azsbchannelLister     listers.AZServicebusChannelLister
	azsbchannelInformer   cache.SharedIndexInformer
	deploymentLister      appsv1listers.DeploymentLister
	serviceLister         corev1listers.ServiceLister
	endpointsLister       corev1listers.EndpointsLister
	secretsLister         corev1listers.SecretLister
	impl                  *controller.Impl
}

var (
	deploymentGVK = appsv1.SchemeGroupVersion.WithKind("Deployment")
	serviceGVK    = corev1.SchemeGroupVersion.WithKind("Service")
)

type envConfig struct {
	MetricsDomain   string `envconfig:"METRICS_DOMAIN" required:"true"`
	SystemNamespace string `envconfig:"SYSTEM_NAMESPACE" required:"true"`
}

// Check that our Reconciler implements controller.Reconciler.
var _ controller.Reconciler = (*Reconciler)(nil)

// Check that our Reconciler implements cache.ResourceEventHandler
var _ cache.ResourceEventHandler = (*Reconciler)(nil)

// NewController initializes the controller and is called by the generated code.
// Registers event handlers to enqueue events.
func NewController(
	ctx context.Context,
	cmw configmap.Watcher,
) *controller.Impl {

	dispatcherNamespace := system.Namespace()
	azServicebusChannelInformer := azservicebuschannel.Get(ctx)
	deploymentInformer := deployment.Get(ctx)
	endpointsInformer := endpoints.Get(ctx)
	serviceInformer := service.Get(ctx)
	secretInformer := secret.Get(ctx)

	azsbChannelClientSet := azservicebuschannelinjection.Get(ctx)

	r := &Reconciler{
		Base:                     reconciler.NewBase(ctx, controllerAgentName, cmw),
		dispatcherNamespace:      dispatcherNamespace,
		dispatcherDeploymentName: dispatcherDeploymentName,
		dispatcherServiceName:    dispatcherServiceName,
		azsbchannelLister:        azServicebusChannelInformer.Lister(),
		azsbchannelInformer:      azServicebusChannelInformer.Informer(),
		deploymentLister:         deploymentInformer.Lister(),
		serviceLister:            serviceInformer.Lister(),
		endpointsLister:          endpointsInformer.Lister(),
		secretsLister:            secretInformer.Lister(),
		azServicebusClientSet:    azsbChannelClientSet,
	}
	r.impl = controller.NewImpl(r, r.Logger, ReconcilerName)
	r.Logger.Info("Setting up event handlers")

	env := &envConfig{}
	if err := envconfig.Process("", env); err != nil {
		r.Logger.Panicf("unable to process AZ Service Bus channel's required environment variables: %v", err)
	}

	r.impl = controller.NewImpl(r, r.Logger, ReconcilerName)

	r.Logger.Info("Setting up event handlers")
	azServicebusChannelInformer.Informer().AddEventHandler(controller.HandleAll(r.impl.Enqueue))

	// Set up watches for dispatcher resources we care about, since any changes to these
	// resources will affect our Channels. So, set up a watch here, that will cause
	// a global Resync for all the channels to take stock of their health when these change.
	deploymentInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: controller.FilterWithNameAndNamespace(dispatcherNamespace, dispatcherDeploymentName),
		Handler:    r,
	})
	serviceInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: controller.FilterWithNameAndNamespace(dispatcherNamespace, dispatcherServiceName),
		Handler:    r,
	})
	secretInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: controller.FilterWithNameAndNamespace(dispatcherNamespace, dispatcherServiceName),
		Handler:    r,
	})
	endpointsInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: controller.FilterWithNameAndNamespace(dispatcherNamespace, dispatcherServiceName),
		Handler:    r,
	})
	return r.impl
}

// OnAdd cache.ResourceEventHandler implementation.
// These 3 functions just cause a Global Resync of the channels, because any changes here
// should be reflected onto the channels.
func (r *Reconciler) OnAdd(obj interface{}) {
	r.impl.GlobalResync(r.azsbchannelInformer)
}

// OnUpdate method of cache.ResourceEventHandler implementation.
func (r *Reconciler) OnUpdate(old, new interface{}) {
	r.impl.GlobalResync(r.azsbchannelInformer)
}

// OnDelete method of cache.ResourceEventHandler implementation.
func (r *Reconciler) OnDelete(obj interface{}) {
	r.impl.GlobalResync(r.azsbchannelInformer)
}

// Reconcile compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the AZServicebusChannel resource
// with the current status of the resource.
func (r *Reconciler) Reconcile(ctx context.Context, key string) error {
	// Convert the namespace/name string into a distinct namespace and name.
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		logging.FromContext(ctx).Error("invalid resource key")
		return nil
	}
	// Get the AZServicebusChannel resource with this namespace/name.
	original, err := r.azsbchannelLister.AZServicebusChannels(namespace).Get(name)
	if apierrs.IsNotFound(err) {
		// The resource may no longer exist, in which case we stop processing.
		logging.FromContext(ctx).Error("AZ Service Bus key in work queue no longer exists")
		return nil
	} else if err != nil {
		return err
	}

	// Don't modify the informers copy.
	channel := original.DeepCopy()

	// Reconcile this copy of the AZServicebusChannel and then write back any status updates regardless of
	// whether the reconcile error out.
	reconcileErr := r.reconcile(ctx, channel)
	if reconcileErr != nil {
		logging.FromContext(ctx).Error("Error reconciling AZServicebusChannel", zap.Error(reconcileErr))
		r.Recorder.Eventf(channel, corev1.EventTypeWarning, channelReconcileFailed, "AZServicebusChannel reconciliation failed: %v", reconcileErr)
	} else {
		logging.FromContext(ctx).Debug("AZServicebusChannel reconciled")
		r.Recorder.Event(channel, corev1.EventTypeNormal, channelReconciled, "AZServicebusChannel reconciled")
	}

	if _, updateStatusErr := r.updateStatus(ctx, channel); updateStatusErr != nil {
		logging.FromContext(ctx).Error("Failed to update AZServicebusChannel status", zap.Error(updateStatusErr))
		r.Recorder.Eventf(channel, corev1.EventTypeWarning, channelUpdateStatusFailed, "Failed to update AZServicebusChannel's status: %v", updateStatusErr)
		return updateStatusErr
	}

	// Requeue if the resource is not ready
	return reconcileErr
}

func (r *Reconciler) reconcile(ctx context.Context, azc *v1alpha1.AZServicebusChannel) error {
	azc.Status.InitializeConditions()
	logger := logging.FromContext(ctx)
	// Verify channel is valid.
	azc.SetDefaults(ctx)
	if err := azc.Validate(ctx); err != nil {
		logger.Error("Invalid az service bus channel", zap.String("channel", azc.Name), zap.Error(err))
		return err
	}

	// See if the channel has been deleted.
	if azc.DeletionTimestamp != nil {
		if azsbManager, err := r.createClient(ctx, azc); err == nil {
			if err := r.deleteTopic(ctx, azc, azsbManager); err != nil {
				return err
			}
		}
		removeFinalizer(azc)
		_, err := r.azServicebusClientSet.MessagingV1alpha1().AZServicebusChannels(azc.Namespace).Update(azc)
		return err
	}

	// If we are adding the finalizer for the first time, then ensure that finalizer is persisted
	// before manipulating Azure Service Bus.
	if err := r.ensureFinalizer(azc); err != nil {
		return err
	}

	// TODO: This is where i'll need to look into the secrets
	// if r.kafkaConfig == nil {
	// 	if r.kafkaConfigError == nil {
	// 		r.kafkaConfigError = errors.New("The config map 'config-kafka' does not exist")
	// 	}
	// 	kc.Status.MarkConfigFailed("MissingConfiguration", "%v", r.kafkaConfigError)
	// 	return r.kafkaConfigError
	// }
	// Create the AZ Service Bus Client
	azsbManager, err := r.createClient(ctx, azc)
	if err != nil {
		azc.Status.MarkConfigFailed("InvalidConfiguration", "Unable to build AZ Client admin client for channel %s: %v", azc.Name, err)
	}

	azc.Status.MarkConfigTrue()

	// We reconcile the status of the Channel by looking at:
	// 1. Azure Service Bus topic used by the channel.
	// 2. Dispatcher Deployment for it's readiness.
	// 3. Dispatcher k8s Service for it's existence.
	// 4. Dispatcher endpoints to ensure that there's something backing the Service.
	// 5. K8s service representing the channel that will use ExternalName to point to the Dispatcher k8s service.

	// TOPIC: This is where I go through the topic create logic
	if err := r.createTopic(ctx, azc, azsbManager); err != nil {
		azc.Status.MarkTopicFailed("TopicCreateFailed", "error while creating topic: %s", err)
		return err
	}
	azc.Status.MarkTopicTrue()

	// Get the Dispatcher Deployment and propagate the status to the Channel
	d, err := r.deploymentLister.Deployments(r.dispatcherNamespace).Get(r.dispatcherDeploymentName)
	if err != nil {
		if apierrs.IsNotFound(err) {
			azc.Status.MarkDispatcherFailed("DispatcherDeploymentDoesNotExist", "Dispatcher Deployment does not exist")
		} else {
			logger.Error("Unable to get the dispatcher Deployment", zap.Error(err))
			azc.Status.MarkDispatcherFailed("DispatcherDeploymentGetFailed", "Failed to get dispatcher Deployment")
		}
		return err
	}
	azc.Status.PropagateDispatcherStatus(&d.Status)

	// Get the Dispatcher Service and propagate the status to the Channel in case it does not exist.
	// We don't do anything with the service because it's status contains nothing useful, so just do
	// an existence check. Then below we check the endpoints targeting it.
	_, err = r.serviceLister.Services(r.dispatcherNamespace).Get(r.dispatcherServiceName)
	if err != nil {
		if apierrs.IsNotFound(err) {
			azc.Status.MarkServiceFailed("DispatcherServiceDoesNotExist", "Dispatcher Service does not exist")
		} else {
			logger.Error("Unable to get the dispatcher service", zap.Error(err))
			azc.Status.MarkServiceFailed("DispatcherServiceGetFailed", "Failed to get dispatcher service")
		}
		return err
	}
	azc.Status.MarkServiceTrue()

	// Get the Dispatcher Service Endpoints and propagate the status to the Channel
	// endpoints has the same name as the service, so not a bug.
	e, err := r.endpointsLister.Endpoints(r.dispatcherNamespace).Get(r.dispatcherServiceName)
	if err != nil {
		if apierrs.IsNotFound(err) {
			azc.Status.MarkEndpointsFailed("DispatcherEndpointsDoesNotExist", "Dispatcher Endpoints does not exist")
		} else {
			logger.Error("Unable to get the dispatcher endpoints", zap.Error(err))
			azc.Status.MarkEndpointsFailed("DispatcherEndpointsGetFailed", "Failed to get dispatcher endpoints")
		}
		return err
	}

	if len(e.Subsets) == 0 {
		logger.Error("No endpoints found for Dispatcher service", zap.Error(err))
		azc.Status.MarkEndpointsFailed("DispatcherEndpointsNotReady", "There are no endpoints ready for Dispatcher service")
		return fmt.Errorf("there are no endpoints ready for Dispatcher service %s", r.dispatcherServiceName)
	}
	azc.Status.MarkEndpointsTrue()

	// Reconcile the k8s service representing the actual Channel. It points to the Dispatcher service via ExternalName
	svc, err := r.reconcileChannelService(ctx, azc)
	if err != nil {
		azc.Status.MarkChannelServiceFailed("ChannelServiceFailed", fmt.Sprintf("Channel Service failed: %s", err))
		return err
	}
	azc.Status.MarkChannelServiceTrue()
	azc.Status.SetAddress(&apis.URL{
		Scheme: "http",
		Host:   names.ServiceHostName(svc.Name, svc.Namespace),
	})

	// close the connection
	// err = azsbManager.
	// if err != nil {
	// 	logger.Error("Error closing the connection", zap.Error(err))
	// 	return err
	// }

	// Ok, so now the Dispatcher Deployment & Service have been created, we're golden since the
	// dispatcher watches the Channel and where it needs to dispatch events to.
	return nil
}

func (r *Reconciler) updateStatus(ctx context.Context, desired *v1alpha1.AZServicebusChannel) (*v1alpha1.AZServicebusChannel, error) {
	azc, err := r.azsbchannelLister.AZServicebusChannels(desired.Namespace).Get(desired.Name)
	if err != nil {
		return nil, err
	}

	if reflect.DeepEqual(azc.Status, desired.Status) {
		return azc, nil
	}

	becomesReady := desired.Status.IsReady() && !azc.Status.IsReady()

	// Don't modify the informers copy.
	existing := azc.DeepCopy()
	existing.Status = desired.Status

	new, err := r.azServicebusClientSet.MessagingV1alpha1().AZServicebusChannels(desired.Namespace).UpdateStatus(existing)
	if err == nil && becomesReady {
		duration := time.Since(new.ObjectMeta.CreationTimestamp.Time)
		r.Logger.Infof("AZServicebusChannel %q became ready after %v", azc.Name, duration)
		if err := r.StatsReporter.ReportReady("AZServicebusChannel", azc.Namespace, azc.Name, duration); err != nil {
			r.Logger.Infof("Failed to record ready for AZServicebusChannel %q: %v", azc.Name, err)
		}
	}
	return new, err
}

func (r *Reconciler) reconcileDispatcher(ctx context.Context, dispatcherNamespace string, azc *v1alpha1.AZServicebusChannel) (*appsv1.Deployment, error) {
	args := resources.DispatcherArgs{
		DispatcherNamespace: dispatcherNamespace,
	}
	expected := resources.MakeDispatcher(args)
	d, err := r.deploymentLister.Deployments(dispatcherNamespace).Get(dispatcherDeploymentName)
	if err != nil {
		if apierrs.IsNotFound(err) {
			d, err := r.KubeClientSet.AppsV1().Deployments(dispatcherNamespace).Create(expected)
			if err == nil {
				r.Recorder.Event(azc, corev1.EventTypeNormal, dispatcherDeploymentCreated, "Dispatcher deployment created")
				azc.Status.PropagateDispatcherStatus(&d.Status)
			} else {
				logging.FromContext(ctx).Error("Unable to create the dispatcher deployment", zap.Error(err))
				r.Recorder.Eventf(azc, corev1.EventTypeWarning, dispatcherDeploymentFailed, "Failed to create the dispatcher deployment: %v", err)
				azc.Status.MarkServiceFailed(dispatcherDeploymentFailed, "Failed to create the dispatcher deployment: %v", err)
			}
			return d, err
		}

		logging.FromContext(ctx).Error("Unable to get the dispatcher deployment", zap.Error(err))
		azc.Status.MarkServiceUnknown("DispatcherDeploymentFailed", "Failed to get dispatcher deployment: %v", err)
		return nil, err
	} else if !reflect.DeepEqual(expected.Spec, d.Spec) {
		logging.FromContext(ctx).Info("Deployment is not what we expect it to be, updating Deployment")
		d, err := r.KubeClientSet.AppsV1().Deployments(dispatcherNamespace).Update(expected)
		if err == nil {
			r.Recorder.Event(azc, corev1.EventTypeNormal, dispatcherDeploymentUpdated, "Dispatcher deployment updated")
			azc.Status.PropagateDispatcherStatus(&d.Status)
		} else {
			logging.FromContext(ctx).Error("Unable to update the dispatcher deployment", zap.Error(err))
			r.Recorder.Eventf(azc, corev1.EventTypeWarning, dispatcherDeploymentUpdateFailed, "Failed to update the dispatcher deployment: %v", err)
			azc.Status.MarkServiceFailed("DispatcherDeploymentUpdateFailed", "Failed to update the dispatcher deployment: %v", err)
		}
		return d, err
	}

	azc.Status.PropagateDispatcherStatus(&d.Status)
	return d, nil
}

func (r *Reconciler) reconcileDispatcherService(ctx context.Context, dispatcherNamespace string, azc *v1alpha1.AZServicebusChannel) (*corev1.Service, error) {
	svc, err := r.serviceLister.Services(dispatcherNamespace).Get(dispatcherDeploymentName)
	if err != nil {
		if apierrs.IsNotFound(err) {
			expected := resources.MakeDispatcherService(dispatcherNamespace)
			svc, err := r.KubeClientSet.CoreV1().Services(dispatcherNamespace).Create(expected)

			if err == nil {
				r.Recorder.Event(azc, corev1.EventTypeNormal, dispatcherServiceCreated, "Dispatcher service created")
				azc.Status.MarkServiceTrue()
			} else {
				logging.FromContext(ctx).Error("Unable to create the dispatcher service", zap.Error(err))
				r.Recorder.Eventf(azc, corev1.EventTypeWarning, dispatcherServiceFailed, "Failed to create the dispatcher service: %v", err)
				azc.Status.MarkServiceFailed("DispatcherServiceFailed", "Failed to create the dispatcher service: %v", err)
			}

			return svc, err
		}

		logging.FromContext(ctx).Error("Unable to get the dispatcher service", zap.Error(err))
		azc.Status.MarkServiceUnknown("DispatcherServiceFailed", "Failed to get dispatcher service: %v", err)
		return nil, err
	}

	azc.Status.MarkServiceTrue()
	return svc, nil
}

func (r *Reconciler) reconcileChannelService(ctx context.Context, channel *v1alpha1.AZServicebusChannel) (*corev1.Service, error) {
	logger := logging.FromContext(ctx)
	// Get the  Service and propagate the status to the Channel in case it does not exist.
	// We don't do anything with the service because it's status contains nothing useful, so just do
	// an existence check. Then below we check the endpoints targeting it.
	// We may change this name later, so we have to ensure we use proper addressable when resolving these.
	svc, err := r.serviceLister.Services(channel.Namespace).Get(resources.MakeChannelServiceName(channel.Name))
	if err != nil {
		if apierrs.IsNotFound(err) {
			svc, err = resources.MakeK8sService(channel, resources.ExternalService(r.dispatcherNamespace, r.dispatcherServiceName))
			if err != nil {
				logger.Error("Failed to create the channel service object", zap.Error(err))
				return nil, err
			}
			svc, err = r.KubeClientSet.CoreV1().Services(channel.Namespace).Create(svc)
			if err != nil {
				logger.Error("simpleFailed to create the channel service", zap.Error(err))
				return nil, err
			}
			return svc, nil
		}
		logger.Error("Unable to get the channel service", zap.Error(err))
		return nil, err
	}
	// Check to make sure that the AZServicebusChannel owns this service and if not, complain.
	if !metav1.IsControlledBy(svc, channel) {
		return nil, fmt.Errorf("azservicebuschannel: %s/%s does not own Service: %q", channel.Namespace, channel.Name, svc.Name)
	}
	return svc, nil
}

func (r *Reconciler) createClient(ctx context.Context, azsbChannel *v1alpha1.AZServicebusChannel) (*azsbus.Namespace, error) {
	azServicebusManager := r.azServicebusManager
	if azServicebusManager == nil {
		var err error
		// TODO: somehow send in the proper secrets to connect to client
		azServicebusManager, err = resources.MakeClient()
		if err != nil {
			return nil, err
		}
	}
	return azServicebusManager, nil
}

func (r *Reconciler) createTopic(ctx context.Context, channel *v1alpha1.AZServicebusChannel, azServicebusManager *azsbus.Namespace) error {
	r.topicMux.Lock()
	defer r.topicMux.Unlock()
	logger := logging.FromContext(ctx)
	// topicName := utils.TopicName(utils.separator, channel.Namespace, channel.Name)
	topicName := channel.Name
	logger.Info("Creating topic on AZ Service Bus cluster", zap.String("topic", topicName))
	tm := azServicebusManager.NewTopicManager()

	te, err := tm.Get(ctx, topicName)

	if err != nil && !strings.ContainsAny(err.Error(), "not found") {
		logger.Warn("Error retrieving topic", zap.String("topic", topicName), zap.Error(err))
	} else if te != nil {
		logger.Warn("Topic already exists created topic", zap.String("topic", topicName))
	} else {
		_, err := tm.Put(ctx, topicName)
		if err != nil {
			logger.Error("Error creating topic", zap.String("topic", topicName), zap.Error(err))
		} else {
			logger.Info("Successfully created topic", zap.String("topic", topicName))
		}
	}

	return nil
}

func (r *Reconciler) deleteTopic(ctx context.Context, channel *v1alpha1.AZServicebusChannel, azServicebusManager *azsbus.Namespace) error {
	r.topicMux.Lock()
	defer r.topicMux.Unlock()
	logger := logging.FromContext(ctx)
	// TODO: change this to be topic name
	topicName := channel.Name
	logger.Info("Deleting topic on Azure Service Bus", zap.String("topic", topicName))
	tm := azServicebusManager.NewTopicManager()
	err := tm.Delete(ctx, topicName)
	if err != nil {
		logger.Error("Error deleting topic", zap.String("topic", topicName), zap.Error(err))
	} else {
		logger.Info("Successfully deleted topic", zap.String("topic", topicName))
	}

	return nil
}

func removeFinalizer(channel *v1alpha1.AZServicebusChannel) {
	finalizers := sets.NewString(channel.Finalizers...)
	finalizers.Delete(finalizerName)
	channel.Finalizers = finalizers.List()
}

func (r *Reconciler) ensureFinalizer(channel *v1alpha1.AZServicebusChannel) error {
	finalizers := sets.NewString(channel.Finalizers...)
	if finalizers.Has(finalizerName) {
		return nil
	}

	mergePatch := map[string]interface{}{
		"metadata": map[string]interface{}{
			"finalizers":      append(channel.Finalizers, finalizerName),
			"resourceVersion": channel.ResourceVersion,
		},
	}

	patch, err := json.Marshal(mergePatch)
	if err != nil {
		return err
	}

	_, err = r.azServicebusClientSet.MessagingV1alpha1().AZServicebusChannels(channel.Namespace).Patch(channel.Name, types.MergePatchType, patch)
	return err
}
