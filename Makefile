#!/usr/bin/make
# Makefile readme (ru): <http://linux.yaroslavl.ru/docs/prog/gnu_make_3-79_russian_manual.html>
# Makefile readme (en): <https://www.gnu.org/software/make/manual/html_node/index.html#SEC_Contents>

SHELL = /bin/sh
LDFLAGS = "-s -w -X github.com/tarampampam/tinifier/v4/internal/pkg/version.version=$(shell git rev-parse HEAD)"

DC_RUN_ARGS = --rm --user "$(shell id -u):$(shell id -g)"
APP_NAME = $(notdir $(CURDIR))

.PHONY : help \
         image build fmt lint gotest test cover \
         clean
.DEFAULT_GOAL : help
.SILENT : lint gotest

help: ## Show this help
	@printf "\033[33m%s:\033[0m\n" 'Available commands'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[32m%-11s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

image: ## Build docker image with app
	docker build -f ./Dockerfile -t $(APP_NAME):local .
	docker run --rm $(APP_NAME):local version
	@printf "\n   \e[30;42m %s \033[0m\n\n" 'Now you can use image like `docker run --rm $(APP_NAME):local ...`';

build: ## Build app binary file
	docker-compose run $(DC_RUN_ARGS) -e "CGO_ENABLED=0" --no-deps app go build -trimpath -ldflags $(LDFLAGS) -o ./tinifier ./cmd/tinifier/main.go

fmt: ## Run source code formatter tools
	docker-compose run $(DC_RUN_ARGS) -e "GO111MODULE=off" --no-deps app sh -c 'go get golang.org/x/tools/cmd/goimports && $$GOPATH/bin/goimports -d -w .'
	docker-compose run $(DC_RUN_ARGS) --no-deps app gofmt -s -w -d .
	docker-compose run $(DC_RUN_ARGS) --no-deps app go mod tidy

lint: ## Run app linters
	docker-compose run --rm --no-deps golint golangci-lint run

gotest: ## Run app tests
	docker-compose run $(DC_RUN_ARGS) --no-deps app go test -v -race -timeout 5s ./...

test: lint gotest ## Run app tests and linters

cover: ## Run app tests with coverage report
	docker-compose run $(DC_RUN_ARGS) app sh -c 'go test -race -covermode=atomic -coverprofile /tmp/cp.out ./... && go tool cover -html=/tmp/cp.out -o ./coverage.html'
	-sensible-browser ./coverage.html && sleep 2 && rm -f ./coverage.html

shell: ## Start shell into container with golang
	docker-compose run $(DC_RUN_ARGS) app bash

playground: ## Make application playground (init "stub" images for a work)
	test -d ./temp || mkdir ./temp
	test ! -d ./temp/playground || rm -R ./temp/playground
	test -f ./temp/playground.tar.gz || curl -SsL -o ./temp/playground.tar.gz 'https://github.com/tarampampam/tinifier/archive/playground.tar.gz'
	tar -xf ./temp/playground.tar.gz -C ./temp
	mv ./temp/tinifier-playground ./temp/playground
	@printf "\n   \e[30;42m %s \033[0m\n\n" 'Sample images located in `./temp/playground` directory';

clean: ## Make clean
	docker-compose down -v -t 1
	-docker rmi $(APP_NAME):local -f
