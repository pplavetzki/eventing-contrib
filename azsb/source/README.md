1. Create the `AzsbSource` custom objects, by configuring the required
   `consumerGroup`, `bootstrapServers` and `topics` values on the CR file of
   your source. Below is an example:

```yaml
apiVersion: sources.knative.dev/v1alpha1
kind: AzsbSource
metadata:
  name: azsb-source
spec:
  subscription: 4fab069b-8707-4892-9dc4-475f28d97af6
  topic: azsb-knative-kne-trigger
  connectionString:
    secretKeyRef:
      name: sb-connection
      key: sb_connection
  sink:
    ref:
      apiVersion: serving.knative.dev/v1
      kind: Service
      name: autoscale-go
```

## Example

```yaml
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: autoscale-go
spec:
  template:
    metadata:
      annotations:
        # Knative concurrency-based autoscaling (default).
        autoscaling.knative.dev/class: hpa.autoscaling.knative.dev
        autoscaling.knative.dev/metric: cpu
        # Target 10 requests in-flight per pod.
        autoscaling.knative.dev/target: "50"
        # Disable scale to zero with a minScale of 1.
        autoscaling.knative.dev/minScale: "0"
        # Limit scaling to 100 pods.
        autoscaling.knative.dev/maxScale: "50"
    spec:
      containers:
      - image: pplavetzki.azurecr.io/dev/knative/auto-scale:20200317.5
        resources:
          limits:
            cpu: 500M
          requests:
            cpu: 200m
      imagePullSecrets:
      - name: acr-login-secret
```