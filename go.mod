module github.com/giantswarm/app-admission-controller

go 1.15

require (
	github.com/giantswarm/apiextensions/v3 v3.6.0
	github.com/giantswarm/app/v3 v3.1.0
	github.com/giantswarm/apptest v0.5.0
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

replace sigs.k8s.io/cluster-api => github.com/giantswarm/cluster-api v0.3.10-gs
