#
# Builder
#
FROM golang:1.13-alpine as builder

RUN apk add --no-cache ca-certificates

RUN go get -v github.com/chinaboard/coral

#
# Final stage
#
FROM alpine:3.10

LABEL maintainer "chinaboard <chinaboard@gmail.com>"

RUN apk add --no-cache \
    ca-certificates \
    openssh-client \
    tini

EXPOSE 5438

# install
COPY --from=builder /go/bin/coral /bin/coral
#COPY doc/sampleConfig.ini /root/.coral/cc.ini

ENTRYPOINT ["tini", "--"]
CMD ["coral"]