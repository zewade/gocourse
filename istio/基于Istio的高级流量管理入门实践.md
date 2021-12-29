# 基于Istio的高级流量管理入门实践
## 目标
* 在Kuberntes集群中安装Istio
* 用Go语言编写一个httpserver处理http请求
* 将httpserver服务在K8S上以Istio Ingress Gateway的形式发布出来，实践Istio的常见概念和应用模式，要求达到以下几点：
   * 配置网关https访问，实现安全保证
   * 配置七层路由规则，演练高级流量管理
   * 配置Tracing接入，使用Jaeger收集服务调用链 
   * 配置路由规则，演练泳道式灰度发布

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
## 安装Jaeger
```go
//部署Jaeger
kubectl apply -f jaeger.yaml
//编辑istio的configmap，为方便演示，设置tracing.sampling采样率为100，配置实时生效
kubectl edit configmap istio -n istio-system
//...
tracing:
  sampling: 100
//...
```
jaeger.yaml

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: jaeger
  namespace: istio-system
  labels:
    app: jaeger
spec:
  selector:
    matchLabels:
      app: jaeger
  template:
    metadata:
      labels:
        app: jaeger
      annotations:
        sidecar.istio.io/inject: "false"
        prometheus.io/scrape: "true"
        prometheus.io/port: "14269"
    spec:
      containers:
        - name: jaeger
          image: "docker.io/jaegertracing/all-in-one:1.23"
          env:
            - name: BADGER_EPHEMERAL
              value: "false"
            - name: SPAN_STORAGE_TYPE
              value: "badger"
            - name: BADGER_DIRECTORY_VALUE
              value: "/badger/data"
            - name: BADGER_DIRECTORY_KEY
              value: "/badger/key"
            - name: COLLECTOR_ZIPKIN_HOST_PORT
              value: ":9411"
            - name: MEMORY_MAX_TRACES
              value: "50000"
            - name: QUERY_BASE_PATH
              value: /jaeger
          livenessProbe:
            httpGet:
              path: /
              port: 14269
          readinessProbe:
            httpGet:
              path: /
              port: 14269
          volumeMounts:
            - name: data
              mountPath: /badger
          resources:
            requests:
              cpu: 10m
      volumes:
        - name: data
          emptyDir: {}
---
apiVersion: v1
kind: Service
metadata:
  name: tracing
  namespace: istio-system
  labels:
    app: jaeger
spec:
  type: ClusterIP
  ports:
    - name: http-query
      port: 80
      protocol: TCP
      targetPort: 16686
    # Note: Change port name if you add '--query.grpc.tls.enabled=true'
    - name: grpc-query
      port: 16685
      protocol: TCP
      targetPort: 16685
  selector:
    app: jaeger
---
# Jaeger implements the Zipkin API. To support swapping out the tracing backend, we use a Service named Zipkin.
apiVersion: v1
kind: Service
metadata:
  labels:
    name: zipkin
  name: zipkin
  namespace: istio-system
spec:
  ports:
    - port: 9411
      targetPort: 9411
      name: http-query
  selector:
    app: jaeger
---
apiVersion: v1
kind: Service
metadata:
  name: jaeger-collector
  namespace: istio-system
  labels:
    app: jaeger
spec:
  type: ClusterIP
  ports:
    - name: jaeger-collector-http
      port: 14268
      targetPort: 14268
      protocol: TCP
    - name: jaeger-collector-grpc
      port: 14250
      targetPort: 14250
      protocol: TCP
    - port: 9411
      targetPort: 9411
      name: http-zipkin
  selector:
    app: jaeger
```
## 准备httpserver演示服务
httpserver是一个用Go语言编写的示例httpserver服务端，它有如下路径：

* `/hello/golang` ：打印提示语，Hello, Golang!
* `/healthz` ：健康检查接口
* `/hello/service1` ：tracing测试，起3个httpserver实例进行串联调用，调用逻辑为client --> server1：`/hello/service1`  --> server2：`/hello/service2`  --> server3：`/hello/service3` 

接下去我们把httpserver服务，构建成Docker镜像，部署成3个独立的微服务到K8S，组成V1版本的httpserver应用。服务的主要代码如下：

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
将V1版本的镜像推送到Docker官方镜像仓库，指定tag为v1.3.2：

```docker
FROM alpine:3.15
ENV VERSION=1.17.1
LABEL multi.lang="go" multi.type="webserver" other="homework"
ADD bin/amd64/httpserver /httpserver
EXPOSE 8080
CMD ["/httpserver", "-v=4","-logtostderr"]
```
## 部署httpserver
### 创建命名空间
为httpserver创建一个命名空间cloudnative，并开启sidecar注入

```bash
kubectl create ns cloudnative
kubectl label ns cloudnative istio-injection=enabled
```
### 部署应用
部署httpserver1/httpserver2/httpserver3

```bash
kubectl -n cloudnative apply -f httpserver1.yaml
kubectl -n cloudnative apply -f httpserver2.yaml
kubectl -n cloudnative apply -f httpserver3.yaml 
```
httpserver1.yaml

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: httpserver-service1
spec:
  replicas: 1
  selector:
    matchLabels:
      app: httpserver-service1
  template:
    metadata:
      labels:
        app: httpserver-service1
    spec:
      containers:
        - name: httpserver-service1
          imagePullPolicy: Always
          image: wadedc/httpserver:v1.3.2
          ports:
            - containerPort: 8080
---
apiVersion: v1
kind: Service
metadata:
  name: httpserver-service1
spec:
  ports:
    - name: http
      port: 80
      protocol: TCP
      targetPort: 8080
  selector:
    app: httpserver-service1
```
httpserver2.yaml

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: httpserver-service2
spec:
  replicas: 1
  selector:
    matchLabels:
      app: httpserver-service2
  template:
    metadata:
      labels:
        app: httpserver-service2
    spec:
      containers:
        - name: httpserver-service2
          imagePullPolicy: Always
          image: wadedc/httpserver:v1.3.2
          ports:
            - containerPort: 8080
---
apiVersion: v1
kind: Service
metadata:
  name: httpserver-service2
spec:
  ports:
    - name: http
      port: 80
      protocol: TCP
      targetPort: 8080
  selector:
    app: httpserver-service2
```
httpserver3.yaml

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: httpserver-service3
spec:
  replicas: 1
  selector:
    matchLabels:
      app: httpserver-service3
  template:
    metadata:
      labels:
        app: httpserver-service3
    spec:
      containers:
        - name: httpserver-service3
          imagePullPolicy: Always
          image: wadedc/httpserver:v1.3.2
          ports:
            - containerPort: 8080
---
apiVersion: v1
kind: Service
metadata:
  name: httpserver-service3
spec:
  ports:
    - name: http
      port: 80
      protocol: TCP
      targetPort: 8080
  selector:
    app: httpserver-service3
```
### 配置Istio网关
生成https证书并保存到secret，配置service1.cncamp.io作为域名发布httpserver1并配置https访问。

```bash
openssl req -x509 -sha256 -nodes -days 365 -newkey rsa:2048 -subj '/O=cncamp Inc./CN=*.cncamp.io' -keyout cncamp.io.key -out cncamp.io.crt
kubectl create -n istio-system secret tls cncamp-credential --key=cncamp.io.key --cert=cncamp.io.crt
kubectl apply -f istio-specs.yaml -n cloudnative
```
cncamp.io.key

```Plain Text
-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQDNKftOzu8h1UbR
w2Rj35BI/DClzdDcOHyhTGBZV/fKf6cISyImR/COehjyCl36vBW/VT4esxcwjCLu
UYDOVX6ROqkJPBNWkHv3ExU4BqkzTm2sG2aZ4kgnHMjtcS4F/lnDNFuuUcwlUOfO
cV3KV2z1U1U/0NL4UMyPoBlWkkOq1vNnYn48qeQ1cT6oB722/illEx/ttCT/m6E5
sv6ho36MhVsW2OQskQIjhi7bo6cSARHx1SxQI1Bt+kg5i36FrP62sAaiDg588A+5
bBNhtIl7/dg0afAqZmcxLuQHZ2RSQj+lGi4qsvRl71hBdDJwSJ7ZuPPslBJ1eInz
ua3xVG3xAgMBAAECggEBAK4zPTzXG9hgXPdyrmTWyI4QA8ZkkTjLhZb3YL+7n5wC
83JpSR+z6Z8wMMoi31UsKlMBL/OlIMWJ14b/ER0hHox1gF1k8w6HM5rASz38+eGk
cB64TU/QAG+lUz244dkY9GJ4rHfA4FO29HwnafmKbeuHzFSZHOjWwjoZOCp3mpkM
z1N0pun5nKUEUy19E4LjE3QU7AFNx8oFlkE9k0pL6t6VZolVhJBPpxGapwj5K0Uu
HZjJjSrZok8qCFJfMauOA6avIIAdxvFrZpHVJflMg0Nu+pEVsYfVzPjOzSMmdHgf
eVSvDf+nxAvb09Oa10iKH/UCGjzTVfH6UjdAyoskxDkCgYEA+44hxd/P/Wo2mGiT
ilELJEOfVHlZIBrGvrrOHPZc1LB0hLxTK9VzN7VYRu+ZSdS5V5ahPkfa4/fgNBfz
zXoioA6Li3bcRCOFlDyAqhBaEXk+YWHH0poeqmSnV6XlRqDOKCSjsHojYfasBGju
g1N2Gh7JusY5ttLjufv7Bt2BRpcCgYEA0MoBvyZxcIMIjuMoRPQ2eJ3W1Yms4CVa
HT/lPOguZSKI/1vywK1doVnrgikUQRLNSVFMGeHwRcbNbWNyTGafc/Q2hNVR8XBc
mRwNcmWqwRqYiN2eOtWOyxN9IhA1vXUAxyYIkDCOq/NmQYwtoqhB4Esmp1ZDv1vh
IkqO+VEDyLcCgYB3u+1DXAaJ3nZiENS5L14YQr+h26iaaWRUAGJ+0pzY96xeSa1k
3dJbn8uG6CCUTdZyZFYXaOg9PgzPft8i3JGCkanGFis9m5LHPg0X5XSZgJY6j+om
ygjynboxM9tvxLab0OTA6UHSLTEvYCq3A5DhWeo3JobuCG8wZUnUuLYBvQKBgGhG
a2bnMUKq/qw2QRdnDIli8zfEwcVUglQXZErt/rXd8KPwbSXTr+50tU1VbNsvI73Z
T3Ohxtlid5iJUT1dB4fm0Q+4Zmt53ZVOUFzw773vpXy9ilgB7oX33sgTZnOPqurL
UP2KcsboEgrskqIo/HWjstNiHwXEQoVYzV0xG/2zAoGAI50SRG1Gxv0HqVJJ+DHR
OI0gBiGVNqo31f+IvQSB4xhX0wzBjV56ov4eNByq/eWVbkOA5kjo3zUyZ9zu9ZFi
A8VTczIQyd1vmTzApsbxI5UVRzwc49vSzQWi7NcoO7YlSrzr6hU/zqtWTnNAyupz
lyKjB5MXvtypRS46hxPCRXU=
-----END PRIVATE KEY-----
```
cncamp.io.crt

```Plain Text
-----BEGIN CERTIFICATE-----
MIIC1DCCAbwCCQCPNLGdLUqaJzANBgkqhkiG9w0BAQsFADAsMRQwEgYDVQQKDAtj
bmNhbXAgSW5jLjEUMBIGA1UEAwwLKi5jbmNhbXAuaW8wHhcNMjExMjE4MTIzNjQz
WhcNMjIxMjE4MTIzNjQzWjAsMRQwEgYDVQQKDAtjbmNhbXAgSW5jLjEUMBIGA1UE
AwwLKi5jbmNhbXAuaW8wggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDN
KftOzu8h1UbRw2Rj35BI/DClzdDcOHyhTGBZV/fKf6cISyImR/COehjyCl36vBW/
VT4esxcwjCLuUYDOVX6ROqkJPBNWkHv3ExU4BqkzTm2sG2aZ4kgnHMjtcS4F/lnD
NFuuUcwlUOfOcV3KV2z1U1U/0NL4UMyPoBlWkkOq1vNnYn48qeQ1cT6oB722/ill
Ex/ttCT/m6E5sv6ho36MhVsW2OQskQIjhi7bo6cSARHx1SxQI1Bt+kg5i36FrP62
sAaiDg588A+5bBNhtIl7/dg0afAqZmcxLuQHZ2RSQj+lGi4qsvRl71hBdDJwSJ7Z
uPPslBJ1eInzua3xVG3xAgMBAAEwDQYJKoZIhvcNAQELBQADggEBAMVIW9n5htvX
sFpJUFrDIILvzgoVfsaE71V/Qa1SyjgjNUOgr7TH7vZLSdXMAPXhjKlhByYTXuUG
6X/54I8f8PdG+5QUkOH8nUC2KuxAsbNY0yxYHQT8G3YBOnJXGSro2/Nmyoww8Bw3
ETjzlBSxTQez1QzJwqgtv9uUfu92+53cZZ0/mPxnBZJFj8OoA+TAH1nVlELhsA6o
PmZS/3D97JzharoFa2yznfbUHWsYGmUI+xj4UcG+VuaZg8zwvLXJyFQqnlfJ/GhM
oTj4bipm7l6XeR283hLGeuyhXrNkM0lU93PuG+4Z6C9bJ3S7xdQNg4pywgdXszyg
+OHht9cP7b8=
-----END CERTIFICATE-----
```
istio-specs.yaml

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: httpserver-service1
spec:
  gateways:
    - httpserver-service1
  hosts:
    - service1.cncamp.io
  http:
    - match:
        - port: 443
      route:
        - destination:
            host: httpserver-service1.cloudnative.svc.cluster.local
            port:
              number: 80
---
apiVersion: networking.istio.io/v1beta1
kind: Gateway
metadata:
  name: httpserver-service1
spec:
  selector:
    istio: ingressgateway
  servers:
    - hosts:
        - service1.cncamp.io
      port:
        name: https-default
        number: 443
        protocol: HTTPS
      tls:
        mode: SIMPLE
        credentialName: cncamp-credential
```
## 测试httpserver及Tracing
## 测试泳道式灰度发布












