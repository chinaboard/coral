#
# Builder
#
FROM golang:1.18.2-alpine as builder
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories
RUN apk add --no-cache ca-certificates
ENV GOPROXY=https://goproxy.io,direct
RUN go install -v github.com/chinaboard/coral/cli@latest
RUN ls -l /go/bin
#
# Final stage
#
FROM alpine:latest

LABEL maintainer "chinaboard <chinaboard@gmail.com>"
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories
RUN apk add --no-cache \
    ca-certificates \
    openssh-client \
    tini

EXPOSE 5438
# install
COPY --from=builder /go/bin/cli /bin/coral
#COPY doc/sampleConfig.ini /root/.coral/cc.ini
ENTRYPOINT ["tini", "--"]
CMD ["coral"]