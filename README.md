# 云原生训练营第0期作业

## [模块二作业 - httpserver](httpserver)

作业目录：httpserver/

编写一个 HTTP 服务器，大家视个人不同情况决定完成到哪个环节，但尽量把 1 都做完。

* 1.接收客户端 request，并将 request 中带的 header 写入 response header
* 2.读取当前系统的环境变量中的 VERSION 配置，并写入 response header
* 3.Server 端记录访问日志包括客户端 IP，HTTP 返回码，输出到 server 端的标准输出
* 4. 当访问 localhost/healthz 时，应返回 200

## 模块三作业

作业目录：httpserver/Dockerfile

* 构建本地镜像。

利用多段构建，在第一个容器中编译代码，将编译生成的二进制文件放入第二个容器内执行

* 编写 Dockerfile 将练习 2.2 编写的 httpserver 容器化（请思考有哪些最佳实践可以引入到 Dockerfile 中来）。
```dockerfile
FROM golang:1.17-alpine AS build
LABEL author="zewade" course="cloudnative"
COPY main.go go.mod /go/src/httpserver/
WORKDIR /go/src/httpserver/
RUN go build -o /bin/httpserver
FROM alpine
COPY --from=build /bin/httpserver /bin/httpserver
EXPOSE 8080
ENTRYPOINT ["/bin/httpserver"]
```
* 将镜像推送至 Docker 官方镜像仓库。
```shell
##在官网注册Dokcer用户名并登录
docker login
##镜像的Tag要以自己的Docker用户名开头，才允许推送
docker push wadedc/httpserver:v1.0.1
```
* 通过 Docker 命令本地启动 httpserver。
```shell
docker run -d -P wadedc/httpserver:v1.0.1
```
* 通过 nsenter 进入容器查看 IP 配置。
```shell
nsenter -n -t 12850 ip a
```
* 作业需编写并提交 Dockerfile 及源代码。