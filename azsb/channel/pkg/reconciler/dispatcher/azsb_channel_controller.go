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
	"os"
	"reflect"

	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
	"knative.dev/eventing-contrib/azsb/channel/pkg/apis/messaging/v1alpha1"
	"knative.dev/eventing-contrib/azsb/channel/pkg/dispatcher"
	"knative.dev/eventing/pkg/logging"
	"knative.dev/eventing/pkg/reconciler"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"

	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	azsbclientset "knative.dev/eventing-contrib/azsb/channel/pkg/client/clientset/versioned"
	azsbScheme "knative.dev/eventing-contrib/azsb/channel/pkg/client/clientset/versioned/scheme"
	azservicebuschannelinjection "knative.dev/eventing-contrib/azsb/channel/pkg/client/injection/client"
	"knative.dev/eventing-contrib/azsb/channel/pkg/client/injection/informers/messaging/v1alpha1/azservicebuschannel"
	listers "knative.dev/eventing-contrib/azsb/channel/pkg/client/listers/messaging/v1alpha1"
	eventingduck "knative.dev/eventing/pkg/apis/duck/v1alpha1"
	messagingv1alpha1 "knative.dev/eventing/pkg/apis/messaging/v1alpha1"

	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	// ReconcilerName is the name of the reconciler.
	ReconcilerName = "AZServicebusChannels"

	// controllerAgentName is the string used by this controller to identify
	// itself when creating events.
	controllerAgentName = "az-servicebus-ch-dispatcher"

	finalizerName = controllerAgentName

	// Name of the corev1.Events emitted from the reconciliation process.
	channelReconciled         = "ChannelReconciled"
	channelReconcileFailed    = "ChannelReconcileFailed"
	channelUpdateStatusFailed = "ChannelUpdateStatusFailed"
)

// Reconciler reconciles AZ Service Bus Channels.
type Reconciler struct {
	*reconciler.Base

	azsbDispatcher dispatcher.AZServicebusDispatcher

	azsbClientSet       azsbclientset.Interface
	azsbchannelLister   listers.AZServicebusChannelLister
	azsbchannelInformer cache.SharedIndexInformer
	impl                *controller.Impl
}

// Check that our Reconciler implements controller.Reconciler.
var _ controller.Reconciler = (*Reconciler)(nil)

func init() {
	_ = azsbScheme.AddToScheme(scheme.Scheme)
}

// NewController controller
func NewController(
	ctx context.Context,
	cmw configmap.Watcher,
) *controller.Impl {
	logger := logging.FromContext(ctx)
	base := reconciler.NewBase(ctx, controllerAgentName, cmw)

	// TODO: add the real connection endpoint here
	connection := os.Getenv("SB_CONNECTION")
	azsbDispatcher, err := dispatcher.NewDispatcher(connection, logger)
	if err != nil {
		logger.Fatal("Unable to create dispatcher", zap.Error(err))
		return nil
	}
	azsbChannelInformer := azservicebuschannel.Get(ctx)
	logger.Info("Starting the AZ Service Bus dispatcher")

	r := &Reconciler{
		Base:                base,
		azsbDispatcher:      azsbDispatcher,
		azsbchannelLister:   azsbChannelInformer.Lister(),
		azsbchannelInformer: azsbChannelInformer.Informer(),
		azsbClientSet:       azservicebuschannelinjection.Get(ctx),
	}
	r.impl = controller.NewImpl(r, r.Logger, ReconcilerName)

	r.Logger.Info("Setting up event handlers")

	// Watch for az service bus channels.
	azsbChannelInformer.Informer().AddEventHandler(controller.HandleAll(r.impl.Enqueue))

	logger.Info("Starting dispatcher.")
	go func() {
		if err := azsbDispatcher.Start(ctx); err != nil {
			logger.Error("Cannot start dispatcher", zap.Error(err))
		}
	}()

	return r.impl
}

// Reconcile reconciler
func (r *Reconciler) Reconcile(ctx context.Context, key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		logging.FromContext(ctx).Error("invalid resource key")
		return nil
	}

	// Get the AZServicebusChannel resource with this namespace/name.
	original, err := r.azsbchannelLister.AZServicebusChannels(namespace).Get(name)
	if apierrs.IsNotFound(err) {
		// The resource may no longer exist, in which case we stop processing.
		logging.FromContext(ctx).Error("AZServicebusChannel key in work queue no longer exists")
		return nil
	} else if err != nil {
		return err
	}

	if !original.Status.IsReady() {
		return fmt.Errorf("Channel is not ready. Cannot configure and update subscriber status")
	}

	// Don't modify the informers copy.
	channel := original.DeepCopy()

	if channel.DeletionTimestamp != nil {
		logging.FromContext(ctx).Info("In the deletion loop, debugging for finalizer")
		c := toChannel(channel)

		if _, err := r.azsbDispatcher.UpdateSubscriptions(ctx, c, true); err != nil {
			logging.FromContext(ctx).Error("Error updating subscriptions", zap.Any("channel", c), zap.Error(err))
			return err
		}
		removeFinalizer(channel)
		_, err := r.azsbClientSet.MessagingV1alpha1().AZServicebusChannels(channel.Namespace).Update(channel)
		return err
	}

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

	// TODO: Should this check for subscribable status rather than entire status?
	if _, updateStatusErr := r.updateStatus(ctx, channel); updateStatusErr != nil {
		logging.FromContext(ctx).Error("Failed to update AZServicebusChannel status", zap.Error(updateStatusErr))
		r.Recorder.Eventf(channel, corev1.EventTypeWarning, channelUpdateStatusFailed, "Failed to update AZServicebusChannel's status: %v", updateStatusErr)
		return updateStatusErr
	}

	// Requeue if the resource is not ready
	return reconcileErr
}

func (r *Reconciler) updateStatus(ctx context.Context, desired *v1alpha1.AZServicebusChannel) (*v1alpha1.AZServicebusChannel, error) {
	nc, err := r.azsbchannelLister.AZServicebusChannels(desired.Namespace).Get(desired.Name)
	if err != nil {
		return nil, err
	}

	if reflect.DeepEqual(nc.Status, desired.Status) {
		return nc, nil
	}

	// Don't modify the informers copy.
	existing := nc.DeepCopy()
	existing.Status = desired.Status

	return r.azsbClientSet.MessagingV1alpha1().AZServicebusChannels(desired.Namespace).UpdateStatus(existing)
}

func (r *Reconciler) reconcile(ctx context.Context, azsbc *v1alpha1.AZServicebusChannel) error {

	c := toChannel(azsbc)

	// If we are adding the finalizer for the first time, then ensure that finalizer is persisted
	// before manipulating Natss.
	if err := r.ensureFinalizer(azsbc); err != nil {
		return err
	}

	failedSubscriptions, err := r.azsbDispatcher.UpdateSubscriptions(ctx, c, false)
	if err != nil {
		logging.FromContext(ctx).Error("Error updating az service bus consumers in dispatcher")
		return err
	}

	azsbc.Status.SubscribableTypeStatus.SubscribableStatus = r.createSubscribableStatus(azsbc.Spec.Subscribable, failedSubscriptions)
	if len(failedSubscriptions) > 0 {
		logging.FromContext(ctx).Error("Some az service bus subscriptions failed to subscribe")
		return fmt.Errorf("Some az service bus subscriptions failed to subscribe")
	}

	channels, err := r.azsbchannelLister.List(labels.Everything())
	if err != nil {
		logging.FromContext(ctx).Error("Error listing az service bus channels")
		return err
	}

	// TODO: revisit this code. Instead of reading all channels and updating consumers and hostToChannel map for all
	// why not just reconcile the current channel. With this the UpdateAZServiceBusConsumers can now return SubscribableStatus
	// for the subscriptions on the channel that is being reconciled.
	azsbChannels := make([]messagingv1alpha1.Channel, 0)
	for _, channel := range channels {
		if channel.Status.IsReady() {
			azsbChannels = append(azsbChannels, *toChannel(channel))
		}
	}

	if err := r.azsbDispatcher.ProcessChannels(ctx, azsbChannels); err != nil {
		logging.FromContext(ctx).Error("Error updating host to channel map", zap.Error(err))
		return err
	}

	return nil
}

func (r *Reconciler) createSubscribableStatus(subscribable *eventingduck.Subscribable, failedSubscriptions map[eventingduck.SubscriberSpec]error) *eventingduck.SubscribableStatus {
	if subscribable == nil {
		return nil
	}
	subscriberStatus := make([]eventingduck.SubscriberStatus, 0)
	for _, sub := range subscribable.Subscribers {
		status := eventingduck.SubscriberStatus{
			UID:                sub.UID,
			ObservedGeneration: sub.Generation,
			Ready:              corev1.ConditionTrue,
		}
		if err, ok := failedSubscriptions[sub]; ok {
			status.Ready = corev1.ConditionFalse
			status.Message = err.Error()
		}
		subscriberStatus = append(subscriberStatus, status)
	}
	return &eventingduck.SubscribableStatus{
		Subscribers: subscriberStatus,
	}
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

	_, err = r.azsbClientSet.MessagingV1alpha1().AZServicebusChannels(channel.Namespace).Patch(channel.Name, types.MergePatchType, patch)
	return err
}

func removeFinalizer(channel *v1alpha1.AZServicebusChannel) {
	finalizers := sets.NewString(channel.Finalizers...)
	finalizers.Delete(finalizerName)
	channel.Finalizers = finalizers.List()
}

func toChannel(azsbChannel *v1alpha1.AZServicebusChannel) *messagingv1alpha1.Channel {
	channel := &messagingv1alpha1.Channel{
		ObjectMeta: v1.ObjectMeta{
			Name:      azsbChannel.Name,
			Namespace: azsbChannel.Namespace,
		},
		Spec: messagingv1alpha1.ChannelSpec{
			Subscribable: azsbChannel.Spec.Subscribable,
		},
	}
	if azsbChannel.Status.Address != nil {
		channel.Status = messagingv1alpha1.ChannelStatus{
			AddressStatus: azsbChannel.Status.AddressStatus,
		}
	}
	return channel
}
