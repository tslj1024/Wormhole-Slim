# Wormhole-Slim

![Static Badge](https://img.shields.io/badge/license-GPLv3.0-blue)

[English](README.md) | **中文**

一个轻量级内网穿透代理

## 简介

Wormhole-Slim 是一个轻量级的内网穿透代理工具，旨在帮助用户轻松实现内网服务的暴露和访问。

想要使用此软件，你需要明白：

1. “Wormhole” 中文译为 “虫洞”，之所以是这个名字，是因为*内网穿透*这件事和空间中的虫洞很相似
2. 此软件是一个轻量级版本，主要功能就是提供流量转发
3. 您内网中的服务是基于TCP或UDP协议的

## 功能特性

- **轻量级设计**：核心功能精简，资源占用低
- **TCP穿透**：支持任意基于TCP协议的服务穿透
- **UDP穿透**：支持任意基于UDP协议的服务穿透
- **简单配置**：易于配置和使用
- **跨平台**：使用Go语言开发，支持Windows/Linux/macOS等主流操作系统
- **多客户端**：使用ClientID区分多个客户端
- **客户端断线重连**：当服务端宕机再次重启后，客户端可以自动重新连接

## 安装方法

### 二进制安装

从[Release页面](https://github.com/tslj1024/Wormhole-Slim/releases)下载预编译的二进制文件

### 源码编译

确保已安装Go 1.22+环境：

```bash
git clone https://github.com/tslj1024/Wormhole-Slim.git
cd Wormhole-Slim/server
go build -o server server.go
cd ../client
go build -o client client.go
```

## 快速开始

### 服务端 (公网服务器)

首先在公网服务器找一个位置，把`server`程序和与其同级文件夹中的`config`文件夹复制过去，如下所示：

```
wormhole
├── server
└── config/
    └── app.yml
```

修改配置文件，`app.yml`:

```yaml
server:
  port: 8081 # 这个端口用于监听客户端连接
  # 这个列表用于定义每一个公网服务器端口到内网服务的映射
  clients:
    - clientId: CLIENTID1 # 区别每个客户端。先复制
      port: 8083  		  # 用户访问内网服务时通过的端口
      tHost: 127.0.0.1 	  # 内网服务的IP地址
      tPort: 80			  # 内网服务的端口
      protocol: tcp/udp   # 协议，“/”分隔
```

运行：

```bash
./server
```

### 客户端 (内网机器)

首先在内网机器服务器找一个位置，把`client`程序和与其同级文件夹中的`config`文件夹复制过去，如下所示：

```
wormhole
├── client
└── config/
    └── app.yml
```

修改配置文件，`app.yml`:

```yaml
client:
  host: localhost     # 公网服务器地址
  port: 8081		  # 公网服务器端口，公网服务器用于监听客户端连接的端口
  clientId: CLIENTID1 # 把从服务器上复制的CLIENTID复制到这里
```

运行：

```bash
./client
```

## 安全建议

1. 不要透露你的任何一个ClientID
2. 使用端到端加密，本软件不提供加密功能，使用端到端加密后也无需本软件加密
3. 只暴露必要的服务

## 常见问题

**Q: 连接失败怎么办？**

A: 检查以下几点：

- [ ] 检查防火墙，尤其是公网服务器
- [ ] 确保客户端以及服务端配置正确，尤其是ClientID
- [ ] 确保内网服务可用
- [ ] 确保客户端可以访问到内网服务
- [ ] 确保客户端可以连接到服务端，连接不到时客户端会输出错误信息
- [ ] 如果是UDP协议连接不上，可以在两边搭建`test`文件夹里面的服务测试连通性

## 贡献指南

欢迎提交Issue和Pull Request。提交代码前请确保：

1. 通过基础测试
2. 遵循现有代码风格
3. 更新相关文档

## 许可证

本项目采用GPL-3.0开源许可证，详情见[LICENSE](LICENSE)文件。

