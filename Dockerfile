# Image page: <https://hub.docker.com/_/golang>
FROM golang:1.13-alpine as builder

ENV LDFLAGS="-s -w"

ADD ./src /src

WORKDIR /src

RUN set -x \
    && go version \
    && go build -ldflags="${LDFLAGS}" -o /tmp/tinifier . \
    && /tmp/tinifier -v

FROM alpine:latest
LABEL Description="Docker image with tinifier" Vendor="Tarampampam"

COPY --from=builder /tmp/tinifier /bin/tinifier

CMD ["/bin/tinifier"]
