# Xecre

A high-performance, containerized code execution engine written in Go and Docker.

## Features
- Multi-language support (Python, Node.js)
- Sandbox isolation using Docker containers
- Automatic image pulling for sandboxes
- Lightweight, multi-stage API build (~20MB)

## Architecture
- **API**: Lightweight `net/http` server
- **Engine**: Docker SDK orchestrator (v0.3.0)
- **Isolation**: Ephemeral containers with host Docker socket mapping

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
