# 使用轻量级的 Alpine Linux 作为基础镜像
FROM alpine:latest

# 安装时区数据和 CA 证书
RUN apk add --no-cache tzdata ca-certificates
ENV TZ=Asia/Shanghai

# 设置工作目录
WORKDIR /app

# 将 glog 可执行文件复制到镜像中
COPY glog /app/glog

# 授予执行权限
RUN chmod +x /app/glog

# 暴露服务端口
EXPOSE 37371

# 设置容器启动时执行的命令
CMD ["/app/glog"]
