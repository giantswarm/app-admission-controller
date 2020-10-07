[![CircleCI](https://circleci.com/gh/giantswarm/app-admission-controller.svg?style=shield)](https://circleci.com/gh/giantswarm/app-admission-controller)

# app-admission-controller (Under Contruction)

Admission controller that implements the following rules:

## Mutating Webhook:

- TODO: Add details.

## Validating Webhook:

- TODO: Add details.

## Ownership

Team Batman

### Local Development

Testing the app-admission-controller in a kind cluster on your local machine:

```nohighlight
kind create cluster

# Build a linux image
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build .
docker build . -t app-admission-controller:dev
kind load docker-image app-admission-controller:dev

# Make sure the Custom Resource Definitions are in place
opsctl ensure crds -k "$(kind get kubeconfig)" -p aws

# Insert the certificate
kubectl apply --context kind-kind -f local_dev/certmanager.yml

## Wait until certmanager is up

kubectl apply --context kind-kind -f local_dev/clusterissuer.yml
helm template app-admission-controller -f helm/app-admission-controller/ci/default-values.yaml helm/app-admission-controller > local_dev/deploy.yaml

## Replace image name with app-admission-controller:dev
kubectl apply --context kind-kind -f local_dev/deploy.yaml
kind delete cluster
```

## Changelog

See [Releases](https://github.com/giantswarm/app-admission-controller/releases)

## Contact

- Bugs: [issues](https://github.com/giantswarm/app-admission-controller/issues)
- Please visit https://www.giantswarm.io/responsible-disclosure for information on reporting security issues.

## Contributing, reporting bugs

See [CONTRIBUTING](CONTRIBUTING.md) for details on submitting patches, the
contribution workflow as well as reporting bugs.

## Add a new webhook

See [docs/webhook.md](https://github.com/giantswarm/app-admission-controller/blob/master/docs/webhook.md)

## Writing tests

See [docs/tests.md](https://github.com/giantswarm/app-admission-controller/blob/master/docs/tests.md)
