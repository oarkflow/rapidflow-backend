# Docker App CI/CD

A simple CI/CD platform built in Go using Docker for isolated build execution.

## Features

- Define pipelines in YAML with multi-step builds
- Execute steps in Docker containers for isolation
- Support for bash scripts, file creation, environment variables
- Job queue with background processing
- HTTP API for pipeline and job management
- Git repository cloning and branch checkout
- Port exposure for services

## Quick Start

1. Build the application:
   ```bash
   go build -o docker-app .
   ```

2. Start the server:
   ```bash
   ./docker-app server
   ```

3. Run a pipeline from YAML:
   ```bash
   ./docker-app run-pipeline --file=testdata/config/pipeline.yaml
   ```

## Pipeline Configuration

Pipelines are defined in YAML format. Example:

```yaml
name: "Golang Server App Pipeline"
language: "golang"
branch: "main"
folder: "./my-go-app"
expose_ports: true
env:
  GOOS: "linux"
  GOARCH: "amd64"
  PORT: "3000"
  DATABASE_URL: "sqlite://db.sqlite"
steps:
  - type: "bash"
    content: |
      apt-get update && apt-get install -y wget curl git build-essential
      echo "Tools installed"
    files:
      config.json: |
        {
          "port": 3000,
          "database": "sqlite://db.sqlite"
        }
  - type: "bash"
    content: |
      cd /workspace
      go mod download
      go build -o server .
      echo "Build completed"
  - type: "bash"
    content: |
      cd /workspace
      ./server > /tmp/server.log 2>&1 &
      SERVER_PID=$!
      sleep 3
      curl http://localhost:3000/health && echo "Health check passed" || echo "Health check failed"
      kill $SERVER_PID
      echo "Server logs:"
      cat /tmp/server.log
```

## API Endpoints

- `POST /pipelines` - Create a new pipeline
- `GET /pipelines` - List all pipelines
- `POST /pipelines/:id/jobs` - Trigger a job for a pipeline
- `GET /jobs` - List all jobs
- `GET /jobs/:id` - Get job details
- `GET /jobs/:id/steps` - Get steps for a job
- `GET /steps/:id` - Get step details
- `GET /health` - Health check

## Architecture

- **Main**: CLI interface and HTTP server
- **API**: HTTP handlers for CRUD operations
- **Models**: Data structures and database schema
- **Worker**: Job execution in Docker containers

Jobs are processed asynchronously using a background queue. Each job runs in its own Docker container with the appropriate base image based on the language specified.

## Requirements

- Go 1.21+
- Docker
- SQLite (for database)

## Missing Features for Full Functionality

To make this platform fully functional and robust for production use, consider adding:

1. **Authentication & Authorization**
   - API key authentication
   - User management
   - Role-based access control

2. **Webhooks**
   - GitHub/GitLab webhook integration
   - Automatic pipeline triggering on push/PR

3. **Advanced Job Queue**
   - Priority queues
   - Worker pools with concurrency limits
   - Job scheduling (cron-like)
   - External queue system (Redis/RabbitMQ)

4. **Monitoring & Logging**
   - Structured logging
   - Metrics collection (Prometheus)
   - Real-time log streaming (WebSockets/SSE)
   - Job execution tracing

5. **Security Enhancements**
   - Container security scanning
   - Resource limits (CPU, memory)
   - Network isolation
   - Secret management

6. **Scalability**
   - Horizontal scaling of workers
   - Database migration system
   - Caching layer

7. **User Interface**
   - Web dashboard for pipeline/job management
   - Real-time job status updates
   - Log viewers

8. **Additional Pipeline Features**
   - Parallel step execution
   - Conditional steps
   - Artifact storage and retrieval
   - Test result parsing
   - Notifications (email, Slack)

## VPS Deployment with Native Nginx

Deploy applications to remote VPS servers with native Nginx configuration (not Nginx Proxy Manager).

### Features

- **SSH-based Docker Deployment**: Deploy Docker containers to VPS servers via SSH
- **Native Nginx Configuration**: Generate and deploy Nginx configuration files directly
- **SSL/TLS Support**: Configure HTTPS with SSL certificates
- **Automatic Service Restart**: Restart Nginx after configuration changes

### Configuration

Add a `nginx` output to your runnable configuration:

```yaml
runnables:
  - name: "production-nginx-server"
    type: "docker_container"
    container_name: "my-app-prod"
    image_name: "my-web-app:latest"
    ports: ["3000"]
    outputs:
      - type: "nginx"
        config:
          # VPS connection
          host: "your-vps.example.com"
          ssh_user: "root"
          ssh_key_path: "~/.ssh/id_rsa"

          # Docker configuration
          docker_host: "unix:///var/run/docker.sock"

          # Service configuration
          domain: "myapp.example.com"
          service_port: "3000"
          container_name: "my-app-prod"
          image_name: "my-web-app:latest"

          # Nginx configuration
          nginx_config_path: "/etc/nginx/sites-enabled"
          nginx_restart_cmd: "systemctl restart nginx"
          ssl: true
          ssl_cert_path: "/etc/letsencrypt/live/myapp.example.com/fullchain.pem"
          ssl_key_path: "/etc/letsencrypt/live/myapp.example.com/privkey.pem"
```

### Prerequisites

1. **VPS Server**:
   - Docker installed and running
   - SSH access with key-based authentication
   - Nginx installed and running (not in Docker)
   - Sudo access for the SSH user

2. **Nginx Setup**:
   - Nginx configuration directories exist
   - SSL certificates (if using HTTPS)
   - Proper permissions for config files

### Generated Nginx Configuration

The provider automatically generates Nginx configuration:

**HTTP Only:**
```nginx
server {
    listen 80;
    server_name myapp.example.com;

    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

**With SSL:**
```nginx
server {
    listen 80;
    server_name myapp.example.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name myapp.example.com;

    ssl_certificate /etc/letsencrypt/live/myapp.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/myapp.example.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES128-GCM-SHA256:ECDHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers off;

    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### Example Pipeline

See `testdata/config/vps-nginx-deployment.yaml` and `testdata/config/vps-nginx-simple-demo.yaml` for complete examples.

### Security Notes

- Use SSH key authentication only
- Ensure SSH user has sudo privileges for Nginx operations
- Store SSL certificates securely
- Test Nginx configuration before deployment
- Use proper firewall rules on VPS

## Email Deployment

Send build artifacts via email with support for multiple email services.

### Features

- **Multiple Transport Methods**: SMTP, AWS SES, and HTTP API support
- **SMTP Integration**: Send emails through any SMTP server
- **AWS SES Integration**: Direct integration with Amazon SES
- **HTTP API Support**: Compatible with services like SendGrid, Mailgun, etc.
- **Artifact Delivery**: Include build artifacts in email notifications
- **Multiple Recipients**: Send to multiple email addresses
- **Customizable Content**: Configure subject, body, and sender information

### Configuration

Add an `email` deployment to your pipeline with the desired transport method:

#### SMTP Transport
```yaml
deployments:
  - name: email-notification
    type: email
    config:
      transport: "smtp"
      smtp_host: "smtp.gmail.com"
      smtp_port: 587
      username: "your-email@gmail.com"
      password: "your-app-password"
      from: "your-email@gmail.com"
      to:
        - "recipient1@example.com"
        - "recipient2@example.com"
      subject: "Build Artifact Delivery"
      body: "Please find the latest build artifact attached."
```

#### AWS SES Transport
```yaml
deployments:
  - name: email-ses
    type: email
    config:
      transport: "ses"
      region: "us-east-1"
      access_key_id: "your-aws-access-key"
      secret_access_key: "your-aws-secret-key"
      from: "noreply@yourdomain.com"
      to:
        - "team@yourdomain.com"
      subject: "Production Build Completed"
      body: "The latest production build has been completed successfully."
```

#### HTTP API Transport
```yaml
deployments:
  - name: email-api
    type: email
    config:
      transport: "http"
      api_url: "https://api.sendgrid.com/v3/mail/send"
      api_key: "your-sendgrid-api-key"
      headers:
        "Content-Type": "application/json"
      from: "builds@yourdomain.com"
      to:
        - "developers@yourdomain.com"
      subject: "Build Notification"
      body: "A new build is ready for deployment."
```

### Prerequisites

#### SMTP Transport
- Valid SMTP server credentials
- App-specific passwords for Gmail (recommended over main password)

#### AWS SES Transport
- AWS account with SES enabled
- IAM user with SES permissions (`ses:SendEmail`)
- Verified sender email address in SES

#### HTTP API Transport
- API endpoint URL for your email service
- Valid API key or authentication credentials
- Compatible JSON payload format

### Example Pipeline

See `testdata/config/email-deployment.yaml` for a complete SMTP example.

### Security Notes

- **SMTP**: Use app-specific passwords, enable 2FA on email accounts
- **SES**: Use IAM roles with minimal permissions, rotate access keys regularly
- **HTTP API**: Store API keys securely, use HTTPS endpoints only
- Store all credentials in environment variables or secure vaults

## SSH Port Configuration

All SSH-based providers (VPS and Nginx) support custom SSH ports for enhanced security.

### Configuration

Add `ssh_port` to your deployment configuration:

```yaml
deployments:
  - name: vps-deployment
    type: vps
    config:
      host: "your-vps.example.com"
      ssh_user: "root"
      ssh_key_path: "~/.ssh/id_rsa"
      ssh_port: 2222  # Custom SSH port (default: 22)
      # ... other config
```

### Benefits

- **Enhanced Security**: Use non-standard ports to reduce automated attacks
- **Firewall Compliance**: Work with restrictive firewall rules
- **Provider Requirements**: Some cloud providers require custom SSH ports

## S3 Deployment

Upload build artifacts to Amazon S3 buckets.

### Features

- **AWS S3 Integration**: Direct upload to S3 buckets
- **IAM Authentication**: Secure authentication using AWS credentials
- **Custom Bucket Paths**: Organize artifacts with custom prefixes
- **Public/Private Access**: Control artifact visibility

### Configuration

Add an `s3` deployment to your pipeline:

```yaml
deployments:
  - name: s3-upload
    type: s3
    config:
      bucket: "my-build-artifacts"
      region: "us-east-1"
      key: "artifacts/my-app-${BUILD_ID}.tar.gz"
      acl: "private"  # or "public-read"
```

### Prerequisites

1. **AWS Credentials**: Configure AWS access keys or IAM roles
2. **S3 Bucket**: Create destination bucket with appropriate permissions
3. **IAM Permissions**: `s3:PutObject` permission for the target bucket

### Example Pipeline

See `testdata/config/s3-deployment.yaml` for a complete example.

### Security Notes

- Use IAM roles with minimal required permissions
- Enable bucket versioning for artifact history
- Configure bucket policies for access control
- Use private ACLs by default

## Features

- Define pipelines in YAML with multi-step builds
- Execute steps in Docker containers for isolation
- Support for bash scripts, file creation, environment variables
- Job queue with background processing
- HTTP API for pipeline and job management
- Git repository cloning and branch checkout
- Port exposure for services
- **VPS Deployment**: Deploy to remote servers via SSH
- **Nginx Configuration**: Automatic web server setup
- **Email Notifications**: Send build artifacts via SMTP
- **S3 Storage**: Upload artifacts to cloud storage
- **SSH Port Support**: Custom SSH ports for all providers

9. **Reliability**
   - Job retry mechanisms
   - Timeout handling
   - Graceful shutdown
   - Backup and recovery

10. **Extensibility**
    - Plugin system for custom steps
    - Support for additional languages/frameworks
    - Custom Docker images
    - Integration with external tools
