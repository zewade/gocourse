package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

const (
	XForwardedFor = "X-Forwarded-For"
	XRealIP       = "X-Real-IP"
)

func main() {
	http.HandleFunc("/", handleRequest)
	log.Panic(http.ListenAndServe(":8080", nil))
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
	io.WriteString(w, "Hello, Golang!")
	return http.StatusOK
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
