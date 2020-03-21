package dispatcher

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	eventingduck "knative.dev/eventing/pkg/apis/duck/v1alpha1"
	messagingv1alpha1 "knative.dev/eventing/pkg/apis/messaging/v1alpha1"
	eventingchannels "knative.dev/eventing/pkg/channel"
	"knative.dev/eventing/pkg/kncloudevents"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/tracing"

	bus "github.com/Azure/azure-service-bus-go"

	"github.com/cloudevents/sdk-go/v1/cloudevents"
	cloudeventsclient "github.com/cloudevents/sdk-go/v1/cloudevents/client"
	cloudeventstransport "github.com/cloudevents/sdk-go/v1/cloudevents/transport/http"
	"go.uber.org/zap"
)

const (
	// maxElements defines a maximum number of outstanding re-connect requests
	maxElements = 10
	// tracingSpanIgnoringPath defines the tracing path to ignore
	tracingSpanIgnoringPath = "/readyz"
)

// SubscriptionChannelMapping maps subcriptions to channels
type SubscriptionChannelMapping map[eventingchannels.ChannelReference]map[subscriptionReference]*bus.Subscription

// SubscriptionsSupervisor superviser for AZ Service Bus
type SubscriptionsSupervisor struct {
	hostToChannelMap atomic.Value
	// hostToChannelMapLock is used to update hostToChannelMap
	hostToChannelMapLock sync.Mutex

	receiver   *eventingchannels.EventReceiver
	dispatcher *eventingchannels.EventDispatcher
	ceClient   cloudeventsclient.Client

	azsbMutex          sync.Mutex
	connectionEndpoint string
	subscriptions      SubscriptionChannelMapping
	subscriptionLock   sync.Mutex
	azNamespace        *bus.Namespace

	logger *zap.Logger
}

// AZServicebusDispatcher dispatches
type AZServicebusDispatcher interface {
	Start(ctx context.Context) error
	UpdateSubscriptions(ctx context.Context, channel *messagingv1alpha1.Channel, isFinalizer bool) (map[eventingduck.SubscriberSpec]error, error)
	ProcessChannels(ctx context.Context, chanList []messagingv1alpha1.Channel) error
}

// NewDispatcher returns a new AZSbDispatcher.
func NewDispatcher(connectionEndpoint string, logger *zap.Logger, args *kncloudevents.ConnectionArgs) (AZServicebusDispatcher, error) {
	d := &SubscriptionsSupervisor{
		logger:             logger,
		dispatcher:         eventingchannels.NewEventDispatcher(logger),
		connectionEndpoint: connectionEndpoint,
		subscriptions:      make(SubscriptionChannelMapping),
		azNamespace:        getNewSasInstance(connectionEndpoint),
	}
	httpTransport, err := cloudeventstransport.New(cloudeventstransport.WithStructuredEncoding(), cloudeventstransport.WithMiddleware(tracing.HTTPSpanIgnoringPaths(tracingSpanIgnoringPath)))
	if err != nil {
		logger.Fatal("failed to create httpTransport", zap.Error(err))
	}
	ceClient, err := kncloudevents.NewDefaultClientGivenHttpTransport(httpTransport, args)
	if err != nil {
		logger.Fatal("failed to create cloudevents client", zap.Error(err))
	}
	d.ceClient = ceClient
	d.setHostToChannelMap(map[string]eventingchannels.ChannelReference{})
	receiver, err := eventingchannels.NewEventReceiver(
		createReceiverFunction(d, logger.Sugar()),
		logger,
		eventingchannels.ResolveChannelFromHostHeader(d.getChannelReferenceFromHost))
	if err != nil {
		return nil, err
	}
	d.receiver = receiver
	return d, nil
}

func (s *SubscriptionsSupervisor) setHostToChannelMap(hcMap map[string]eventingchannels.ChannelReference) {
	s.logger.Sugar().Infof("this is the hcMap: %v", hcMap)
	for k, v := range hcMap {
		s.logger.Sugar().Infof("key: %s, name: %s, namespace: %s", k, v.Name, v.Namespace)
	}
	s.hostToChannelMap.Store(hcMap)
}

// Start starts the dispatcher
func (s *SubscriptionsSupervisor) Start(ctx context.Context) error {
	s.logger.Info("Starting the Azure Service Bus Dispatcher")
	return s.ceClient.StartReceiver(ctx, s.receiver.ServeHTTP)
}

func createReceiverFunction(s *SubscriptionsSupervisor, logger *zap.SugaredLogger) eventingchannels.ReceiverFunc {
	logger.Info("inside the createReceiverFunction")
	return func(ctx context.Context, channel eventingchannels.ChannelReference, event cloudevents.Event) error {
		logger.Infof("Received message from %q channel", channel.String())
		m, err := event.MarshalJSON()
		if err != nil {
			logger.Errorf("Error during marshaling of the message: %v", err)
			return err
		}
		logger.Debugf("cloud event received: %s", m)
		topic, err := s.azNamespace.NewTopic(channel.Name)
		if err != nil {
			logger.Errorf("Error creating the topic in the createReceiverFunction: %v", err)
			return err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		topic.Close(ctx)
		defer cancel()
		err = topic.Send(ctx, toAZServicebusMessage(&event))
		if err != nil {
			logger.Errorf("Error sending message to azure service bus topic %v", err)
			return err
		}
		topic.Close(ctx)
		return nil
	}
}

func toAZServicebusMessage(event *cloudevents.Event) *bus.Message {
	azsbMessage := bus.Message{}
	azsbMessage.Data, _ = event.MarshalJSON()
	azsbMessage.Label = event.Subject()
	azsbMessage.ID = event.ID()
	azsbMessage.ContentType = event.DataContentType()
	up := map[string]interface{}{
		"Ce-Type":        event.Type(),
		"Ce-Specversion": event.SpecVersion(),
		"Ce-Time":        event.Time(),
	}
	azsbMessage.UserProperties = up
	return &azsbMessage
}

// NewHostNameToChannelRefMap parses each channel from cList and creates a map[string(Status.Address.HostName)]ChannelReference
func newHostNameToChannelRefMap(cList []messagingv1alpha1.Channel, logger *zap.SugaredLogger) (map[string]eventingchannels.ChannelReference, error) {
	hostToChanMap := make(map[string]eventingchannels.ChannelReference, len(cList))
	for _, c := range cList {
		url := c.Status.Address.GetURL()
		logger.Infof("this is the url in the newHostNameToChannelRefMap: %v", url)
		if cr, present := hostToChanMap[url.Host]; present {
			return nil, fmt.Errorf(
				"duplicate hostName found. Each channel must have a unique host header. HostName:%s, channel:%s.%s, channel:%s.%s",
				url.Host,
				c.Namespace,
				c.Name,
				cr.Namespace,
				cr.Name)
		}
		logger.Infof("this is the url Host: %s, the channel name: %s, and the namespace: %s", url.Host, c.Name, c.Namespace)
		hostToChanMap[url.Host] = eventingchannels.ChannelReference{Name: c.Name, Namespace: c.Namespace}
	}
	return hostToChanMap, nil
}

// ProcessChannels will be called from the controller that watches az service bus channels.
// It will update internal hostToChannelMap which is used to resolve the hostHeader of the
// incoming request to the correct ChannelReference in the receiver function.
func (s *SubscriptionsSupervisor) ProcessChannels(ctx context.Context, chanList []messagingv1alpha1.Channel) error {
	hostToChanMap, err := newHostNameToChannelRefMap(chanList, s.logger.Sugar())
	if err != nil {
		logging.FromContext(ctx).Info("ProcessChannels: Error occurred when creating the new hostToChannel map.", zap.Error(err))
		return err
	}
	s.setHostToChannelMap(hostToChanMap)
	logging.FromContext(ctx).Info("hostToChannelMap updated successfully.")
	return nil
}

func (s *SubscriptionsSupervisor) subscribe(ctx context.Context, channel eventingchannels.ChannelReference, subscription subscriptionReference) (*bus.Subscription, error) {
	s.logger.Info("Subscribe to channel:", zap.Any("channel", channel), zap.Any("subscription", subscription))
	messageHandler := func() bus.HandlerFunc {
		return func(eventCtx context.Context, msg *bus.Message) error {
			event := cloudevents.Event{}
			err := event.UnmarshalJSON(msg.Data)
			s.logger.Sugar().Infof("received full message: %v, payload:", msg, string(msg.Data))
			s.logger.Sugar().Infof("Azure Service Bus message received from label: %v; id: %v; timestamp: %v'", msg.Label, msg.ID, msg.ScheduleAt)
			s.logger.Sugar().Infof("Azure Service Bus message subscription info Topic URL: %s, Reply URL: %s.", subscription.TopicURL, subscription.TopicURL)
			if err != nil {
				s.logger.Error(err.Error(), zap.Error(err))
				return err
			}
			if err := event.Validate(); err != nil {
				s.logger.Error(err.Error(), zap.Error(err))
				return err
			}
			if err := s.dispatcher.DispatchEventWithDelivery(context.TODO(), event, subscription.TopicURL, subscription.ReplyURL, &subscription.Delivery); err != nil {
				s.logger.Error("Failed to dispatch message: ", zap.Error(err))
				return nil
			}
			if err := msg.Complete(ctx); err != nil {
				s.logger.Error("Failed to acknowledge message: ", zap.Error(err))
			}
			return nil
		}
	}
	s.logger.Info("created the message handler successfully")
	subName := subscription.String()
	s.logger.Info("here is the subcription name to create:", zap.Any("subscriptionName", subName))

	t, err := s.azNamespace.NewTopic(channel.Name)
	if err != nil {
		s.logger.Error("Getting the Topic from Azure Service Bus failed: ", zap.Error(err))
		return nil, err
	}

	var sub *bus.Subscription
	sm := t.NewSubscriptionManager()
	_, err = sm.Get(ctx, subName)

	if err != nil && strings.ContainsAny(err.Error(), "not found") {
		s.logger.Sugar().Infof("did not find the subscription, so we're going to try and create it", subName)
		_, err = sm.Put(ctx, subName)
		if err != nil {
			s.logger.Error("Creating the subscription from Azure Service Bus failed: ", zap.Error(err))
			return nil, err
		}
		s.logger.Sugar().Infof("successfully created the subscription", subName)
		// Try again to get the newly created subscription
		sub, err = t.NewSubscription(subName)
		if err != nil {
			s.logger.Error("Failed to retrieve the newly created Azure Service Bus subscription: ", zap.Error(err))
			return nil, err
		}
	} else if err != nil {
		s.logger.Error("different error than not found:", zap.Error(err))
		return nil, err
	} else {
		s.logger.Info("subscription already exists so just assigning creating the object")
		sub, err = t.NewSubscription(subName)
	}
	go func() {
		err = sub.Receive(ctx, messageHandler())
		if err != nil {
			log.Println(err)
		}
	}()
	if err != nil {
		s.logger.Error("Failed to set up the receive handler for the subscription: ", zap.Error(err))
		return nil, err
	}
	s.logger.Sugar().Infof("Azure Service Bus Subscription created: %+v", sub.Name)

	return sub, nil
}

// should be called only while holding subscriptionsMux
func (s *SubscriptionsSupervisor) unsubscribe(ctx context.Context, channel eventingchannels.ChannelReference, subscription subscriptionReference) error {
	s.logger.Info("Unsubscribe from channel:", zap.Any("channel", channel), zap.Any("subscription", subscription))

	if azsbSub, ok := s.subscriptions[channel][subscription]; ok {
		// Close Subscription
		if err := azsbSub.Close(ctx); err != nil {
			s.logger.Error("Closing Azure Service Bus Subscription failed: ", zap.Error(err))
			return err
		}
		s.logger.Info("Unsubscribed from channel let's now try to delete it: ", zap.Any("channel", channel), zap.Any("subscription", subscription))
		t, err := s.azNamespace.NewTopic(channel.Name)
		if err != nil {
			s.logger.Error("Getting the Topic from Azure Service Bus failed: ", zap.Error(err))
			return err
		}
		sm := t.NewSubscriptionManager()
		err = sm.Delete(ctx, subscription.Name)
		if err != nil {
			s.logger.Error("Deleting the subscription from Azure Service Bus failed: ", zap.Error(err))
			return err
		}
		delete(s.subscriptions[channel], subscription)
	}
	return nil
}

// UpdateSubscriptions creates/deletes the azsb subscriptions based on channel.Spec.Subscribable.Subscribers
// Return type:map[eventingduck.SubscriberSpec]error --> Returns a map of subscriberSpec that failed with the value=error encountered.
// Ignore the value in case error != nil
func (s *SubscriptionsSupervisor) UpdateSubscriptions(ctx context.Context, channel *messagingv1alpha1.Channel, isFinalizer bool) (map[eventingduck.SubscriberSpec]error, error) {
	s.subscriptionLock.Lock()
	defer s.subscriptionLock.Unlock()

	failedToSubscribe := make(map[eventingduck.SubscriberSpec]error)
	cRef := eventingchannels.ChannelReference{Namespace: channel.Namespace, Name: channel.Name}
	if channel.Spec.Subscribable == nil || isFinalizer {
		s.logger.Sugar().Infof("Empty subscriptions for channel Ref: %v; unsubscribe all active subscriptions, if any", cRef)
		chMap, ok := s.subscriptions[cRef]
		if !ok {
			// nothing to do
			s.logger.Sugar().Infof("No channel Ref %v found in subscriptions map", cRef)
			return failedToSubscribe, nil
		}
		for sub := range chMap {
			s.logger.Sugar().Infof("Unsubscribing to channel Ref %v found in subscriptions map", cRef)
			s.unsubscribe(ctx, cRef, sub)
		}
		delete(s.subscriptions, cRef)
		return failedToSubscribe, nil
	}

	s.logger.Sugar().Infof("right before channel.Spec.Subscribable.Subscribers")
	subscriptions := channel.Spec.Subscribable.Subscribers
	activeSubs := make(map[subscriptionReference]bool) // it's logically a set
	s.logger.Sugar().Infof("right after activeSubs")

	chMap, ok := s.subscriptions[cRef]
	if !ok {
		chMap = make(map[subscriptionReference]*bus.Subscription)
		s.subscriptions[cRef] = chMap
	}
	var errStrings []string
	s.logger.Sugar().Infof("before subscription loops")
	for _, sub := range subscriptions {
		// check if the subscription already exist and do nothing in this case
		subRef := newSubscriptionReference(sub)
		subRef.Name = cRef.Name
		subRef.Namespace = cRef.Namespace
		s.logger.Sugar().Infof("subscription reference: %v", subRef)
		if _, ok := chMap[subRef]; ok {
			activeSubs[subRef] = true
			s.logger.Sugar().Infof("Subscription: %v already active for channel: %v", sub, cRef)
			continue
		}
		// subscribe and update failedSubscription if subscribe fails
		s.logger.Sugar().Infof("trying subscribe to channel Ref %v found in subscriptions map", cRef)
		azsub, err := s.subscribe(ctx, cRef, subRef)
		if err != nil {
			errStrings = append(errStrings, err.Error())
			s.logger.Sugar().Errorf("failed to subscribe (subscription:%q) to channel: %v. Error:%s", sub, cRef, err.Error())
			failedToSubscribe[sub] = err
			continue
		}
		chMap[subRef] = azsub
		activeSubs[subRef] = true
	}
	// Unsubscribe for deleted subscriptions
	for sub := range chMap {
		if ok := activeSubs[sub]; !ok {
			s.unsubscribe(ctx, cRef, sub)
		}
	}
	// delete the channel from s.subscriptions if chMap is empty
	if len(s.subscriptions[cRef]) == 0 {
		delete(s.subscriptions, cRef)
	}
	return failedToSubscribe, nil
}

func getSubject(channel eventingchannels.ChannelReference) string {
	return channel.Name + "." + channel.Namespace
}

func (s *SubscriptionsSupervisor) getHostToChannelMap() map[string]eventingchannels.ChannelReference {
	s.logger.Sugar().Infof("inside the getHostToChannelMap")
	return s.hostToChannelMap.Load().(map[string]eventingchannels.ChannelReference)
}

func (s *SubscriptionsSupervisor) getChannelReferenceFromHost(host string) (eventingchannels.ChannelReference, error) {
	s.logger.Sugar().Infof("inside the getChannelReferenceFromHost, looking for host: %s", host)
	chMap := s.getHostToChannelMap()
	cr, ok := chMap[host]
	if !ok {
		return cr, fmt.Errorf("Invalid HostName:%q. HostName not found in any of the watched az service bus channels", host)
	}
	return cr, nil
}
func getNewSasInstance(connection string, opts ...bus.NamespaceOption) *bus.Namespace {
	ns, err := bus.NewNamespace(append(opts, bus.NamespaceWithConnectionString(connection))...)
	if err != nil {
		log.Fatal(err)
	}
	return ns
}
