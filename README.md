<p align="center">
  <img src="https://tinypng.com/images/apng/panda-waving.png" alt="Logo" width="128" />
</p>

# CLI tool for images compressing

[![Release version][badge_release_version]][link_gopkg]
[![Project language][badge_language]][link_golang]
[![Build Status][badge_build]][link_actions]
[![Coverage][badge_coverage]][link_coverage]
[![Go Report][badge_goreport]][link_goreport]
[![Image size][badge_size_latest]][link_docker_hub]
[![License][badge_license]][link_license]

This tool uses [tinypng.com][tinypng.com] API endpoint for compressing your local jpg/png images (it supports parallel jobs):

<p align="center">
    <a href="https://asciinema.org/a/381354?autoplay=1" target="_blank"><img src="https://asciinema.org/a/381354.svg" width="900"></a>
</p>

> API key can be set using environment variable named `TINYPNG_API_KEY`

## :computer: Installing

Download latest binary file for your os/arch from [releases page][link_releases] or use our [docker image][link_docker_hub] ([ghcr.io][link_ghcr]).

## :fire: Usage example

> [tinypng.com][tinypng.com] API key is required. For API key getting you should:
> - Open [tinypng.com/developers](https://tinypng.com/developers)
> - Fill-up the form (enter your name and email address) and press "Get your API key" button
> - Check for email in the mailbox from previous step (click on "verification link")
> - In opened dashboard page - activate API key and save it somewhere

Compress single image:

```bash
$ tinifier compress -k 'YOUR-API-KEY-GOES-HERE' ./image.png
```

Compress all `png` images in some directory and 2 another images:

```bash
$ tinifier compress -k 'YOUR-API-KEY-GOES-HERE' -e png ./images-directory ./image-1.png ./image-2.png
```

Compress jpg/png images in some directory (recursively) using 20 threads:

```bash
$ tinifier compress -k 'YOUR-API-KEY-GOES-HERE' -e png -e jpg -e PNG -e JPG -t 20 -r ./images-directory
```

### :whale: Using docker

[![image stats](https://dockeri.co/image/tarampampam/tinifier)][link_docker_hub]

> All supported image tags [can be found here][link_docker_hub] and [here][link_ghcr].

Compress all images in **current** directory:

```bash
$ docker run --rm -ti \
    -u "$(id -u):$(id -g)" \
    -v "$(pwd):/rootfs:rw" \
    -w /rootfs \
    tarampampam/tinifier compress -k 'YOUR-API-KEY-GOES-HERE' -r .
```

or

```bash
$ docker run --rm -ti \
    -u "$(id -u):$(id -g)" \
    -v "$(pwd):/rootfs:rw" \
    -w /rootfs \
    -e 'TINYPNG_API_KEY=YOUR-API-KEY-GOES-HERE' \
    tarampampam/tinifier compress -r .
```

## Testing

For application testing and building we use built-in golang testing feature and `docker-ce` + `docker-compose` as develop environment. So, just write into your terminal after repository cloning:

```shell
$ make test
```

Or build the binary file:

```shell
$ make build
```

## Releasing

New versions publishing is very simple - just make required changes in this repository, update [changelog file](CHANGELOG.md) and "publish" new release using repo releases page.

Binary files and docker images will be build and published automatically.

> New release will overwrite the `latest` docker image tag in both registers.

## Changelog

[![Release date][badge_release_date]][link_releases]
[![Commits since latest release][badge_commits_since_release]][link_commits]

Changes log can be [found here][link_changes_log].

## Support

[![Issues][badge_issues]][link_issues]
[![Issues][badge_pulls]][link_pulls]
[![Chat][badge_discord]][link_discord]

If you will find any package errors, please, [make an issue][link_create_issue] in current repository.

## License

This is open-sourced software licensed under the [MIT License][link_license].

[badge_build]:https://img.shields.io/github/workflow/status/tarampampam/tinifier/tests/master
[badge_coverage]:https://img.shields.io/codecov/c/github/tarampampam/tinifier/master.svg?maxAge=30
[badge_goreport]:https://goreportcard.com/badge/github.com/tarampampam/tinifier
[badge_size_latest]:https://img.shields.io/docker/image-size/tarampampam/tinifier/latest?label=docker&maxAge=30
[badge_release_version]:https://img.shields.io/github/release/tarampampam/tinifier.svg?maxAge=30
[badge_language]:https://img.shields.io/github/go-mod/go-version/tarampampam/tinifier?longCache=true
[badge_license]:https://img.shields.io/github/license/tarampampam/tinifier.svg?longCache=true
[badge_release_date]:https://img.shields.io/github/release-date/tarampampam/tinifier.svg?maxAge=180
[badge_commits_since_release]:https://img.shields.io/github/commits-since/tarampampam/tinifier/latest.svg?maxAge=45
[badge_issues]:https://img.shields.io/github/issues/tarampampam/tinifier.svg?maxAge=45
[badge_pulls]:https://img.shields.io/github/issues-pr/tarampampam/tinifier.svg?maxAge=45
[badge_discord]:https://img.shields.io/discord/788484223563595837

[link_golang]:https://golang.org/
[link_goreport]:https://goreportcard.com/report/github.com/tarampampam/tinifier
[link_coverage]:https://codecov.io/gh/tarampampam/tinifier
[link_gopkg]:https://pkg.go.dev/github.com/tarampampam/tinifier/v3
[link_actions]:https://github.com/tarampampam/tinifier/actions
[link_docker_hub]:https://hub.docker.com/r/tarampampam/tinifier/
[link_ghcr]:https://github.com/users/tarampampam/packages/container/package/tinifier
[link_license]:https://github.com/tarampampam/tinifier/blob/master/LICENSE
[link_releases]:https://github.com/tarampampam/tinifier/releases
[link_commits]:https://github.com/tarampampam/tinifier/commits
[link_changes_log]:https://github.com/tarampampam/tinifier/blob/master/CHANGELOG.md
[link_issues]:https://github.com/tarampampam/tinifier/issues
[link_create_issue]:https://github.com/tarampampam/tinifier/issues/new/choose
[link_pulls]:https://github.com/tarampampam/tinifier/pulls
[link_discord]:https://discord.gg/pTAtuWzJbz

[tinypng.com]:https://tinypng.com/
