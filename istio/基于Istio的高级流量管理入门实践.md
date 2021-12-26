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
























