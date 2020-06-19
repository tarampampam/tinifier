# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog][keepachangelog] and this project adheres to [Semantic Versioning][semver].

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
