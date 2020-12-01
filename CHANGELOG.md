# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).



## [Unreleased]

## [0.1.0] - 2020-12-01

### Added

- Add mutation webhook with defaulting logic that is enabled for app CRs with
`app-operator.giantswarm.io/version` label value >= `3.0.0`.
- Add validation webhook that is enabled for app CRs with
`app-operator.giantswarm.io/version` label value >= `3.0.0`.

[Unreleased]: https://github.com/giantswarm/app-admission-controller/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/giantswarm/app-admission-controller/releases/tag/v0.1.0
