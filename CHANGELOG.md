# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/giantswarm/app-admission-controller/compare/v0.4.1...HEAD
[0.4.1]: https://github.com/giantswarm/app-admission-controller/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/giantswarm/app-admission-controller/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/giantswarm/app-admission-controller/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/giantswarm/app-admission-controller/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/giantswarm/app-admission-controller/releases/tag/v0.1.0
