package gatewayapi

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"go.klarlabs.de/rollops/pkg/plugin"
)

const sampleRoute = `{
  "apiVersion": "gateway.networking.k8s.io/v1",
  "kind": "HTTPRoute",
  "metadata": {"name": "app-route", "namespace": "prod"},
  "spec": {
    "parentRefs": [{"name": "gw"}],
    "rules": [{
      "matches": [{"path": {"type": "PathPrefix", "value": "/"}}],
      "backendRefs": [
        {"name": "app-stable", "port": 80, "weight": 100},
        {"name": "app-canary", "port": 80, "weight": 0}
      ]
    }]
  }
}`

func TestSetWeight_UpdatesBackendWeights(t *testing.T) {
	var getArgs []string
	var replaced []byte
	run := func(_ context.Context, stdin []byte, args ...string) (string, error) {
		if args[1] == "get" {
			getArgs = args
			return sampleRoute, nil
		}
		replaced = stdin // replace -f -
		return "", nil
	}
	r := Router{Run: run}
	err := r.SetWeight(context.Background(), plugin.TrafficChange{
		Route: "app-route", Namespace: "prod",
		StableService: "app-stable", CanaryService: "app-canary", Weight: 30,
	})
	if err != nil {
		t.Fatalf("SetWeight: %v", err)
	}
	// kubectl get must target the right route/namespace.
	if strings.Join(getArgs, " ") != "kubectl get httproute app-route -n prod -o json" {
		t.Errorf("unexpected get args: %v", getArgs)
	}
	// The replaced route must carry canary=30, stable=70, with ports preserved.
	var route map[string]any
	if err := json.Unmarshal(replaced, &route); err != nil {
		t.Fatalf("parse replaced: %v", err)
	}
	refs := route["spec"].(map[string]any)["rules"].([]any)[0].(map[string]any)["backendRefs"].([]any)
	stable := refs[0].(map[string]any)
	canary := refs[1].(map[string]any)
	if int(canary["weight"].(float64)) != 30 || int(stable["weight"].(float64)) != 70 {
		t.Errorf("weights = stable %v / canary %v, want 70/30", stable["weight"], canary["weight"])
	}
	if int(canary["port"].(float64)) != 80 {
		t.Errorf("port must be preserved, got %v", canary["port"])
	}
}

func TestSetWeight_NoMatchingBackends(t *testing.T) {
	run := func(_ context.Context, _ []byte, args ...string) (string, error) {
		if args[1] == "get" {
			return sampleRoute, nil
		}
		return "", nil
	}
	r := Router{Run: run}
	err := r.SetWeight(context.Background(), plugin.TrafficChange{
		Route: "app-route", Namespace: "prod",
		StableService: "nope-stable", CanaryService: "nope-canary", Weight: 50,
	})
	if err == nil || !strings.Contains(err.Error(), "no backendRefs") {
		t.Fatalf("expected no-backendRefs error, got %v", err)
	}
}

func TestSetWeight_DefaultsNamespace(t *testing.T) {
	var getArgs []string
	run := func(_ context.Context, _ []byte, args ...string) (string, error) {
		if args[1] == "get" {
			getArgs = args
			return sampleRoute, nil
		}
		return "", nil
	}
	r := Router{Run: run}
	_ = r.SetWeight(context.Background(), plugin.TrafficChange{
		Route: "app-route", StableService: "app-stable", CanaryService: "app-canary", Weight: 10,
	})
	if strings.Join(getArgs, " ") != "kubectl get httproute app-route -n default -o json" {
		t.Errorf("empty namespace must default to 'default', got %v", getArgs)
	}
}

func TestSetWeight_RequiresRunner(t *testing.T) {
	if err := (Router{}).SetWeight(context.Background(), plugin.TrafficChange{Route: "r"}); err == nil {
		t.Fatal("missing runner must error")
	}
}
