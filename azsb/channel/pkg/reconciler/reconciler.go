package reconciler

import (
	"context"
	"time"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	dynamicclient "knative.dev/pkg/injection/clients/dynamicclient"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/logging/logkey"
	"knative.dev/pkg/system"

	corev1 "k8s.io/api/core/v1"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	clientset "knative.dev/eventing-contrib/azsb/channel/pkg/client/clientset/versioned"
	azsbScheme "knative.dev/eventing-contrib/azsb/channel/pkg/client/clientset/versioned/scheme"
	legacyclientset "knative.dev/eventing/pkg/legacyclient/clientset/versioned"
	legacyScheme "knative.dev/eventing/pkg/legacyclient/clientset/versioned/scheme"
	legacyclient "knative.dev/eventing/pkg/legacyclient/injection/client"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
)

// Options defines the common reconciler options.
// We define this to reduce the boilerplate argument list when
// creating our controllers.
type Options struct {
	KubeClientSet    kubernetes.Interface
	DynamicClientSet dynamic.Interface

	AZServicebusClientSet clientset.Interface

	Recorder      record.EventRecorder
	StatsReporter StatsReporter

	ConfigMapWatcher *configmap.InformedWatcher
	Logger           *zap.SugaredLogger

	ResyncPeriod time.Duration
	StopChannel  <-chan struct{}
}

// This is mutable for testing.
var resetPeriod = 30 * time.Second

// NewOptionsOrDie creates a new Clientset for the given config.
func NewOptionsOrDie(cfg *rest.Config, logger *zap.SugaredLogger, stopCh <-chan struct{}) Options {
	kubeClient := kubernetes.NewForConfigOrDie(cfg)
	dynamicClient := dynamic.NewForConfigOrDie(cfg)

	azsbClient := clientset.NewForConfigOrDie(cfg)

	configMapWatcher := configmap.NewInformedWatcher(kubeClient, system.Namespace())

	return Options{
		KubeClientSet:         kubeClient,
		DynamicClientSet:      dynamicClient,
		ConfigMapWatcher:      configMapWatcher,
		AZServicebusClientSet: azsbClient,
		Logger:                logger,
		ResyncPeriod:          10 * time.Hour, // Based on controller-runtime default.
		StopChannel:           stopCh,
	}
}

// GetTrackerLease returns a multiple of the resync period to use as the
// duration for tracker leases. This attempts to ensure that resyncs happen to
// refresh leases frequently enough that we don't miss updates to tracked
// objects.
func (o Options) GetTrackerLease() time.Duration {
	return o.ResyncPeriod * 3
}

// Base implements the core controller logic, given a Reconciler.
type Base struct {
	// LegacyClientSet allows us to configure Legacy Eventing objects
	LegacyClientSet legacyclientset.Interface
	// KubeClientSet allows us to talk to the k8s for core APIs
	KubeClientSet kubernetes.Interface

	// DynamicClientSet allows us to configure pluggable Build objects
	DynamicClientSet dynamic.Interface

	// ConfigMapWatcher allows us to watch for ConfigMap changes.
	ConfigMapWatcher configmap.Watcher

	// Recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	Recorder record.EventRecorder

	// StatsReporter reports reconciler's metrics.
	StatsReporter StatsReporter

	// Sugared logger is easier to use but is not as performant as the
	// raw logger. In performance critical paths, call logger.Desugar()
	// and use the returned raw logger instead. In addition to the
	// performance benefits, raw logger also preserves type-safety at
	// the expense of slightly greater verbosity.
	Logger *zap.SugaredLogger
}

// NewBase instantiates a new instance of Base implementing
// the common & boilerplate code between our reconcilers.
func NewBase(ctx context.Context, controllerAgentName string, cmw configmap.Watcher) *Base {
	// Enrich the logs with controller name
	// Enrich the logs with controller name
	logger := logging.FromContext(ctx).Named(controllerAgentName).With(zap.String(logkey.ControllerType, controllerAgentName))

	kubeClient := kubeclient.Get(ctx)

	recorder := controller.GetEventRecorder(ctx)

	if recorder == nil {
		// Create event broadcaster
		logger.Debug("Creating event broadcaster")
		eventBroadcaster := record.NewBroadcaster()
		watches := []watch.Interface{
			eventBroadcaster.StartLogging(logger.Named("event-broadcaster").Infof),
			eventBroadcaster.StartRecordingToSink(
				&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")}),
		}
		recorder = eventBroadcaster.NewRecorder(
			scheme.Scheme, corev1.EventSource{Component: controllerAgentName})
		go func() {
			<-ctx.Done()
			for _, w := range watches {
				w.Stop()
			}
		}()
	}

	statsReporter := GetStatsReporter(ctx)
	if statsReporter == nil {
		logger.Debug("Creating stats reporter")
		var err error
		statsReporter, err = NewStatsReporter(controllerAgentName)
		if err != nil {
			logger.Fatal(err)
		}
	}

	base := &Base{
		KubeClientSet:    kubeClient,
		DynamicClientSet: dynamicclient.Get(ctx),
		LegacyClientSet:  legacyclient.Get(ctx),
		ConfigMapWatcher: cmw,
		Recorder:         recorder,
		StatsReporter:    statsReporter,
		Logger:           logger,
	}

	return base
}

func init() {
	// Add run types to the default Kubernetes Scheme so Events can be
	// logged for run types.
	_ = azsbScheme.AddToScheme(scheme.Scheme)
	_ = legacyScheme.AddToScheme(scheme.Scheme)
}
