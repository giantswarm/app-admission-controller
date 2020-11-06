module github.com/giantswarm/app-admission-controller

go 1.15

require (
	github.com/giantswarm/apiextensions/v3 v3.6.0
	github.com/giantswarm/app/v3 v3.1.0
	github.com/giantswarm/apptest v0.4.2-0.20201106073136-b245f0bda36f
	github.com/giantswarm/backoff v0.2.0
	github.com/giantswarm/k8sclient/v5 v5.0.0
	github.com/giantswarm/microerror v0.2.1
	github.com/giantswarm/micrologger v0.3.3
	github.com/prometheus/client_golang v1.8.0
	golang.org/x/tools v0.0.0-20200706234117-b22de6825cf7 // indirect
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	k8s.io/api v0.18.9
	k8s.io/apimachinery v0.18.9
	k8s.io/client-go v0.18.9
)

replace (
	// Use v0.8.10 of hcsshim to fix nancy alert.
	github.com/Microsoft/hcsshim v0.8.7 => github.com/Microsoft/hcsshim v0.8.10
	// Apply fix for CVE-2020-15114 not yet released in github.com/spf13/viper.
	github.com/bketelsen/crypt => github.com/bketelsen/crypt v0.0.3
	// Apply security fix not present in v1.4.0.
	github.com/gorilla/websocket => github.com/gorilla/websocket v1.4.2
	// Use v1.0.0-rc7 of runc to fix nancy alert.
	github.com/opencontainers/runc v0.1.1 => github.com/opencontainers/runc v1.0.0-rc7
	// Apply security fix not present in 1.6.2.
	github.com/spf13/viper => github.com/spf13/viper v1.7.1
	// Use fork of CAPI with Kubernetes 1.18 support.
	sigs.k8s.io/cluster-api => github.com/giantswarm/cluster-api v0.3.10-gs
)
