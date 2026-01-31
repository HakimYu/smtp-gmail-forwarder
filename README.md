# SMTP 到 Gmail 转发器

一个 Go 应用程序，运行 SMTP 服务器接收邮件，并通过 Gmail API 转发出去。

## 功能特性

- 通过 SMTP 协议接收邮件
- 使用 Gmail API 转发邮件
- Google OAuth2 认证
- 保留原始邮件头和内容
- 支持附件和 MIME 类型
- **安全设计**：仅本地访问，外部无法连接

## 快速开始

### 前置要求

1. Docker 和 Docker Compose
2. Google Cloud 项目并启用 Gmail API
3. OAuth2 凭据（credentials.json）

### 1. 配置 Google Cloud

1. 访问 [Google Cloud Console](https://console.cloud.google.com/)
2. 创建项目并启用 Gmail API
3. 创建 OAuth2 凭据（桌面应用类型）
4. 下载凭据保存为 `credentials.json`
5. 在 **OAuth 同意屏幕** → **测试用户** 中添加你的 Gmail 地址

> 💡 **遇到 403 错误？** 确保你的 Gmail 地址已添加到测试用户列表

### 2. 部署到服务器

```bash
# 上传代码到服务器
scp -r smtp-gmail-forwarder root@your-server:~/

# SSH 登录
ssh root@your-server

# 进入目录
cd ~/smtp-gmail-forwarder

# 准备配置
mkdir -p config data
cp config.yaml.example config/config.yaml

# 上传凭据（在本地执行）
scp credentials.json root@your-server:~/smtp-gmail-forwarder/config/
```

### 3. OAuth2 认证

**方法 A：在本地认证（推荐）**

```bash
# 在本地电脑运行
go build -o forwarder cmd/forwarder/main.go
./forwarder -config config.yaml.example

# 浏览器完成授权，生成 token.json
# 上传到服务器
scp token.json root@your-server:~/smtp-gmail-forwarder/data/
```

**方法 B：在服务器认证**

```bash
# 在服务器上
docker build -t smtp-gmail-forwarder:latest .

docker run --rm -it \
  -v $(pwd)/config:/app/config:ro \
  -v $(pwd)/data:/app/data \
  smtp-gmail-forwarder:latest

# 复制链接到本地浏览器授权，粘贴 code 回终端
```

### 4. 启动服务

```bash
# 创建 Docker 网络（用于容器间通信）
docker network create webnet

# 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f
```

### 5. 配置你的应用

#### 本地测试

```bash
python3 -c "
import smtplib
from email.message import EmailMessage

msg = EmailMessage()
msg['Subject'] = '测试'
msg['From'] = 'your-gmail@gmail.com'
msg['To'] = 'recipient@example.com'
msg.set_content('测试内容')

with smtplib.SMTP('127.0.0.1', 2525) as s:
    s.send_message(msg)
"
```

#### PHP 应用（Docker 容器内）

```php
// PHPMailer 配置
$mail->isSMTP();
$mail->Host = 'smtp-gmail-forwarder';  // 容器名
$mail->Port = 2525;
$mail->SMTPAuth = false;
$mail->setFrom('your-gmail@gmail.com');
```

**Laravel .env**：

```env
MAIL_HOST=smtp-gmail-forwarder
MAIL_PORT=2525
MAIL_USERNAME=null
MAIL_PASSWORD=null
MAIL_ENCRYPTION=null
MAIL_FROM_ADDRESS=your-gmail@gmail.com
```

**连接 PHP 容器到网络**：

```bash
# 查看 PHP 容器名
docker ps | grep php

# 连接到同一网络
docker network connect webnet your-php-container-name
```

## 配置说明

### config.yaml

```yaml
smtp:
  host: 0.0.0.0      # 容器内监听地址
  port: 2525         # SMTP 端口

gmail:
  credentials_file: /app/config/credentials.json
  token_file: /app/data/token.json
```

### docker-compose.yml

```yaml
services:
  smtp-forwarder:
    build: .
    container_name: smtp-gmail-forwarder
    restart: unless-stopped
    ports:
      - "127.0.0.1:2525:2525"  # 仅本地访问
    volumes:
      - ./config:/app/config:ro
      - ./data:/app/data
    environment:
      - TZ=Asia/Shanghai
    networks:
      - webnet

networks:
  webnet:
    external: true
```

## 安全说明

- 端口仅绑定到 `127.0.0.1`，外部网络无法访问
- 未实现 SMTP 认证（设计为本地使用）
- credentials.json 和 token.json 已在 .gitignore 中
- 适合单服务器部署，多容器间通信

## 常见问题

### 403: access_denied

OAuth2 应用在测试模式，需要添加测试用户：
1. 访问 [OAuth 同意屏幕](https://console.cloud.google.com/apis/credentials/consent)
2. 滚动到 **测试用户** 部分
3. 点击 **+ ADD USERS**
4. 添加你的 Gmail 地址
5. 删除 `data/token.json` 重新认证

### Connection refused

**本地测试**：确保使用 `127.0.0.1:2525`

**Docker 容器内**：使用容器名 `smtp-gmail-forwarder:2525`

检查服务状态：
```bash
docker-compose logs -f
docker ps | grep smtp
```

### 容器间无法通信

确保两个容器在同一网络：
```bash
# 创建网络
docker network create webnet

# 连接 PHP 容器
docker network connect webnet your-php-container-name

# 验证
docker network inspect webnet
```

### Token 过期

```bash
rm data/token.json
docker-compose restart
docker-compose logs -f  # 查看新的授权链接
```

## 使用限制

- Gmail API 免费配额：每天 100-500 封邮件
- 设计用于本地或受信任的网络环境
- 不支持 SMTP 认证（通过网络隔离保证安全）

## 项目结构

```
smtp-gmail-forwarder/
├── cmd/forwarder/main.go       # 程序入口
├── internal/
│   ├── config/config.go        # 配置管理
│   ├── gmail/client.go         # Gmail API 客户端
│   └── smtp/
│       ├── backend.go          # SMTP 服务器
│       └── handler.go          # 邮件转发逻辑
├── config/
│   ├── config.yaml             # 配置文件
│   └── credentials.json        # OAuth2 凭据（不提交）
├── data/
│   └── token.json              # OAuth2 令牌（不提交）
├── Dockerfile
├── docker-compose.yml
└── config.yaml.example
```

## 本地开发

```bash
# 编译
go build -o forwarder cmd/forwarder/main.go

# 运行
./forwarder -config config.yaml

# 测试
go test ./...
```

## 管理命令

```bash
# 查看日志
docker-compose logs -f

# 重启服务
docker-compose restart

# 停止服务
docker-compose down

# 查看容器状态
docker ps | grep smtp

# 进入容器
docker exec -it smtp-gmail-forwarder sh
```

## 许可证

MIT
