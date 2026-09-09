//go:build k8srequired
// +build k8srequired

package env

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// EnvVarE2EKubeconfig is the process environment variable representing the
	// E2E_KUBECONFIG env var.
	EnvVarE2EKubeconfig = "E2E_KUBECONFIG"
)

var (
	buildVersion string
	kubeconfig   string
)

func init() {
	filePath := filepath.Join(os.Getenv("CIRCLE_WORKING_DIRECTORY"), ".build_version")
	buf, err := os.ReadFile(filePath)
	if err != nil {
		panic(fmt.Sprintf("error reading .build_version: %v", err))
	}

	buildVersion = strings.TrimSpace(string(buf))
	if buildVersion == "" {
		panic(".build_version must not be empty")
	}

	kubeconfig = os.Getenv(EnvVarE2EKubeconfig)
	if kubeconfig == "" {
		panic(fmt.Sprintf("env var '%s' must not be empty", EnvVarE2EKubeconfig))
	}
}

func BuildVersion() string {
	return buildVersion
}

func KubeConfig() string {
	return kubeconfig
}
