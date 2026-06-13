// Command rollops-plugin-gatewayapi is a Rollops traffic-router plugin backed by
// the Kubernetes Gateway API. Build it, pin its sha256, and point a rollout's
// trafficRouting.plugin at the binary.
package main

import (
	"fmt"
	"os"

	gatewayapi "github.com/klarlabs-studio/rollops-plugin-gatewayapi"
	"go.klarlabs.de/rollops/pkg/plugin"
)

// version is overwritten at build time via -ldflags.
var version = "dev"

func main() {
	safety := plugin.Safety{
		// Drives the cluster via kubectl (ambient kubeconfig / in-cluster); no
		// direct egress to declare.
		EnvVars:   []string{"GATEWAYAPI_KUBECTL", "KUBECONFIG"},
		RiskClass: plugin.RiskActive,
	}
	if err := plugin.ServeTrafficRouter("klarlabs/gatewayapi", version, gatewayapi.FromEnv(), safety); err != nil {
		fmt.Fprintln(os.Stderr, "rollops-plugin-gatewayapi:", err)
		os.Exit(1)
	}
}
