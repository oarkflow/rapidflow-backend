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
