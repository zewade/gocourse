# 基于Istio的高级流量管理入门实践
## 目标
* 在Kuberntes集群中安装Istio
* 通过将演示的httpserver服务在K8S上以Istio Ingress Gateway的形式发布出来，实践Istio的常见概念和应用模式，要求达到以下几点：
   * 配置https访问，实现安全保证
   * 配置七层路由规则，演练高级流量管理
   * 配置接入Open Tracing，整合Jaeger显示服务调用链路

## 安装Isito
**远程到K8S的master节点进行操作，首先下载Istio最新稳定版如下：**

```bash
curl -L https://istio.io/downloadIstio | sh -
```
成功下载会有如下提示：

```bash
[root@k8s-master1 ~]# curl -L https://istio.io/downloadIstio | sh -
  % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
                                 Dload  Upload   Total   Spent    Left  Speed
100   102  100   102    0     0    193      0 --:--:-- --:--:-- --:--:--   193
100  4549  100  4549    0     0   4092      0  0:00:01  0:00:01 --:--:--  4092

Downloading istio-1.12.1 from https://github.com/istio/istio/releases/download/1.12.1/istio-1.12.1-linux-amd64.tar.gz ...

Istio 1.12.1 Download Complete!

Istio has been successfully downloaded into the istio-1.12.1 folder on your system.

Next Steps:
See https://istio.io/latest/docs/setup/install/ to add Istio to your Kubernetes cluster.

To configure the istioctl client tool for your workstation,
add the /root/istio-1.12.1/bin directory to your environment path variable with:
	 export PATH="$PATH:/root/istio-1.12.1/bin"

Begin the Istio pre-installation check by running:
	 istioctl x precheck

Need more information? Visit https://istio.io/latest/docs/setup/install/
```
**根据提示将istioctl安装目录添加到PATH环境变量**

```bash
export PATH="$PATH:/root/istio-1.12.1/bin"
```
**执行Istio安装命令**

通过profile可以指定一组预置的安装配置文件，例如demo类型的配置文件用于配合官方的Booinfo演示应用程序，也可以用来测试Istio的各项功能。这个配置文件打开了高级别的链路追踪和访问日志（影响性能），因此不适合进行性能测试。全部配置文件的介绍参见官网：[https://istio.io/latest/docs/setup/additional-setup/config-profiles/](https://istio.io/latest/docs/setup/additional-setup/config-profiles/)

```bash
istioctl install --set profile=demo
```
安装过程如下：

```bash
[root@k8s-master1 ~]# istioctl install --set profile=demo
This will install the Istio 1.12.1 demo profile with ["Istio core" "Istiod" "Ingress gateways" "Egress gateways"] components into the cluster. Proceed? (y/N) y
✔ Istio core installed
✔ Istiod installed
✔ Egress gateways installed
✔ Ingress gateways installed
✔ Installation complete                                                                                       Making this installation the default for injection and validation.

Thank you for installing Istio 1.12.  Please take a few minutes to tell us about your install/upgrade experience!
```
## httpserver演示服务介绍
httpserver是一个用Go语言编写的示例httpserver服务端，它有如下路径：

* `/hello/golang` ：打印提示语，Hello, Golang!
* `/healthz` ：健康检查接口
* `/hello/service1` ：tracing测试，起3个httpserver实例进行串联调用，调用逻辑为client --> server1:8080`/hello/service1`  --> server2:8081`/hello/service2`  --> server3:8082`/hello/service3` 

**httpserver的代码如下：**

```go
package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"httpserver/metrics"
)

const (
	XForwardedFor = "X-Forwarded-For"
	XRealIP       = "X-Real-IP"
)

//glog输出到控制台的启动参数 -v=4 -logtostderr
func main() {
	//指定端口
	addr := flag.String("addr", ":8080", "Server Address.")
	flag.Parse()

	glog.V(2).Info("Starting http server...")
	//注册metrics
	metrics.Register()

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleRequest)
	//发布metrics
	mux.Handle("/metrics", promhttp.Handler())

	srv := http.Server{
		Addr:    *addr,
		Handler: mux,
	}
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()
	log.Printf("Server Started")
	<-done
	log.Printf("Server stopped")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		//extra handling
		cancel()
	}()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown Failed:%+v", err)
	}
	log.Print("Server Exited Properly")
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	//Request Header的结构为map[string][]string
	for name, values := range r.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}
	//获取VERSION环境变量
	w.Header().Add("VERSION", os.Getenv("VERSION"))
	//处理地址路由
	var statusCode int
	if r.URL.Path == "/hello/golang" {
		statusCode = helloGolang(w, r)
	} else if r.URL.Path == "/hello/service1" {
		statusCode = helloService(w, r, "service1")
	} else if r.URL.Path == "/hello/service2" {
		statusCode = helloService(w, r, "service2")
	} else if r.URL.Path == "/hello/service3" {
		statusCode = helloService(w, r, "service3")
	} else if r.URL.Path == "/healthz" {
		statusCode = healthz(w, r)
	} else {
		statusCode = http.StatusNotFound
	}
	w.WriteHeader(statusCode)
	//记录访问日志包括客户端 IP，HTTP 返回码
	fmt.Printf("Time:%s  IP:%s  Status:%d\n", time.Now().Format("2006-01-02 15:04:05.000"), getClientIP(r), statusCode)
}

func healthz(w http.ResponseWriter, r *http.Request) int {
	io.WriteString(w, "Status OK!")
	return http.StatusOK
}

func helloGolang(w http.ResponseWriter, r *http.Request) int {
	glog.V(4).Info("entering hello Golang handler")
	//设置延时
	timer := metrics.NewTimer()
	defer timer.ObserveTotal()
	delay := randInt(10, 2000)
	time.Sleep(time.Millisecond * time.Duration(delay))

	io.WriteString(w, "Hello, Golang!")
	glog.V(4).Infof("Respond in %d ms", delay)
	return http.StatusOK
}

func randInt(min int, max int) int {
	rand.Seed(time.Now().UTC().UnixNano())
	return min + rand.Intn(max-min)
}

func getClientIP(r *http.Request) string {
	remoteAddr := r.RemoteAddr
	if ip := r.Header.Get(XRealIP); ip != "" {
		remoteAddr = ip
	} else if ip = r.Header.Get(XForwardedFor); ip != "" {
		remoteAddr = ip
	} else {
		remoteAddr, _, _ = net.SplitHostPort(remoteAddr)
	}
	if remoteAddr == "::1" {
		remoteAddr = "127.0.0.1"
	}
	return remoteAddr
}

func helloService(w http.ResponseWriter, r *http.Request, svc string) int {
	glog.V(4).Info("entering hello service1 handler")
	//设置延时
	timer := metrics.NewTimer()
	defer timer.ObserveTotal()
	delay := randInt(10, 2000)
	time.Sleep(time.Millisecond * time.Duration(delay))

	var req *http.Request
	var err error
	if svc == "service1" {
		io.WriteString(w, "Hello, Service1!\n")
		//req, err = http.NewRequest("GET", "http://localhost:8081/hello/service2", nil)
		req, err = http.NewRequest("GET", "http://httpserver-svc2/hello/service2", nil)
		if err != nil {
			fmt.Printf("%s", err)
		}
	} else if svc == "service2" {
		io.WriteString(w, "Hello, Service2!\n")
		//req, err = http.NewRequest("GET", "http://localhost:8082/hello/service3", nil)
		req, err = http.NewRequest("GET", "http://httpserver-svc3/hello/service3", nil)
		if err != nil {
			fmt.Printf("%s", err)
		}
	} else {
		io.WriteString(w, "Hello, Service3!\n")
		for k, v := range r.Header {
			for _, h := range v {
				io.WriteString(w, fmt.Sprintf("%s: %s\n", k, h))
			}
		}
		glog.V(4).Infof("Respond in %d ms", delay)
		return http.StatusOK
	}
	lowerCaseHeader := make(http.Header)
	for key, value := range r.Header {
		lowerCaseHeader[strings.ToLower(key)] = value
	}
	glog.Info("headers:", lowerCaseHeader)
	req.Header = lowerCaseHeader
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		glog.Info("HTTP get failed with error: ", "error", err)
	} else {
		glog.Info("HTTP get succeeded")
	}
	resp.Write(w)

	glog.V(4).Infof("Respond in %d ms", delay)
	return http.StatusOK
}
```






















