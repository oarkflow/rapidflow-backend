# RapidFlow Backend API

## New Build Management Endpoints

### Get Job Details with Output Streams
**GET** `/jobs/:id/details`

Returns detailed job information including pipeline configuration, all steps with their outputs, and environment variables.

### Get Job Logs
**GET** `/jobs/:id/logs`

Returns all logs for a specific job in a structured format.

**Response:**
```json
{
  "job_id": 1,
  "status": "success",
  "logs": [
    {
      "step_id": 1,
      "order_num": 1,
      "type": "bash",
      "content": "go mod download",
      "status": "success",
      "output": "go: downloading github.com/gofiber/fiber/v2 v2.50.0\ngo: downloading github.com/jmoiron/sqlx v1.3.5\n",
      "created_at": "2025-09-26T10:00:05Z"
    },
    {
      "step_id": 2,
      "order_num": 2,
      "type": "bash",
      "content": "go build -o app .",
      "status": "success",
      "output": "Build completed successfully\n",
      "created_at": "2025-09-26T10:01:30Z"
    }
  ]
}
```

### Stream Job Logs (Real-time)
**GET** `/jobs/:id/logs/stream?follow=true&since=123`

Streams logs for a job in real-time. Useful for monitoring builds as they happen.

**Query Parameters:**
- `follow=true` - Keep connection open and stream new logs as they arrive
- `since=<step_id>` - Only return logs from steps after the specified step ID

**Response:** Plain text stream
```
=== Step 1 (bash) ===
go: downloading github.com/gofiber/fiber/v2 v2.50.0
go: downloading github.com/jmoiron/sqlx v1.3.5

=== Step 2 (bash) ===
Build completed successfully

=== Build completed ===
```

### Get Step Logs
**GET** `/steps/:id/logs`

Returns logs for a specific build step.

**Response:**
```json
{
  "step_id": 1,
  "job_id": 1,
  "order_num": 1,
  "type": "bash",
  "content": "go mod download",
  "status": "success",
  "output": "go: downloading github.com/gofiber/fiber/v2 v2.50.0\ngo: downloading github.com/jmoiron/sqlx v1.3.5\n",
  "created_at": "2025-09-26T10:00:05Z"
}
```

**Response:**
```json
{
  "job": {
    "id": 1,
    "pipeline_id": 1,
    "status": "success",
    "branch": "main",
    "repo_name": "https://github.com/user/repo",
    "language": "golang",
    "version": "1.21",
    "folder": "./my-go-app",
    "expose_ports": false,
    "cancelled": false,
    "container_id": "abc123",
    "created_at": "2025-09-26T10:00:00Z",
    "started_at": "2025-09-26T10:00:05Z",
    "finished_at": "2025-09-26T10:02:30Z"
  },
  "pipeline": {
    "id": 1,
    "name": "Go Build Pipeline",
    "config": "name: Go Build Pipeline\nlanguage: golang\n..."
  },
  "steps": [
    {
      "id": 1,
      "job_id": 1,
      "order_num": 1,
      "type": "bash",
      "content": "go mod download",
      "status": "success",
      "output": "go: downloading github.com/gofiber/fiber/v2 v2.50.0\n...",
      "created_at": "2025-09-26T10:00:05Z"
    },
    {
      "id": 2,
      "job_id": 1,
      "order_num": 2,
      "type": "bash",
      "content": "go build -o app .",
      "status": "success",
      "output": "Build completed successfully\n",
      "created_at": "2025-09-26T10:00:05Z"
    }
  ],
  "environments": [
    {
      "id": 1,
      "job_id": 1,
      "key": "CGO_ENABLED",
      "value": "0"
    }
  ]
}
```

### Cancel Running Job
**POST** `/jobs/:id/cancel`

Cancels a running or pending job. If the job is currently executing, it will be stopped gracefully.

**Response:**
```json
{
  "message": "job cancelled successfully"
}
```

**Error Responses:**
- `400` - Job cannot be cancelled (already completed/failed)
- `404` - Job not found

### Retry/Re-run Job
**POST** `/jobs/:id/retry`

Creates a new job based on an existing job's configuration. Useful for retrying failed builds or re-running successful builds.

**Response:**
```json
{
  "id": 5,
  "pipeline_id": 1,
  "status": "pending",
  "branch": "main",
  "repo_name": "https://github.com/user/repo",
  "language": "golang",
  "version": "1.21",
  "folder": "./my-go-app",
  "expose_ports": false,
  "cancelled": false,
  "container_id": null,
  "created_at": "2025-09-26T11:00:00Z",
  "started_at": null,
  "finished_at": null
}
```

**Error Responses:**
- `400` - Cannot retry running or pending job
- `404` - Original job not found

## Job Status Values

- `pending` - Job is queued and waiting to start
- `running` - Job is currently executing
- `success` - Job completed successfully
- `failed` - Job completed with errors
- `cancelled` - Job was cancelled by user

## Step Status Values

- `pending` - Step is waiting to execute
- `running` - Step is currently executing
- `success` - Step completed successfully
- `failed` - Step completed with errors
- `cancelled` - Step was cancelled (job was cancelled)

## Build Output Streaming

The job details endpoint provides access to all build output streams:

1. **Step Outputs**: Each step's `output` field contains the complete console output from that step's execution
2. **Real-time Monitoring**: Poll the job details endpoint to get updated status and output as the build progresses
3. **Error Tracking**: Failed steps will have their error output captured in the `output` field

## Usage Examples

### Monitor a Build in Progress
```bash
# Start a build
curl -X POST http://localhost:3000/pipelines/1/jobs

# Monitor progress with structured logs
curl http://localhost:3000/jobs/1/logs

# Stream logs in real-time (keeps connection open)
curl "http://localhost:3000/jobs/1/logs/stream?follow=true"

# Get only new logs since step 5
curl "http://localhost:3000/jobs/1/logs/stream?since=5"

# Cancel if needed
curl -X POST http://localhost:3000/jobs/1/cancel
```

### Access Specific Step Logs
```bash
# Get logs for a specific step
curl http://localhost:3000/steps/123/logs

# Get all job details (includes logs)
curl http://localhost:3000/jobs/1/details
```

### Retry a Failed Build
```bash
# Check failed job logs
curl http://localhost:3000/jobs/1/logs

# Check specific failed step
curl http://localhost:3000/steps/123/logs

# Retry the job
curl -X POST http://localhost:3000/jobs/1/retry
```

## Log Endpoints Comparison

| Endpoint | Use Case | Format | Real-time |
|----------|----------|--------|-----------|
| `/jobs/:id/logs` | Get all job logs | JSON structured | No |
| `/jobs/:id/logs/stream` | Real-time monitoring | Plain text stream | Yes |
| `/jobs/:id/details` | Complete job info + logs | JSON with metadata | No |
| `/steps/:id/logs` | Single step logs | JSON structured | No |

## Real-time Log Streaming

The streaming endpoint (`/jobs/:id/logs/stream`) is perfect for:

- **CI/CD dashboards** - Show live build progress
- **Command line tools** - Follow build output like `tail -f`
- **Web applications** - Real-time log updates in browser
- **Debugging** - Watch builds fail in real-time

**Features:**
- Automatically closes when build completes
- Supports `since` parameter to avoid re-reading old logs
- Handles job cancellation gracefully
- Plain text format for easy parsing
