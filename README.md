# rollops-plugin-gatewayapi

A [Rollops](https://github.com/klarlabs-studio/rollops) traffic-router plugin
backed by the Kubernetes [Gateway API](https://gateway-api.sigs.k8s.io/). On each
canary step it patches an `HTTPRoute`'s `backendRefs` so the canary Service
receives the step weight and the stable Service the remainder — shifting real
network traffic in lockstep with a Rollops canary (10% → 50% → 100%).

## How it works

Rollops calls the plugin's `set_weight` tool per progressive step with the route,
namespace, stable/canary service names, and the current weight. The plugin:

1. `kubectl get httproute <route> -n <ns> -o json`
2. rewrites the weight of the canary and stable `backendRefs` in place across
   every rule that references them (preserving ports, matches, filters, and any
   other backends),
3. `kubectl replace -f -` with the modified route.

Working in place means the plugin never has to reconstruct the route — only the
two backend weights change.

## Topology

You run a stable and a canary `Service` behind one `HTTPRoute` with both as
`backendRefs`:

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: app-route
  namespace: prod
spec:
  parentRefs: [{ name: gw }]
  rules:
    - backendRefs:
        - { name: app-stable, port: 80, weight: 100 }
        - { name: app-canary, port: 80, weight: 0 }
```

Rollops drives the two weights as the canary advances. Managing the canary
workload itself is the deploy target's job.

## Configuration

The plugin drives the cluster through `kubectl` using the ambient kubeconfig /
in-cluster config — the Rollops target spec carries only the route and service
names, never cluster credentials.

| Env var              | Required | Default   | Description              |
|----------------------|----------|-----------|--------------------------|
| `GATEWAYAPI_KUBECTL` | no       | `kubectl` | kubectl binary to use    |
| `KUBECONFIG`         | no       | —         | resolved by kubectl as usual |

## Install

```sh
rollops plugin install gatewayapi
```

Then wire it into a rollout spec:

```yaml
spec:
  strategy:
    type: canary
    steps: [{ weight: 20 }, { weight: 50 }, { weight: 100 }]
  trafficRouting:
    plugin: ~/.rollops/plugins/gatewayapi
    sha256: <pin>
    route: app-route
    namespace: prod
    stableService: app-stable
    canaryService: app-canary
```

## License

MIT
