公共API
欢迎使用我们的公共 API！该 API 是完全免费的，您可以随意调用和使用它来进行学习、练习或开发项目。

获取最新同步 (GET)
https://docker.aityp.com/api/v1/latest
获取网站信息 (GET)
https://docker.aityp.com/api/v1/website
获取等待同步 (GET)
https://docker.aityp.com/api/v1/wait
获取今日同步 (GET)
https://docker.aityp.com/api/v1/today
获取错误同步 (GET)
https://docker.aityp.com/api/v1/error
监控以及健康检查 (GET)
https://docker.aityp.com/api/v1/health
根据邮箱地址获取已经同步的镜像 (GET)
https://docker.aityp.com/api/v1/email/您的邮箱地址
获取你的外网真实IP (GET)
https://docker.aityp.com/api/v1/ip
查询镜像API (GET)
https://docker.aityp.com/api/v1/image?search=
示例:
https://docker.aityp.com/api/v1/image?search=gcr.io/google-containers/coredns:1.2

https://docker.aityp.com/api/v1/image?search=python&site=docker.io&platform=linux/arm64

https://docker.aityp.com/api/v1/image?search=python&site=All&platform=linux/arm64

最大返回50条数据

site参数: All gcr.io ghcr.io quay.io k8s.gcr.io docker.io registry.k8s.io docker.elastic.co skywalking.docker.scarf.sh mcr.microsoft.com platform参数: All linux/386 linux/amd64 linux/arm64 linux/arm linux/ppc64le linux/s390x linux/mips64le linux/riscv64 linux/loong64