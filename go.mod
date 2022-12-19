module github.com/giantswarm/app-admission-controller

go 1.16

require (
	github.com/Masterminds/semver/v3 v3.1.1
	github.com/dyson/certman v0.2.1
	github.com/giantswarm/apiextensions-application v0.6.0
	github.com/giantswarm/app/v6 v6.15.1
	github.com/giantswarm/apptest v1.0.1
	github.com/giantswarm/backoff v1.0.0
	github.com/giantswarm/k8sclient/v6 v6.1.0
	github.com/giantswarm/k8smetadata v0.13.0
	github.com/giantswarm/microerror v0.4.0
	github.com/giantswarm/micrologger v0.6.0
	github.com/google/go-cmp v0.5.9
	github.com/prometheus/client_golang v1.12.1
	github.com/stretchr/testify v1.7.2 // indirect
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	k8s.io/api v0.21.4
	k8s.io/apimachinery v0.21.4
	k8s.io/client-go v0.21.4
	sigs.k8s.io/controller-runtime v0.9.7
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
	// To solve CVE-2020-36565
	github.com/labstack/echo/v4 v4.1.11 => github.com/labstack/echo/v4 v4.9.1
	// Fix CVE
	github.com/nats-io/nats-server/v2 => github.com/nats-io/nats-server/v2 v2.9.3
	// Use v1.0.0-rc7 of runc to fix nancy alert.
	github.com/opencontainers/runc v0.1.1 => github.com/opencontainers/runc v1.0.0-rc7
	// Use v1.7.1 of viper to fix nancy alert.
	github.com/spf13/viper => github.com/spf13/viper v1.7.1
	// To solve CVE-2022-41717
	golang.org/x/net v0.2.0 => golang.org/x/net v0.4.0
	// Fix CVE
	golang.org/x/text => golang.org/x/text v0.3.8
	// Same as go-logr/logr, klog/v2 is using logr v0.2.0
	k8s.io/klog/v2 v2.4.0 => k8s.io/klog/v2 v2.0.0
)
