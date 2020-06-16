# Image page: <https://hub.docker.com/_/golang>
FROM golang:1.14-alpine as builder

# can be passed with any prefix (like `v1.2.3@GITHASH`)
# e.g.: `docker build --build-arg "APP_VERSION=v1.2.3@GITHASH" .`
ARG APP_VERSION="undefined@docker"

RUN set -x \
    && mkdir /src \
    # SSL ca certificates (ca-certificates is required to call HTTPS endpoints)
    && apk add --no-cache ca-certificates upx \
    && update-ca-certificates

WORKDIR /src

COPY ./go.mod ./go.sum ./

# Burn modules cache
RUN set -x \
    && go version \
    && go mod download \
    && go mod verify

COPY . /src

RUN set -x \
    && upx -V \
    && go version \
    && GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X tinifier/version.version=${APP_VERSION}" -o /tmp/tinifier . \
    && upx -7 /tmp/tinifier \
    && /tmp/tinifier version \
    && /tmp/tinifier -h

# Image page: <https://hub.docker.com/_/alpine>
FROM alpine:latest as runtime

ARG APP_VERSION="undefined@docker"

LABEL \
    org.label-schema.name="tinifier" \
    org.label-schema.description="Docker image with tinifier" \
    org.label-schema.url="https://github.com/tarampampam/tinifier" \
    org.label-schema.vcs-url="https://github.com/tarampampam/tinifier" \
    org.label-schema.vendor="tarampampam" \
    org.label-schema.license="MIT" \
    org.label-schema.version="$APP_VERSION"
    org.label-schema.schema-version="1.0"

RUN set -x \
    && mkdir /app \
    # Unprivileged user creation <https://stackoverflow.com/a/55757473/12429735RUN>
    && adduser \
        --disabled-password \
        --gecos "" \
        --home "/nonexistent" \
        --shell "/sbin/nologin" \
        --no-create-home \
        --uid "10001" \
        "appuser"

# Use an unprivileged user
USER appuser:appuser

# Import from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /tmp/tinifier /bin/tinifier

ENTRYPOINT ["/bin/tinifier"]
