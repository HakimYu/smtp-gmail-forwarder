# SMTP to Gmail Forwarder

A Go application that runs an SMTP server to receive emails and forwards them via Gmail API.

## Features

- Receives emails via SMTP protocol
- Forwards emails using Gmail API
- Google OAuth2 authentication
- Preserves original email headers and content
- Supports attachments and MIME types
- **Security**: Localhost only, no external access

## Quick Start

### Prerequisites

1. Docker and Docker Compose
2. Google Cloud Project with Gmail API enabled
3. OAuth2 credentials (credentials.json)

### 1. Configure Google Cloud

1. Visit [Google Cloud Console](https://console.cloud.google.com/)
2. Create a project and enable Gmail API
3. Create OAuth2 credentials (Desktop application type)
4. Download credentials as `credentials.json`
5. Add your Gmail address to **Test Users** in OAuth consent screen

> 💡 **Got 403 error?** Make sure your Gmail address is added to the test users list

### 2. Deploy to Server

```bash
# Upload code to server
scp -r smtp-gmail-forwarder root@your-server:~/

# SSH login
ssh root@your-server

# Navigate to directory
cd ~/smtp-gmail-forwarder

# Prepare configuration
mkdir -p config data
cp config.yaml.example config/config.yaml

# Upload credentials (run on local machine)
scp credentials.json root@your-server:~/smtp-gmail-forwarder/config/
```

### 3. OAuth2 Authentication

**Method A: Authenticate Locally (Recommended)**

```bash
# On local machine
go build -o forwarder cmd/forwarder/main.go
./forwarder -config config.yaml.example

# Complete authorization in browser, generates token.json
# Upload to server
scp token.json root@your-server:~/smtp-gmail-forwarder/data/
```

**Method B: Authenticate on Server**

```bash
# On server
docker build -t smtp-gmail-forwarder:latest .

docker run --rm -it \
  -v $(pwd)/config:/app/config:ro \
  -v $(pwd)/data:/app/data \
  smtp-gmail-forwarder:latest

# Copy link to local browser for authorization, paste code back
```

### 4. Start Service

```bash
# Create Docker network (for inter-container communication)
docker network create webnet

# Start service
docker-compose up -d

# View logs
docker-compose logs -f
```

### 5. Configure Your Application

#### Local Testing

```bash
python3 -c "
import smtplib
from email.message import EmailMessage

msg = EmailMessage()
msg['Subject'] = 'Test'
msg['From'] = 'your-gmail@gmail.com'
msg['To'] = 'recipient@example.com'
msg.set_content('Test content')

with smtplib.SMTP('127.0.0.1', 2525) as s:
    s.send_message(msg)
"
```

#### PHP Application (in Docker container)

```php
// PHPMailer configuration
$mail->isSMTP();
$mail->Host = 'smtp-gmail-forwarder';  // Container name
$mail->Port = 2525;
$mail->SMTPAuth = false;
$mail->setFrom('your-gmail@gmail.com');
```

**Laravel .env**:

```env
MAIL_HOST=smtp-gmail-forwarder
MAIL_PORT=2525
MAIL_USERNAME=null
MAIL_PASSWORD=null
MAIL_ENCRYPTION=null
MAIL_FROM_ADDRESS=your-gmail@gmail.com
```

**Connect PHP container to network**:

```bash
# View PHP container name
docker ps | grep php

# Connect to same network
docker network connect webnet your-php-container-name
```

## Configuration

### config.yaml

```yaml
smtp:
  host: 0.0.0.0      # Listen address inside container
  port: 2525         # SMTP port

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
      - "127.0.0.1:2525:2525"  # Localhost only
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

## Security

- Port binds to `127.0.0.1` only, no external access
- No SMTP authentication (secured by network isolation)
- credentials.json and token.json are in .gitignore
- Designed for single-server deployment with inter-container communication

## Troubleshooting

### 403: access_denied

OAuth2 app is in testing mode, need to add test users:
1. Visit [OAuth consent screen](https://console.cloud.google.com/apis/credentials/consent)
2. Scroll to **Test Users** section
3. Click **+ ADD USERS**
4. Add your Gmail address
5. Delete `data/token.json` and re-authenticate

### Connection refused

**Local testing**: Use `127.0.0.1:2525`

**Inside Docker container**: Use container name `smtp-gmail-forwarder:2525`

Check service status:
```bash
docker-compose logs -f
docker ps | grep smtp
```

### Inter-container communication issues

Ensure both containers are in the same network:
```bash
# Create network
docker network create webnet

# Connect PHP container
docker network connect webnet your-php-container-name

# Verify
docker network inspect webnet
```

### Token expired

```bash
rm data/token.json
docker-compose restart
docker-compose logs -f  # View new authorization link
```

## Limitations

- Gmail API free quota: 100-500 emails/day
- Designed for local or trusted network environments
- No SMTP authentication support (secured by network isolation)

## Project Structure

```
smtp-gmail-forwarder/
├── cmd/forwarder/main.go       # Application entry
├── internal/
│   ├── config/config.go        # Configuration management
│   ├── gmail/client.go         # Gmail API client
│   └── smtp/
│       ├── backend.go          # SMTP server
│       └── handler.go          # Email forwarding logic
├── config/
│   ├── config.yaml             # Configuration file
│   └── credentials.json        # OAuth2 credentials (not committed)
├── data/
│   └── token.json              # OAuth2 token (not committed)
├── Dockerfile
├── docker-compose.yml
└── config.yaml.example
```

## Local Development

```bash
# Build
go build -o forwarder cmd/forwarder/main.go

# Run
./forwarder -config config.yaml

# Test
go test ./...
```

## Management Commands

```bash
# View logs
docker-compose logs -f

# Restart service
docker-compose restart

# Stop service
docker-compose down

# View container status
docker ps | grep smtp

# Enter container
docker exec -it smtp-gmail-forwarder sh
```

## License

MIT
