# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.15.0] - 2022-01-13

### Added

- Ensure users are not allowed to create in-cluster Apps outside the org- and WC-related namespaces.

## [0.14.0] - 2021-12-21

### Added

- Support for App CRs with a `v` prefixed version. This enables Flux to automatically update the version based on its image tag.
- Ensure `.spec.Namespace` in App CRs is immutable.

### Changed

- Use apiextensions-application instead of apiextensions for CRDs to remove CAPI dependency.

## [0.13.0] - 2021-11-25

### Changed

- Skip validation of configmap and secret names when `giantswarm.io/managedby`
label is set to flux.

## [0.12.0] - 2021-09-13

### Added

- Add support for defaulting the kubeconfig secret for CAPI clusters.

### Fixed

- Don't restrict user values configmap name for NGINX Ingress Controller

## [0.11.0] - 2021-08-17

### Changed

- Always default app CR labels so we can retire the legacy `1.0.0` version label.
- Validate `.spec.catalog` using Catalog CRs instead of AppCatalog CRs.

## [0.10.1] - 2021-06-16

### Changed

- Prepare helm values to configuration management.
- Update architect-orb to v3.0.0.

## [0.10.0] - 2021-04-30

### Added

- Emit events when App CR version is updated.

## [0.9.0] - 2021-04-19

### Added

- Add validation that `.metadata.name` is not longer than 53 chars due to limit
on the length of Helm release names.

### Removed

- Stop adding obsolete `config-controller.giantswarm.io/version` label on App
  CRs reconciled by unique app-operator.

## [0.8.0] - 2021-03-29

### Added

- Add validation for user configmap and secret names for apps in the default catalog.

### Changed

- Update TLS minimum version as 1.2.

## [0.7.0] - 2021-03-19

### Added

- Add `namespaceConfig` validation.

## [0.6.0] - 2021-03-04

### Added

- Apply `compatibleProvider`,`namespace` metadata validation based on the relevant `AppCatalogEntry` CR.

### Fixed

- Don't default `.spec.config` if app is a Management Cluster app.

## [0.5.1] - 2021-02-17

### Fixed

- Support both `v1` and `v1beta1` for admission requests.

## [0.5.0] - 2021-02-16

### Changed

- Update `k8s.io/api/admission` to v1.

### Deleted

- Remove setting `pause` annotation on `App` CR.

## [0.4.1] - 2021-01-14

### Fixed

- Pause Management Cluster app reconciliation when App CR version is updated.

## [0.4.0] - 2021-01-12

### Added

- Add `app-operator.giantswarm.io/paused: "true"` annotation to newly created
  App CRs installing apps running in Giant Swarm Management Cluster.
- Add `config-controller.giantswarm.io/version: "0.0.0"` label to newly created
  App CRs installing apps running in Giant Swarm Management Cluster.

### Changed

- Update cert apiVersion to v1.
- Update `giantswarm/app` to `v4.2.0`.

## [0.3.0] - 2021-01-11

### Added

- Add support for reloading certs when they expire.

### Changed

- Fail when mutation review request fails. So far mutation review failures were
  ignored.

## [0.2.0] - 2020-12-15

### Added

- Add to app collections for all providers.
- Enable admission controller logic for unique app CRs.
- Update to v4 of app library and skip if app dependencies are not ready to
allow app CR creation.

## [0.1.0] - 2020-12-01

### Added

- Add mutation webhook with defaulting logic that is enabled for app CRs with
`app-operator.giantswarm.io/version` label value >= `3.0.0`.
- Add validation webhook that is enabled for app CRs with
`app-operator.giantswarm.io/version` label value >= `3.0.0`.

[Unreleased]: https://github.com/giantswarm/app-admission-controller/compare/v0.15.0...HEAD
[0.15.0]: https://github.com/giantswarm/app-admission-controller/compare/v0.14.0...v0.15.0
[0.14.0]: https://github.com/giantswarm/app-admission-controller/compare/v0.13.0...v0.14.0
[0.13.0]: https://github.com/giantswarm/app-admission-controller/compare/v0.12.0...v0.13.0
[0.12.0]: https://github.com/giantswarm/app-admission-controller/compare/v0.11.0...v0.12.0
[0.11.0]: https://github.com/giantswarm/app-admission-controller/compare/v0.10.1...v0.11.0
[0.10.1]: https://github.com/giantswarm/app-admission-controller/compare/v0.10.0...v0.10.1
[0.10.0]: https://github.com/giantswarm/app-admission-controller/compare/v0.9.0...v0.10.0
[0.9.0]: https://github.com/giantswarm/app-admission-controller/compare/v0.8.0...v0.9.0
[0.8.0]: https://github.com/giantswarm/app-admission-controller/compare/v0.7.0...v0.8.0
[0.7.0]: https://github.com/giantswarm/app-admission-controller/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/giantswarm/app-admission-controller/compare/v0.5.1...v0.6.0
[0.5.1]: https://github.com/giantswarm/app-admission-controller/compare/v0.5.0...v0.5.1
[0.5.0]: https://github.com/giantswarm/app-admission-controller/compare/v0.4.1...v0.5.0
[0.4.1]: https://github.com/giantswarm/app-admission-controller/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/giantswarm/app-admission-controller/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/giantswarm/app-admission-controller/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/giantswarm/app-admission-controller/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/giantswarm/app-admission-controller/releases/tag/v0.1.0
