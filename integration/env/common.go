//go:build k8srequired
// +build k8srequired

package env

import (
	"fmt"
	"os"
)

const (
	// EnvVarCircleSHA is the process environment variable representing the
	// CIRCLE_SHA1 env var.
	EnvVarCircleSHA = "CIRCLE_SHA1"
	// EnvVarE2EKubeconfig is the process environment variable representing the
	// E2E_KUBECONFIG env var.
	EnvVarE2EKubeconfig = "E2E_KUBECONFIG"
)

var (
	circleSHA  string
	kubeconfig string
)

func init() {
	circleSHA = os.Getenv(EnvVarCircleSHA)
	if circleSHA == "" {
		panic(fmt.Sprintf("env var '%s' must not be empty", EnvVarCircleSHA))
	}

	kubeconfig = os.Getenv(EnvVarE2EKubeconfig)
	if kubeconfig == "" {
		panic(fmt.Sprintf("env var '%s' must not be empty", EnvVarE2EKubeconfig))
	}
}

func CircleSHA() string {
	return circleSHA
}

// ChartVersionSHASuffix returns the trailing SHA fragment that architect stamps
// onto dev chart versions, e.g. "ha781825" for the version
// "2.0.2-dev.my-branch.2026-08-19.13-58-39.ha781825".
//
// appcatalog resolves a chart by testing strings.HasSuffix(entry.Version, sha),
// which worked while architect orb 6.x published "<version>-<full 40-char SHA>".
// Orb 9.x publishes the gitsemver dev format above, whose suffix is "h" plus the
// 7-character short SHA, so the full SHA can never match and every lookup fails
// with "no app ... in index.yaml with given appVersion".
func ChartVersionSHASuffix() string {
	const shortLen = 7

	if len(circleSHA) < shortLen {
		return circleSHA
	}

	return fmt.Sprintf("h%s", circleSHA[:shortLen])
}

func KubeConfig() string {
	return kubeconfig
}
