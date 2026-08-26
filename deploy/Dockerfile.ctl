FROM golang:1.25-alpine AS build
WORKDIR /src
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
ENV GOPROXY=https://goproxy.cn,direct \
    CGO_ENABLED=0 \
    GOTOOLCHAIN=local
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -mod=readonly -trimpath -o /out/menvctl ./cmd/menvctl

FROM alpine:3.21
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Shanghai
COPY --from=build /out/menvctl /usr/local/bin/menvctl
USER nobody
ENTRYPOINT ["/usr/local/bin/menvctl"]
