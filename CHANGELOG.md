# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog][keepachangelog] and this project adheres to [Semantic Versioning][semver].

## UNRELEASED

### Changed

- Go version updated from `1.16.3` up to `1.17.3`

## v3.3.0

### Added

- Global flag `--debug`

### Changed

- Go version updated from `1.16.2` up to `1.16.3`

## v3.2.0

### Added

- Support for `linux/386`, `linux/arm64`, `linux/arm/v6` and `linux/arm/v7` platforms for docker image

## v3.1.0

### Changed

- Go version updated from `1.15.7` up to `1.16.0`

## v3.0.1

### Fixed

- Application version setting in CI

## v3.0.0

### Added

- Multiple keys support for `compress` command
- Flag `--max-errors` _(maximum errors count to stop the process)_ for `compress` command
- Flag `--max-key-errors` _(maximum API key errors (compression retries) to disable the key)_ for `compress` command
- Flag `--recursive` (`-r`) _(search for files in listed directories recursively)_ for `compress` command
- Compression retries
- Automated 3rd party (like [goreportcard.com](https://goreportcard.com/) and [pkg.go.dev](https://pkg.go.dev/)) service updates

### Changed

- Source code totally re-wrote
- Dependency `github.com/jessevdk/go-flags` replaced with `github.com/spf13/cobra`
- Package `github.com/tarampampam/tinifier/v3/pkg/tinypng` can be installed into another go-applications
- Go version updated from `1.14.x` up to `1.15.6`

### Removed

- Progress bar (package `github.com/schollz/progressbar/v3`)
- Binary file packing (using `upx`)

### Fixed

- Errors handling in a lot of places

## v2.1.0

### Added

- Total processed images count showing [#12]

[#12]:https://github.com/tarampampam/tinifier/issues/12

## v2.0.0

### Changed

- Application was totally re-wrote
- For different actions (like get version value, images compressing and used quota getting) was made separate sub-commands
- For running application in docker-container uses unprivileged user

### Added

- Sub-command `quota` for currently used quota getting
- Sub-command `version` for application version displaying
- Sub-command `compress` for images compressing

### Removed

- Options `-v (verbose)`, `-V (version)`, `-q (quota)`, `-C (no-colors)` and `-m (max-errors)`

## v1.0.2

### Added

- MacOS build

## v1.0.1

### Fixed

- Build for windows

## v1.0.0

### Changed

- Source code decomposited _(partially)_
- On pressing `Ctrl + C` _(while process is in progress)_ results will be printed
- Migrated from Travis CI to GitHub actions
- Binary file names for releases _(for windows)_

### Added

- Code tests

### Fixed

- Option `-C` now works correctly

## v0.1.0

### Changed

- Source code decomposited
- Progress now shows in minimalistic style
- Binary file names for releases

### Added

- Option `-q` (`--quota`) for requesting current quota usage
- Option `-C` (`--no-colors`) for disabling color output
- Option `-m` (`--max-errors`) for setting maximum possible errors count
- For verbose option (`-v`) added long alias `--verbose`

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
