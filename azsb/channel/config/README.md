# Azure Service Bus Channels

Azure Service Bus channels are beta-quality Channels that are backed by

## Deployment steps

1. Setup
   [Knative Eventing](https://github.com/knative/eventing-contrib/blob/master/DEVELOPMENT.md).
1. Apply the Azure Service Bus configuration:

   ```shell
   ko apply -f azsb/config
   ```

1. Create Azure Service Bus channels:

   ```yaml
    apiVersion: messaging.knative.dev/v1alpha1
    kind: AZServicebusChannel
    metadata:
        name: azsb-knative # this is what is used to create the topic
    spec:
        enablePartition: false
        maxTopicSize: 1024
   ```
2. Create Azure Service Bus Brokers:
```yaml
apiVersion: eventing.knative.dev/v1alpha1
kind: Broker
metadata:
  name: azsb-knative
spec:
  channelTemplateSpec:
    apiVersion: messaging.knative.dev/v1alpha1
    kind: AZServicebusChannel
```