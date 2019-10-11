<p align="center">
  <img src="https://tinypng.com/images/apng/panda-waving.png" alt="Logo" width="128" />
</p>

# :panda_face: CLI tool for images compressing

![Release version][badge_release_version]
![Project language][badge_language]
[![Build Status][badge_build]][link_build]
[![Coverage][badge_coverage]][link_coverage]
[![Image size][badge_size_latest]][link_docker_build]
[![License][badge_license]][link_license]

This tool uses [tinypng.com][tinypng.com] API endpoint for compressing your local jpg/png images (multi-threads, of course):

```
Usage:
  tinifier [OPTIONS] files-and-directories...

Application Options:
  -v                           Show verbose debug information
  -V, --version                Show version and exit
  -e, --ext=                   Target file extensions (default: jpg,JPG,jpeg,JPEG,png,PNG)
  -k, --api-key=               API key <https://tinypng.com/dashboard/api> [$TINYPNG_API_KEY]
  -t, --threads=               Threads processing count (default: 5)

Help Options:
  -h, --help                   Show this help message
```

> API key can be passed in environment variable named `TINYPNG_API_KEY`

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
$ docker run --rm -v "$(pwd):/rootfs:rw" -w /rootfs tarampampam/tinifier -k 'YOUR-API-KEY-GOES-HERE' .
```

### :star2: Testing

For application testing we use built-in golang testing feature and `docker-ce` + `docker-compose` as develop environment. So, just write into your terminal after repository cloning:

```shell
$ make test
```

## :notebook: Changes log

[![Release date][badge_release_date]][link_releases]
[![Commits since latest release][badge_commits_since_release]][link_commits]

Changes log can be [found here][link_changes_log].

## :ambulance: Support

[![Issues][badge_issues]][link_issues]
[![Issues][badge_pulls]][link_pulls]

If you will find any package errors, please, [make an issue][link_create_issue] in current repository.

## :eyes: License

This is open-sourced software licensed under the [MIT License][link_license].

[badge_build]:https://img.shields.io/travis/com/tarampampam/tinifier/master.svg?maxAge=10
[badge_coverage]:https://img.shields.io/codecov/c/github/tarampampam/tinifier/master.svg?maxAge=30
[badge_size_latest]:https://images.microbadger.com/badges/image/tarampampam/tinifier.svg
[badge_release_version]:https://img.shields.io/github/release/tarampampam/tinifier.svg?maxAge=30
[badge_language]:https://img.shields.io/badge/language-go_1.13-blue.svg?longCache=true
[badge_license]:https://img.shields.io/github/license/tarampampam/tinifier.svg?longCache=true
[badge_release_date]:https://img.shields.io/github/release-date/tarampampam/tinifier.svg?maxAge=180
[badge_commits_since_release]:https://img.shields.io/github/commits-since/tarampampam/tinifier/latest.svg?maxAge=45
[badge_issues]:https://img.shields.io/github/issues/tarampampam/tinifier.svg?maxAge=45
[badge_pulls]:https://img.shields.io/github/issues-pr/tarampampam/tinifier.svg?maxAge=45

[link_build]:https://travis-ci.com/tarampampam/tinifier
[link_coverage]:https://codecov.io/gh/tarampampam/tinifier
[link_docker_build]:https://hub.docker.com/r/tarampampam/tinifier/
[link_license]:https://github.com/tarampampam/tinifier/blob/master/LICENSE
[link_releases]:https://github.com/tarampampam/tinifier/releases
[link_commits]:https://github.com/tarampampam/tinifier/commits
[link_changes_log]:https://github.com/tarampampam/tinifier/blob/master/CHANGELOG.md
[link_issues]:https://github.com/tarampampam/tinifier/issues
[link_create_issue]:https://github.com/tarampampam/tinifier/issues/new/choose
[link_pulls]:https://github.com/tarampampam/tinifier/pulls

[tinypng.com]:https://tinypng.com/
