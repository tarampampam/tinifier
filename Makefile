#!/usr/bin/make
# Makefile readme (ru): <http://linux.yaroslavl.ru/docs/prog/gnu_make_3-79_russian_manual.html>
# Makefile readme (en): <https://www.gnu.org/software/make/manual/html_node/index.html#SEC_Contents>

SHELL = /bin/sh
# GOOS can be 'android darwin dragonfly freebsd linux nacl netbsd openbsd plan9 solaris windows'
GOOS = linux
# GOARCH can be '386 amd64 amd64p32 arm arm64 ppc64 ppc64le mips mipsle mips64 mips64le mips64p32 mips64p32le ppc s390 s390x sparc sparc64'
GOARCH = amd64
# LDFLAGS flags help: 'go help build'
LDFLAGS = "-s -w"
# GETFLAGS flags help: 'go help get'
GETFLAGS = -t -v -u

DOCKER_BIN = $(shell command -v docker 2> /dev/null)
DC_BIN = $(shell command -v docker-compose 2> /dev/null)
DC_RUN_ARGS = --rm --user "$(shell id -u):$(shell id -g)" app
APP_NAME = $(notdir $(CURDIR))

.PHONY : help build test clean image
.DEFAULT_GOAL : help

# This will output the help for each task. thanks to https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
help: ## Show this help
	@printf "\033[33m%s:\033[0m\n" 'Available commands'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[32m%-11s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

#deps: ## Install all build dependencies
#	$(CC) get $(GETFLAGS) ./...

build: ## Build app binary file
	$(DC_BIN) run -e "GOARCH=$(GOARCH)" -e "GOOS=$(GOOS)" $(DC_RUN_ARGS) \
		go build -ldflags=$(LDFLAGS) -o '/build/$(APP_NAME)' ./main.go

.SILENT:
test: ## Run app tests
	printf "\033[33m %s \033[0m\n" 'Start gofmt..'
	$(DC_BIN) run $(DC_RUN_ARGS) sh -c 'test -z "$$(gofmt -d -e .)"'

.SILENT:
shell: ## Start shell into container with golang
	$(DC_BIN) run $(DC_RUN_ARGS) sh

image: ## Build docker image with app
	$(DOCKER_BIN) build -f ./Dockerfile -t $(APP_NAME) .

clean: ## Make clean
	$(DC_BIN) down -v -t 1
	$(DOCKER_BIN) rmi $(APP_NAME) -f
