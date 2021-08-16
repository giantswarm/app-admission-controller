module github.com/giantswarm/app-admission-controller

go 1.16

require (
	github.com/Masterminds/semver/v3 v3.1.1
	github.com/dyson/certman v0.2.1
	github.com/giantswarm/apiextensions/v3 v3.32.0
	github.com/giantswarm/app/v5 v5.2.1-0.20210816133125-1f209949d008
	github.com/giantswarm/apptest v0.11.0
	github.com/giantswarm/backoff v0.2.0
	github.com/giantswarm/k8sclient/v5 v5.12.0
	github.com/giantswarm/k8smetadata v0.3.0
	github.com/giantswarm/microerror v0.3.0
	github.com/giantswarm/micrologger v0.5.0
	github.com/google/go-cmp v0.5.6
	github.com/prometheus/client_golang v1.11.0
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	k8s.io/api v0.18.19
	k8s.io/apimachinery v0.18.19
	k8s.io/client-go v0.18.19
)

replace (
	// Use v0.8.10 of hcsshim to fix nancy alert.
	github.com/Microsoft/hcsshim v0.8.7 => github.com/Microsoft/hcsshim v0.8.10
	// Apply fix for CVE-2020-15114 not yet released in github.com/spf13/viper.
	github.com/bketelsen/crypt => github.com/bketelsen/crypt v0.0.3
	github.com/dgrijalva/jwt-go => github.com/dgrijalva/jwt-go/v4 v4.0.0-preview1
	// Use go-logr/logr v0.1.0 due to breaking changes in v0.2.0 that can't be applied.
	github.com/go-logr/logr v0.2.0 => github.com/go-logr/logr v0.1.0
	// Use v1.3.2 of gogo/protobuf to fix nancy alert.
	github.com/gogo/protobuf v1.3.1 => github.com/gogo/protobuf v1.3.2
	// Use v1.4.2 of gorilla/websocket to fix nancy alert.
	github.com/gorilla/websocket v1.4.0 => github.com/gorilla/websocket v1.4.2
	// Use v1.0.0-rc7 of runc to fix nancy alert.
	github.com/opencontainers/runc v0.1.1 => github.com/opencontainers/runc v1.0.0-rc7
	// Use v1.7.1 of viper to fix nancy alert.
	github.com/spf13/viper => github.com/spf13/viper v1.7.1
	// Same as go-logr/logr, klog/v2 is using logr v0.2.0
	k8s.io/klog/v2 v2.4.0 => k8s.io/klog/v2 v2.0.0
	// Use fork of CAPI with Kubernetes 1.18 support.
	sigs.k8s.io/cluster-api => github.com/giantswarm/cluster-api v0.3.10-gs
)
