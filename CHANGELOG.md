# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog][keepachangelog] and this project adheres to [Semantic Versioning][semver].

## v0.0.3

### Fixed

- Threads count now depends on total files count _(threads count cannot be grater then total files count)_

## v0.0.2

### Changed

- Docker image now uses `entrypoint` instead `command`

### Fixed

- Binary filename in travis config for windows

## v0.0.1

### Added

- Basic features (like threads end errors handling)

[keepachangelog]:https://keepachangelog.com/en/1.0.0/
[semver]:https://semver.org/spec/v2.0.0.html
