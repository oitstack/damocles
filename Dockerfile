FROM golang:1.13 as build

ENV GO111MODULE=on
ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /go/release

ADD . .

RUN GOOS=linux CGO_ENABLED=0 GOARCH=amd64 go build -ldflags="-s -w" -installsuffix cgo -o app src/main.go

FROM scratch as prod

LABEL theSword=true

COPY --from=build /usr/share/zoneinfo/Asia/Shanghai /etc/localtime

COPY --from=build /go/release/app /

# 启动服务
CMD ["/app"]

# docker run -v /var/run/docker.sock:/var/run/docker.sock -p 8080:8080 --env targetName="group=bbb"  --privileged=true 32c21cfcb33a