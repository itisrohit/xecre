# Xecre

A local Python and JavaScript code runner written in Go, with Docker sandboxes and a plain HTML playground.

## Features
- **Warm Container Pooling**: A small pool of unused sandboxes reduces startup work on the request path.
- **HTML Playground**: Edit code, run it, and inspect output and measured timings at the same origin.
- **Async Replenishment**: Background goroutines automatically replenish the pool after use.
- **Execution Limits**: CPU, memory, process, output, request-size, and wall-time budgets.
- **Bounded Concurrency**: Go channels limit admitted requests and sandbox capacity.

## Architecture
- **API**: Lightweight `net/http` server.
- **Engine**: Docker SDK orchestrator (v0.3.0) with persistent container pools.
- **Replenishment**: One builder per language targets two ready containers; consumed containers are discarded.
- **Isolation**: Ephemeral sandboxes with networking disabled. Only the API receives the host Docker socket.
- **Lifecycle**: App/session labels identify owned containers for shutdown cleanup and startup recovery.

## Getting Started
With Docker running, pull the execution runtimes once, then start the app:
```bash
docker pull python:3.11-alpine
docker pull node:20-alpine
docker compose up --build -d
```

Open http://localhost:8080. Stop with `docker compose stop`.

The Docker build uses Go 1.26; running the benchmark locally requires Go 1.26.1 or a compatible toolchain. This deployment uses one API instance. The HTTP page can become available before both execution pools are ready.

### Local resource budget and lifecycle

This is a single-API local deployment: two ready containers per language, six live owned sandboxes maximum, and four admitted execution requests maximum. Excess requests receive HTTP 429. Each sandbox has a 256 MiB memory limit, a half-core CPU quota, a 64-process limit, and disabled networking. Requests have a ten-second engine deadline, a 128 KiB JSON body limit, and a combined 1 MiB output limit. These are initial laptop defaults, not measured optimal settings.

New sandboxes carry app and session ownership labels. Graceful shutdown drains HTTP requests, stops pool creation, and removes owned sandboxes. On startup the single API reconciles leftovers labeled `io.xecre.app=xecre-local`; do not run multiple API instances using that same app ID. Unlabeled legacy containers are never automatically removed. Failed removals retain capacity rather than allowing unbounded replacement. A forced kill may leave containers until the next startup reconciliation.

The sandbox containers are created directly through the host Docker daemon; limits on the API service alone do not constrain them. Docker containers are not a complete security boundary for a public untrusted-code service.

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
Run the included benchmark against a running local API:
```bash
go run scripts/benchmark.go
```
It sends 10 Python requests at concurrency 2 and reports each request's observed client-side time, mean, median, p95, total wall time, and failures. Use `-requests 20 -concurrency 1` for a small sequential baseline. Results depend on your machine, Docker, and system load; treat them as a local sample, not a universal performance guarantee. The client-side measurement includes HTTP overhead and any wait for a pooled container. Latency statistics cover successful HTTP responses; failures are reported separately. Check output correctness separately before performance testing.

## Performance Note
The API's `time_taken` field is the server-side duration for one request, measured from when the engine accepts it through container-pool wait and output collection. It is encoded as a Go duration in nanoseconds. The UI shows client round-trip time and exposes this server duration under **Timing details**. Memory usage is not currently measured; `memory_used` is retained in the response for compatibility and should not be interpreted as a measurement.

`pool_wait` measures sandbox acquisition; `execution_time` covers the remaining Docker execution calls, output collection, and exit inspection. All three durations use nanoseconds. They do not represent CPU time or pure interpreter runtime. Asynchronous container removal is outside the request timing.

The latest local sample completed 80 measured requests with correct output and zero failures. Mean round-trip latency ranged from 446 ms to 1,243 ms across the tested languages and concurrency levels. See [benchmark methodology and results](docs/BENCHMARKS.md) for environment, percentiles, and limitations.

## API behavior

`POST /execute` accepts `language` (`python` or `javascript`) and nonempty `code`. The legacy `input` and `timeout` request fields are not implemented; execution uses the fixed ten-second engine deadline. The response includes `stdout`, `stderr`, the actual `exit_code`, and timing fields. A program exiting nonzero still produces HTTP 200 with its exit code.

Validation errors return HTTP 400; admission exhaustion returns 429; engine deadline expiry returns 504. Other execution failures, including output-limit violations, return 500. Error responses are JSON objects with an `error` string.

## Development checks

```bash
go test -race ./...
go vet ./...
```

The focused engine tests use a fake Docker HTTP endpoint and do not start sandbox workloads.
