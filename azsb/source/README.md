1. Create the `AzsbSource` custom objects, by configuring the required
   `consumerGroup`, `bootstrapServers` and `topics` values on the CR file of
   your source. Below is an example:

   ```yaml
   apiVersion: sources.knative.dev/v1alpha1
   kind: AzsbSource
   metadata:
     name: azsb-source
   spec:
     subscription: knative-subscription
     topic: knative-demo-topic
     connectionString:
       secretKeyRef:
         name: sb-connection
         key: sb_connection
     sink:
       ref:
         apiVersion: serving.knative.dev/v1alpha1
         kind: Service
         name: event-display
   ```

## Example