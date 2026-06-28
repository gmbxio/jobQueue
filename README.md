# JobQueue — Distributed Async Job Queue in Go

A production-style distributed job queue system built in Go, backed by PostgreSQL and Redis, with a pluggable AI inference backend via [Gollama](https://github.com/gmbxio/gollama).

## Architecture

```
Client (curl / API)
        │
        ▼
┌───────────────┐       ┌─────────┐
│  Server API   │──────▶│  Redis  │  (task spike / message broker)
│  (Go + mux)  │       └────┬────┘
└───────┬───────┘            │
        │                    ▼
        │           ┌───────────────┐       ┌─────────────┐
        │           │    Worker     │──────▶│   Gollama   │  (LLM inference)
        │           │   Engine     │       │  FastAPI    │
        │           └───────┬───────┘       └─────────────┘
        │                   │
        ▼                   ▼
┌──────────────────────────────┐
│         PostgreSQL           │  (job state + results)
└──────────────────────────────┘
```

**Flow:**
1. Client POSTs a job → Server saves it to PostgreSQL as `pending`, pushes job ID to Redis
2. Worker blocks on Redis queue (BRPop), picks up job ID
3. Worker fetches payload from PostgreSQL, forwards to Gollama for LLM inference
4. Worker writes result back to PostgreSQL as `completed` or `failed`
5. Client polls `GET /jobs/{id}` to check status and retrieve result

## Tech Stack

| Layer | Technology |
|---|---|
| API Server | Go, gorilla/mux |
| Message Broker | Redis 7 |
| Database | PostgreSQL 15 |
| LLM Backend | Gollama (FastAPI + Ollama) |
| Containerization | Docker, Docker Compose |

## Project Structure

```
jobQueue/
├── cmd/
│   ├── server/
│   │   ├── main.go        # HTTP server, router, DB/Redis init
│   │   └── handlers.go    # CreateJob, GetJobStatus, ListJobs handlers
│   └── worker/
│       └── main.go        # Blocking worker loop, Gollama dispatcher
├── internal/
│   ├── db/
│   │   └── db.go          # PostgreSQL connection + schema init
│   └── queue/
│       └── queue.go       # Redis connection init
├── docker-compose.yml
├── go.mod
└── go.sum
```

## Database Schema

```sql
CREATE TABLE jobs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_type     VARCHAR(50) NOT NULL,
    payload       JSONB NOT NULL,
    status        VARCHAR(20) DEFAULT 'pending',  -- pending | running | completed | failed
    result        JSONB,
    error_message TEXT,
    retry_count   INT DEFAULT 0,
    created_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

## Prerequisites

- [Go 1.21+](https://golang.org/dl/)
- [Docker + Docker Compose](https://docs.docker.com/get-docker/)
- [Gollama](https://github.com/gmbxio/gollama) running locally with Ollama

## Getting Started

### 1. Start PostgreSQL and Redis

```bash
docker compose up -d
```

Verify both containers are running:
```bash
docker ps
```

### 2. Start the Gollama inference backend

In a separate terminal, navigate to your Gollama project and run:
```bash
uvicorn api:app --port 8000
```

### 3. Start the API Server

```bash
go run cmd/server/main.go cmd/server/handlers.go
```

You should see:
```
Bakery Cashier API is waking up...
Successfully connected to PostgreSQL!
PostgreSQL database schema verified.
Successfully connected to Redis!
Server is running on port 8080...
```

### 4. Start the Worker

In another terminal:
```bash
go run cmd/worker/main.go
```

You should see:
```
Worker Engine initialized...
Successfully connected to PostgreSQL!
PostgreSQL database schema verified.
Successfully connected to Redis!
Worker listening to queue. Forwarding tasks to Gollama at: http://localhost:8000/generate
```

## API Reference

### POST /jobs — Submit a job

```bash
curl -X POST http://localhost:8080/jobs \
     -H "Content-Type: application/json" \
     -d '{
       "task_type": "gollama_inference",
       "payload": {
         "prompt": "Write a Python script to reverse a string",
         "model": "gemma3:1b"
       }
     }'
```

**Response:**
```json
{
  "job_id": "a0fe453c-1daf-4774-8414-10cd55ba22d6",
  "status": "pending",
  "message": "Job safely queued!"
}
```

---

### GET /jobs/{id} — Get job status and result

```bash
curl http://localhost:8080/jobs/a0fe453c-1daf-4774-8414-10cd55ba22d6
```

**Response (completed):**
```json
{
  "id": "a0fe453c-1daf-4774-8414-10cd55ba22d6",
  "task_type": "gollama_inference",
  "payload": { "model": "gemma3:1b", "prompt": "Write a Python script to reverse a string" },
  "status": "completed",
  "result": {
    "model": "gemma3:1b",
    "response": "```python\ndef reverse_string(s):\n    return s[::-1]\n```",
    "done": false,
    "queue_execution_latency_seconds": 28.58
  }
}
```

---

### GET /jobs — List all jobs

```bash
curl http://localhost:8080/jobs
```

**Response:**
```json
{
  "total": 8,
  "jobs": [
    {
      "id": "a0fe453c-1daf-4774-8414-10cd55ba22d6",
      "task_type": "gollama_inference",
      "status": "completed",
      "created_at": "2026-06-28T10:40:31.567549Z",
      "updated_at": "2026-06-28T10:41:00.174749Z"
    }
  ]
}
```

---

### GET /health — Health check

```bash
curl http://localhost:8080/health
```

## Supported Models (via Gollama + Ollama)

Any model pulled via `ollama pull` works. Tested with:

- `gemma3:1b`
- `llama3.2:1b`
- `deepseek-r1:1.5b`
- `qwen2.5:1.5b`

Check available models:
```bash
ollama list
```

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `GOLLAMA_URL` | `http://localhost:8000/generate` | Gollama inference endpoint |

## Stopping the System

```bash
# Stop containers (keeps data)
docker compose down

# Stop containers and wipe database
docker compose down -v
```