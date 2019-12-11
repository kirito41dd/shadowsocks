FROM golang:1.13-alpine
WORKDIR /
RUN apk --no-cache add git \
	&& git clone https://github.com/zshorz/shadowsocks.git
WORKDIR /shadowsocks/
RUN GOPROXY=https://goproxy.io GO111MODULE="on" go build -o server ./ss-server/
FROM alpine:latest
WORKDIR /app/
COPY --from=0 /shadowsocks/server .
ENTRYPOINT ["./server"]
