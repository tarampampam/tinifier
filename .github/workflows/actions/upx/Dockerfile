FROM alpine:latest

RUN apk update && apk add --no-cache upx

COPY docker-entrypoint.sh /docker-entrypoint.sh

ENTRYPOINT ["/docker-entrypoint.sh"]
