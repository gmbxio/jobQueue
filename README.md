# JobQueue — Distributed Async Job Queue in Go

A production-style distributed job queue system built in Go, backed by PostgreSQL and Redis, with Groq LPU as the AI inference backend. Features a real-time dashboard UI showing live job status as it flows through the pipeline.

**Live Demo:** [jobqueue.up.railway.app](https://jobqueue-production-9e90.up.railway.app/)

## Architecture

```
Client (Browser / curl)
        │
        ▼
┌───────────────┐       ┌─────────┐
│  Server API   │──────▶│  Redis  │  (message broker)
│  (Go + mux)  │       └────┬────┘
└───────┬───────┘            │
        │                    ▼
        │           ┌───────────────┐       ┌─────────────┐
        │           │    Worker     │──────▶│  Groq API   │  (LLM inference)
        │           │  (x3 replicas)│       │  LPU cloud  │
        │           └───────┬───────┘       └─────────────┘
        │                   │
        ▼                   ▼
┌──────────────────────────────┐
│         PostgreSQL           │  (job state + results)
└──────────────────────────────┘
```

**Flow:**
1. Client submits a prompt via dashboard or REST API
2. Server saves job to PostgreSQL as `pending`, pushes job ID to Redis
3. One of 3 worker replicas picks up the job via BRPop
4. Worker fetches payload from PostgreSQL, forwards to Groq API for inference
5. Worker writes result back to PostgreSQL as `completed` or `failed`
6. Dashboard polls every 2 seconds and displays the result live

## Tech Stack

| Layer | Technology |
|---|---|
| API Server | Go, gorilla/mux |
| Message Broker | Redis 7 |
| Database | PostgreSQL 15 |
| LLM Inference | Groq API (LPU cloud) |
| Frontend | Vanilla HTML/CSS/JS |
| Containerization | Docker, Docker Compose |

## Project Structure

```
jobQueue/
├── cmd/
│   ├── server/
│   │   ├── main.go        # HTTP server, router, DB/Redis init
│   │   ├── handlers.go    # CreateJob, GetJobStatus, ListJobs, DeleteJob handlers
│   │   └── static/
│   │       └── index.html # Real-time dashboard UI
│   └── worker/
│       └── main.go        # Blocking worker loop, Groq API dispatcher
├── internal/
│   ├── db/
│   │   └── db.go          # PostgreSQL connection + schema init
│   └── queue/
│       └── queue.go       # Redis connection init
├── docker-compose.yml
├── Dockerfile.api
├── Dockerfile.worker
├── .env.example
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

## Getting Started

### Prerequisites

- [Go 1.21+](https://golang.org/dl/)
- [Docker + Docker Compose](https://docs.docker.com/get-docker/)
- [Groq API key](https://console.groq.com) — free, no credit card required

### 1. Clone the repo

```bash
git clone https://github.com/gmbxio/jobQueue.git
cd jobQueue
```

### 2. Set up environment variables

```bash
cp .env.example .env
# Edit .env and add your Groq API key
```

`.env`:
```
GROQ_API_KEY=your_groq_api_key_here
```

### 3. Build and run with Docker

```bash
docker compose build
docker compose up -d
```

### 4. Open the dashboard

```
http://localhost:8080
```

That's it. The API server, 3 worker replicas, PostgreSQL, and Redis all start together.

## API Reference

### POST /jobs — Submit a job

```bash
curl -X POST http://localhost:8080/jobs \
     -H "Content-Type: application/json" \
     -d '{
       "task_type": "groq_inference",
       "payload": {
         "prompt": "Write a Python script to reverse a string",
         "model": "llama-3.1-8b-instant"
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
  "task_type": "groq_inference",
  "payload": { "model": "llama-3.1-8b-instant", "prompt": "Write a Python script to reverse a string" },
  "status": "completed",
  "result": {
    "model": "llama-3.1-8b-instant",
    "response": "def reverse_string(s):\n    return s[::-1]",
    "prompt_tokens": 18,
    "completion_tokens": 42,
    "total_tokens": 60,
    "queue_execution_latency_seconds": 1.24
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
  "total": 3,
  "jobs": [
    {
      "id": "a0fe453c-1daf-4774-8414-10cd55ba22d6",
      "task_type": "groq_inference",
      "status": "completed",
      "created_at": "2026-06-28T10:40:31.567549Z",
      "updated_at": "2026-06-28T10:41:00.174749Z"
    }
  ]
}
```

---

### DELETE /jobs/{id} — Delete a job

```bash
curl -X DELETE http://localhost:8080/jobs/a0fe453c-1daf-4774-8414-10cd55ba22d6
```

---

### GET /health — Health check

```bash
curl http://localhost:8080/health
```

## Supported Models (via Groq free tier)

| Model | Best For |
|---|---|
| `llama-3.1-8b-instant` | Fast tasks, high volume |
| `llama-3.3-70b-versatile` | Complex reasoning |
| `meta-llama/llama-4-scout-17b-16e-instruct` | Long context, vision |
| `openai/gpt-oss-20b` | General purpose |
| `openai/gpt-oss-120b` | Premium reasoning |
| `qwen/qwen3-32b` | Multilingual |

Get a free API key at [console.groq.com](https://console.groq.com) — no credit card required.

## Environment Variables

| Variable | Required | Description |
|---|---|---|
| `GROQ_API_KEY` | ✅ Yes | Groq API key from console.groq.com |
| `DB_HOST` | Auto | Set to `postgres` in Docker |
| `REDIS_HOST` | Auto | Set to `redis` in Docker |

## Stopping the System

```bash
# Stop containers (keeps data)
docker compose down

# Stop containers and wipe database
docker compose down -v
```