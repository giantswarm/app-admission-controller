module github.com/giantswarm/app-admission-controller

go 1.15

require (
	github.com/Masterminds/semver/v3 v3.1.1
	github.com/giantswarm/apiextensions/v3 v3.8.0
	github.com/giantswarm/app/v3 v3.3.0
	github.com/giantswarm/apptest v0.7.1
	github.com/giantswarm/backoff v0.2.0
	github.com/giantswarm/k8sclient/v5 v5.0.0
	github.com/giantswarm/microerror v0.2.1
	github.com/giantswarm/micrologger v0.3.4
	github.com/google/go-cmp v0.5.3
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/uuid v1.1.2 // indirect
	github.com/onsi/ginkgo v1.14.1 // indirect
	github.com/onsi/gomega v1.10.2 // indirect
	github.com/prometheus/client_golang v1.8.0
	golang.org/x/crypto v0.0.0-20201002170205-7f63de1d35b0 // indirect
	golang.org/x/net v0.0.0-20201002202402-0a1ea396d57c // indirect
	golang.org/x/tools v0.0.0-20200706234117-b22de6825cf7 // indirect
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	k8s.io/api v0.18.9
	k8s.io/apimachinery v0.18.9
	k8s.io/client-go v0.18.9
	k8s.io/kube-openapi v0.0.0-20200410145947-bcb3869e6f29 // indirect
	k8s.io/utils v0.0.0-20200912215256-4140de9c8800 // indirect
)

replace (
	// Use v0.8.10 of hcsshim to fix nancy alert.
	github.com/Microsoft/hcsshim v0.8.7 => github.com/Microsoft/hcsshim v0.8.10
	// Apply fix for CVE-2020-15114 not yet released in github.com/spf13/viper.
	github.com/bketelsen/crypt => github.com/bketelsen/crypt v0.0.3
	// Use v1.4.2 of gorilla/websocket to fix nancy alert.
	github.com/gorilla/websocket v1.4.0 => github.com/gorilla/websocket v1.4.2
	// Use v1.0.0-rc7 of runc to fix nancy alert.
	github.com/opencontainers/runc v0.1.1 => github.com/opencontainers/runc v1.0.0-rc7
	// Use v1.7.1 of viper to fix nancy alert.
	github.com/spf13/viper => github.com/spf13/viper v1.7.1
	// Use fork of CAPI with Kubernetes 1.18 support.
	sigs.k8s.io/cluster-api => github.com/giantswarm/cluster-api v0.3.10-gs
)
