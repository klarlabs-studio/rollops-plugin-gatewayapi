package gatewayapi

import (
	"bytes"
	"context"
	"os"
	"os/exec"
)

// FromEnv builds a Router that drives the cluster via kubectl using the ambient
// kubeconfig / in-cluster config. The Rollops target spec carries only the
// route, namespace, and backend service names; cluster credentials come from the
// plugin's own environment, exactly like a kubectl invocation would resolve them.
//
//	GATEWAYAPI_KUBECTL  kubectl binary to use (default "kubectl")
func FromEnv() Router {
	return Router{
		Kubectl: os.Getenv("GATEWAYAPI_KUBECTL"),
		Run:     execRunner,
	}
}

// execRunner runs kubectl with optional stdin.
func execRunner(ctx context.Context, stdin []byte, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	out, err := cmd.Output()
	return string(out), err
}
