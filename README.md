<p align="center">
  <img src="https://tinypng.com/images/apng/panda-waving.png" alt="Logo" width="128" />
</p>

# CLI tool for images compressing

![Release version][badge_release_version]
![Project language][badge_language]
[![Build Status][badge_build]][link_build]
[![Coverage][badge_coverage]][link_coverage]
[![Go Report][badge_goreport]][link_goreport]
[![Image size][badge_size_latest]][link_docker_build]
[![License][badge_license]][link_license]

This tool uses [tinypng.com][tinypng.com] API endpoint for compressing your local jpg/png images (it supports parallel jobs):

<p align="center">
    <a href="https://asciinema.org/a/LGL0Brjm0YzRPs7TcyXVTCSXz" target="_blank"><img src="https://asciinema.org/a/LGL0Brjm0YzRPs7TcyXVTCSXz.svg" width="900"></a>
</p>

> API key can be passed in environment variable named `TINYPNG_API_KEY`

> **Recursive (deep) directories walking is not supported**

## :fire: Usage example

> [tinypng.com][tinypng.com] API key is required. You get own API key by pressing link "login" on service main page.

Compress single image:

```bash
$ tinifier -k 'YOUR-API-KEY-GOES-HERE' ./image.png
```

Compress all `png` images in some directory and 2 another images:

```bash
$ tinifier -k 'YOUR-API-KEY-GOES-HERE' -e png ./images-directory ./image-1.png ./image-2.png
```

Compress all images in some directory using 20 threads:

```bash
$ tinifier -k 'YOUR-API-KEY-GOES-HERE' -e png -e jpg -e PNG,JPG -t 20 ./images-directory
```

## :whale: Using docker

Compress all images in **current** directory:

```bash
$ docker run --rm \
    -v "$(pwd):/rootfs:rw" \
    -w /rootfs \
    tarampampam/tinifier -k 'YOUR-API-KEY-GOES-HERE' .
```

or 

```bash
$ docker run --rm \
    -v "$(pwd):/rootfs:rw" \
    -w /rootfs \
    -e 'TINYPNG_API_KEY=YOUR-API-KEY-GOES-HERE' \
    tarampampam/tinifier .
```

### Testing

For application testing we use built-in golang testing feature and `docker-ce` + `docker-compose` as develop environment. So, just write into your terminal after repository cloning:

```shell
$ make test
```

Or build binary file:

```shell
$ make build
```

## Changes log

[![Release date][badge_release_date]][link_releases]
[![Commits since latest release][badge_commits_since_release]][link_commits]

Changes log can be [found here][link_changes_log].

## Support

[![Issues][badge_issues]][link_issues]
[![Issues][badge_pulls]][link_pulls]

If you will find any package errors, please, [make an issue][link_create_issue] in current repository.

## License

This is open-sourced software licensed under the [MIT License][link_license].

[badge_build]:https://github.com/actions/tarampampam/workflows/Build%20on%20Push/badge.svg
[badge_coverage]:https://img.shields.io/codecov/c/github/tarampampam/tinifier/master.svg?maxAge=30
[badge_goreport]:https://goreportcard.com/badge/github.com/tarampampam/tinifier
[badge_size_latest]:https://images.microbadger.com/badges/image/tarampampam/tinifier.svg
[badge_release_version]:https://img.shields.io/github/release/tarampampam/tinifier.svg?maxAge=30
[badge_language]:https://img.shields.io/badge/language-go_1.13-blue.svg?longCache=true
[badge_license]:https://img.shields.io/github/license/tarampampam/tinifier.svg?longCache=true
[badge_release_date]:https://img.shields.io/github/release-date/tarampampam/tinifier.svg?maxAge=180
[badge_commits_since_release]:https://img.shields.io/github/commits-since/tarampampam/tinifier/latest.svg?maxAge=45
[badge_issues]:https://img.shields.io/github/issues/tarampampam/tinifier.svg?maxAge=45
[badge_pulls]:https://img.shields.io/github/issues-pr/tarampampam/tinifier.svg?maxAge=45
[link_goreport]:https://goreportcard.com/report/github.com/tarampampam/tinifier

[link_coverage]:https://codecov.io/gh/tarampampam/tinifier
[link_build]:https://github.com/tarampampam/tinifier/actions
[link_docker_build]:https://hub.docker.com/r/tarampampam/tinifier/
[link_license]:https://github.com/tarampampam/tinifier/blob/master/LICENSE
[link_releases]:https://github.com/tarampampam/tinifier/releases
[link_commits]:https://github.com/tarampampam/tinifier/commits
[link_changes_log]:https://github.com/tarampampam/tinifier/blob/master/CHANGELOG.md
[link_issues]:https://github.com/tarampampam/tinifier/issues
[link_create_issue]:https://github.com/tarampampam/tinifier/issues/new/choose
[link_pulls]:https://github.com/tarampampam/tinifier/pulls

[tinypng.com]:https://tinypng.com/
