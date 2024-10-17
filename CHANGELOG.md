# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed

- Fixed logic fetching Releases for cluster apps to work with cross-provider MCs

## [0.26.0] - 2024-06-27

### Fixed

- Fix cluster app mutation, so it merges all config before trying to read release version.

## [0.25.0] - 2024-06-11

### Added

- Support for CAPI workload clusters that use Release resources. See [RFC](https://github.com/giantswarm/rfc/pull/96) for more details.

## [0.24.3] - 2024-04-23

## [0.24.2] - 2024-02-15

### Changed

- The cluster values ConfigMap is from now on injected into extra configs and is obligatory for workload cluster apps.
- Use base image from `gsoci.azurecr.io`
- Update api version of `PodDisruptionBudget` to `v1` for k8s 1.25.

## [0.24.1] - 2024-01-29

### Fixed

- Move pss values under the global property

## [0.24.0] - 2023-12-12

### Changed

- Enable app mutation for disabling PSPs in CAPI clusters.

## [0.23.1] - 2023-12-05

### Changed

- Configure `gsoci.azurecr.io` as the default container image registry.

## [0.23.0] - 2023-11-17

### Changed

- Add patch updating labels to match `pss-operator`.
- Add a switch for PSP CR installation.

## [0.22.0] - 2023-10-20

### Added

- Add customized overrides for PSP removal.

## [0.21.1] - 2023-10-17

### Changed

- Limit PSP removal logic to vintage providers.

## [0.21.0] - 2023-10-13

### Added

- Mutate Vintage Workload Cluster Apps to disable Pod Security Policies.

### Removed

- Removed duplicate of Github pre-commit workflow.

## [0.20.0] - 2023-08-15

### Changed

- Bump `github.com/giantswarm/apptest` to `v1.2.1`
- Bump `architect-orb` to `v4.31.0`
- Bump `app` to `v7.0.0`, see: https://github.com/giantswarm/app/pull/294

### Removed

- Mutate App: Remove `nginx-ingress-controller-app` exception.

## [0.19.0] - 2023-07-04

### Changed

- Updated default `securityContext` values to comply with PSS policies.

## [0.18.7] - 2023-06-02

### Added

- Add service monitor to be scraped by Prometheus Agent.

### Removed

- Remove push to `shared-app-collection` as it is deprecated.
- Stop pushing to `openstack-app-collection\`.

## [0.18.6] - 2023-04-05

### Changed

- Bump `giantswarm/app` package to `v6.15.6`

## [0.18.5] - 2023-03-10

### Changed

- Bump `giantswarm/app` package to `v6.15.5`
-

## [0.18.4] - 2023-03-09

### Changed

- Bump `giantswarm/app` package to `v6.15.3` to fix cluster and namespace singletons checks for CAPI.

## [0.18.3] - 2023-02-02

### Changed

- Bump `giantswarm/app` package to `v6.15.2` to weaken the condition on userConfig names for default apps.

### Changed

- Add the use of the runtime/default seccomp profile
- Set 60 seconds timeout for serve TLS and metrics endpoints to potential Slowloris Attack vector reported by `golangcli-lint`

## [0.18.2] - 2022-11-21

### Changed

- Bump `giantswarm/app` library to `v6.15.1` to fix cluster singleton.

## [0.18.1] - 2022-09-13

## [0.18.0] - 2022-08-25

### Changed

- Bump `giantswarm/app` library to `v6.13.0` that contains a new App CR validator for unique in-cluster app names

## [0.17.2] - 2022-07-12

### Added

- Extend security validation logic to inspect referenced namespaces in `spec.extraConfigs` entries of App CR

## [0.17.1] - 2022-06-10

### Added

- Extend security validation logic with checking the `kubeconfig` namespace against blacklisted namespaces.

## [0.17.0] - 2022-05-27

### Added

- Run additional validation logic for submitting the `0.0.0`-labeld App CRs by a non privileged users.

## [0.16.3] - 2022-02-25

### Fixed

- Remove compatible providers validation for `AppCatalogEntry` as its overly strict.
- Push image to Docker Hub to not rely on crsync.

## [0.16.2] - 2022-02-09

## [0.16.1] - 2022-01-26

### Changed

- Adapt mutation logic for `giantswarm.io/cluster` label to support App CRs in the org namespace.

## [0.16.0] - 2022-01-24

### Added

- Add support for validating `giantswarm.io/cluster` label for org-namespaced App CRs.

## [0.15.1] - 2022-01-20

### Fixed

- Remove `key.IsManagedByFlux` check and move it to `github.com/giantswarm/app` package.

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

[Unreleased]: https://github.com/giantswarm/app-admission-controller/compare/v0.26.0...HEAD
[0.26.0]: https://github.com/giantswarm/app-admission-controller/compare/v0.25.0...v0.26.0
[0.25.0]: https://github.com/giantswarm/app-admission-controller/compare/v0.24.3...v0.25.0
[0.24.3]: https://github.com/giantswarm/app-admission-controller/compare/v0.24.2...v0.24.3
[0.24.2]: https://github.com/giantswarm/app-admission-controller/compare/v0.24.1...v0.24.2
[0.24.1]: https://github.com/giantswarm/app-admission-controller/compare/v0.24.0...v0.24.1
[0.24.0]: https://github.com/giantswarm/app-admission-controller/compare/v0.23.1...v0.24.0
[0.23.1]: https://github.com/giantswarm/app-admission-controller/compare/v0.23.0...v0.23.1
[0.23.0]: https://github.com/giantswarm/app-admission-controller/compare/v0.22.0...v0.23.0
[0.22.0]: https://github.com/giantswarm/app-admission-controller/compare/v0.21.1...v0.22.0
[0.21.1]: https://github.com/giantswarm/app-admission-controller/compare/v0.21.0...v0.21.1
[0.21.0]: https://github.com/giantswarm/app-admission-controller/compare/v0.20.0...v0.21.0
[0.20.0]: https://github.com/giantswarm/app-admission-controller/compare/v0.19.0...v0.20.0
[0.19.0]: https://github.com/giantswarm/app-admission-controller/compare/v0.18.7...v0.19.0
[0.18.7]: https://github.com/giantswarm/app-admission-controller/compare/v0.18.6...v0.18.7
[0.18.6]: https://github.com/giantswarm/app-admission-controller/compare/v0.18.5...v0.18.6
[0.18.5]: https://github.com/giantswarm/app-admission-controller/compare/v0.18.4...v0.18.5
[0.18.4]: https://github.com/giantswarm/app-admission-controller/compare/v0.18.3...v0.18.4
[0.18.3]: https://github.com/giantswarm/app-admission-controller/compare/v0.18.2...v0.18.3
[0.18.2]: https://github.com/giantswarm/app-admission-controller/compare/v0.18.1...v0.18.2
[0.18.1]: https://github.com/giantswarm/app-admission-controller/compare/v0.18.0...v0.18.1
[0.18.0]: https://github.com/giantswarm/app-admission-controller/compare/v0.17.2...v0.18.0
[0.17.2]: https://github.com/giantswarm/app-admission-controller/compare/v0.17.1...v0.17.2
[0.17.1]: https://github.com/giantswarm/app-admission-controller/compare/v0.17.0...v0.17.1
[0.17.0]: https://github.com/giantswarm/app-admission-controller/compare/v0.16.3...v0.17.0
[0.16.3]: https://github.com/giantswarm/app-admission-controller/compare/v0.16.2...v0.16.3
[0.16.2]: https://github.com/giantswarm/app-admission-controller/compare/v0.16.1...v0.16.2
[0.16.1]: https://github.com/giantswarm/app-admission-controller/compare/v0.16.0...v0.16.1
[0.16.0]: https://github.com/giantswarm/app-admission-controller/compare/v0.15.1...v0.16.0
[0.15.1]: https://github.com/giantswarm/app-admission-controller/compare/v0.15.0...v0.15.1
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
