# Image page: <https://hub.docker.com/_/golang>
FROM golang:1.13-alpine as builder

# UPX parameters help: <https://www.mankier.com/1/upx>
ARG upx_params
ENV upx_params=${upx_params:--5}

RUN apk add --no-cache upx

ADD ./src /src

WORKDIR /src

RUN set -x \
    && upx -V \
    && go version \
    && go build -ldflags='-s -w' -o /tmp/tinifier . \
    && upx ${upx_params} /tmp/tinifier \
    && /tmp/tinifier -V \
    && /tmp/tinifier -h

FROM alpine:latest
LABEL Description="Docker image with tinifier" Vendor="Tarampampam"

COPY --from=builder /tmp/tinifier /bin/tinifier

ENTRYPOINT ["/bin/tinifier"]
