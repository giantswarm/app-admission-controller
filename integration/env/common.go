//go:build k8srequired
// +build k8srequired

package env

import (
	"fmt"
	"os"
	"path/filepath"
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
	buildVersion string
	circleSHA    string
	kubeconfig   string
)

func init() {
	filePath := filepath.Join(os.Getenv("CIRCLE_WORKING_DIRECTORY"), ".build_version")
	buf, _ := os.ReadFile(filePath)
	if string(buf) == "" {
		panic(fmt.Sprintf(".build_version must not be empty"))
	}
	buildVersion = string(buf)

	circleSHA = os.Getenv(EnvVarCircleSHA)
	if circleSHA == "" {
		panic(fmt.Sprintf("env var '%s' must not be empty", EnvVarCircleSHA))
	}

	kubeconfig = os.Getenv(EnvVarE2EKubeconfig)
	if kubeconfig == "" {
		panic(fmt.Sprintf("env var '%s' must not be empty", EnvVarE2EKubeconfig))
	}
}

func BuildVersion() string {
	return buildVersion
}

func CircleSHA() string {
	return circleSHA
}

func KubeConfig() string {
	return kubeconfig
}
