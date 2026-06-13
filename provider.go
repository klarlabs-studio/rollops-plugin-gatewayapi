// Package gatewayapi is a Rollops traffic-router plugin backed by the Kubernetes
// Gateway API. On each canary step it patches an HTTPRoute's backendRefs so the
// canary Service receives the step weight and the stable Service the remainder,
// shifting real traffic in lockstep with a Rollops canary.
//
// It drives the cluster through kubectl (ambient kubeconfig / in-cluster
// config): get the HTTPRoute as JSON, rewrite the two backends' weights in
// place, and replace it. Working in place preserves every other field of the
// route (ports, matches, filters, other backends).
package gatewayapi

import (
	"context"
	"encoding/json"
	"fmt"

	"go.klarlabs.de/rollops/pkg/plugin"
)

// Runner runs kubectl with optional stdin and returns stdout. Injectable so the
// router is testable without a cluster.
type Runner func(ctx context.Context, stdin []byte, args ...string) (string, error)

// Router shifts traffic by editing an HTTPRoute's backendRef weights.
type Router struct {
	Kubectl string // kubectl binary (default "kubectl")
	Run     Runner
}

func (r Router) bin() string {
	if r.Kubectl != "" {
		return r.Kubectl
	}
	return "kubectl"
}

// SetWeight gets the HTTPRoute, sets the canary backend to c.Weight and the
// stable backend to 100-c.Weight across every rule that references them, and
// replaces the route.
func (r Router) SetWeight(ctx context.Context, c plugin.TrafficChange) error {
	if r.Run == nil {
		return fmt.Errorf("gatewayapi: no kubectl runner configured")
	}
	ns := c.Namespace
	if ns == "" {
		ns = "default"
	}
	out, err := r.Run(ctx, nil, r.bin(), "get", "httproute", c.Route, "-n", ns, "-o", "json")
	if err != nil {
		return fmt.Errorf("gatewayapi: get httproute %q: %w", c.Route, err)
	}
	var route map[string]any
	if err := json.Unmarshal([]byte(out), &route); err != nil {
		return fmt.Errorf("gatewayapi: parse httproute: %w", err)
	}
	matched, err := setBackendWeights(route, c.CanaryService, c.StableService, c.Weight)
	if err != nil {
		return err
	}
	if matched == 0 {
		return fmt.Errorf("gatewayapi: httproute %q has no backendRefs for %q/%q", c.Route, c.CanaryService, c.StableService)
	}
	patched, err := json.Marshal(route)
	if err != nil {
		return err
	}
	if _, err := r.Run(ctx, patched, r.bin(), "replace", "-f", "-"); err != nil {
		return fmt.Errorf("gatewayapi: replace httproute %q: %w", c.Route, err)
	}
	return nil
}

// setBackendWeights walks spec.rules[].backendRefs[] and sets the weight of the
// canary and stable backends. Returns how many backendRefs it updated.
func setBackendWeights(route map[string]any, canary, stable string, weight int) (int, error) {
	spec, ok := route["spec"].(map[string]any)
	if !ok {
		return 0, fmt.Errorf("gatewayapi: httproute has no spec")
	}
	rules, ok := spec["rules"].([]any)
	if !ok {
		return 0, fmt.Errorf("gatewayapi: httproute spec has no rules")
	}
	matched := 0
	for _, ru := range rules {
		rule, ok := ru.(map[string]any)
		if !ok {
			continue
		}
		refs, ok := rule["backendRefs"].([]any)
		if !ok {
			continue
		}
		for _, br := range refs {
			ref, ok := br.(map[string]any)
			if !ok {
				continue
			}
			name, _ := ref["name"].(string)
			switch name {
			case canary:
				ref["weight"] = weight
				matched++
			case stable:
				ref["weight"] = 100 - weight
				matched++
			}
		}
	}
	return matched, nil
}
