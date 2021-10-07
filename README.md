# 云原生训练营第0期作业

## [模块二作业 - httpserver](httpserver)

作业目录：/httpserver

编写一个 HTTP 服务器，大家视个人不同情况决定完成到哪个环节，但尽量把 1 都做完。

* 1.接收客户端 request，并将 request 中带的 header 写入 response header
* 2.读取当前系统的环境变量中的 VERSION 配置，并写入 response header
* 3.Server 端记录访问日志包括客户端 IP，HTTP 返回码，输出到 server 端的标准输出
* 4. 当访问 localhost/healthz 时，应返回 200