# Xecre

A high-performance, containerized code execution engine written in Go and Docker.

## Features
- **Asynchronous Warm Container Pooling**: Pre-started sandboxes for near-instant execution.
- **Latency Optimization**: Sub-100ms request handling via Docker `Exec` bypassing.
- **Async Replenishment**: Background goroutines automatically replenish the pool after use.
- **Sandbox Isolation**: Full container-level isolation for Python and Node.js.
- **Concurrent Execution**: Orchestrated via Go channels for high RPS.

## Architecture
- **API**: Lightweight `net/http` server.
- **Engine**: Docker SDK orchestrator (v0.3.0) with persistent container pools.
- **Replenishment**: Asynchronous background workers managing a buffer (size 10) of ready containers.
- **Isolation**: Ephemeral sandboxes with host Docker socket mapping.

## Getting Started
Run the entire system with one command:
```bash
docker compose up --build -d
```

## Usage
Send a POST request to `/execute` with the following JSON structure:

```json
{
  "language": "python",
  "code": "print('Hello, Xecre!')"
}
```

### Example Tests

**Python**
```bash
curl -X POST http://localhost:8080/execute \
-H "Content-Type: application/json" \
-d '{"language": "python", "code": "print(\"Python is live\")"}'
```

**JavaScript**
```bash
curl -X POST http://localhost:8080/execute \
-H "Content-Type: application/json" \
-d '{"language": "javascript", "code": "console.log(\"Node is live\")"}'
```

## Benchmarks
You can run the included benchmark script to verify performance on your local system:
```bash
go run scripts/benchmark.go
```

## Performance Note
Current benchmarks show an average execution latency of **~84ms** per request in concurrent loads.
