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
	"syscall"
	"time"

	"httpserver/metrics"
)

const (
	XForwardedFor = "X-Forwarded-For"
	XRealIP       = "X-Real-IP"
)

func main() {
	flag.Set("v", "4")
	glog.V(2).Info("Starting http server...")
	//注册metrics
	metrics.Register()

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleRequest)
	//发布metrics
	mux.Handle("/metrics", promhttp.Handler())

	srv := http.Server{
		Addr:    ":8080",
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
