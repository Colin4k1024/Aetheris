# Quickstart

This guide gets Aetheris running locally and submits one agent job through the simplest supported path: a tiny HTTP agent wrapped by Aetheris.

Embedded mode uses local stores and does not require Docker, PostgreSQL, or Redis.

## Requirements

- Go 1.26.1+
- Git
- Python 3 for the tiny mock HTTP agent below

## 1. Start A Tiny HTTP Agent

In terminal 1, run a local mock agent:

```bash
python3 -c '
import json
from http.server import BaseHTTPRequestHandler, HTTPServer

class Handler(BaseHTTPRequestHandler):
    def do_POST(self):
        length = int(self.headers.get("content-length", "0"))
        payload = json.loads(self.rfile.read(length) or b"{}")
        body = {
            "answer": "Aetheris received: " + payload.get("message", ""),
            "final": True,
            "metadata": {"mock": True},
        }
        encoded = json.dumps(body).encode()
        self.send_response(200)
        self.send_header("content-type", "application/json")
        self.send_header("content-length", str(len(encoded)))
        self.end_headers()
        self.wfile.write(encoded)

HTTPServer(("127.0.0.1", 9000), Handler).serve_forever()
'
```

## 2. Create A Quickstart Runtime Config

In terminal 2, clone the repo and create a temporary embedded config that registers the mock agent:

```bash
git clone https://github.com/Colin4k1024/Aetheris.git
cd Aetheris

go mod download
cp configs/api.embedded.yaml /tmp/aetheris-api.quickstart.yaml
cat >> /tmp/aetheris-api.quickstart.yaml <<'YAML'

agents:
  agents:
    quickstart_http:
      type: "external_http"
      description: "Quickstart mock HTTP agent"
      external:
        url: "http://127.0.0.1:9000/invoke"
        timeout: "30s"
YAML
```

## 3. Start Aetheris

For the smallest local loop, start only the API. In embedded mode, the API process can execute jobs locally.

```bash
API_CONFIG_PATH=/tmp/aetheris-api.quickstart.yaml \
MODEL_CONFIG_PATH=configs/model.yaml \
go run ./cmd/api
```

In terminal 3, check that the API is up:

```bash
curl http://localhost:8080/api/health
```

Expected shape:

```json
{
  "status": "ok"
}
```

## 4. Submit A Job

Submit a message to the `quickstart_http` agent:

```bash
curl -X POST http://localhost:8080/api/agents/quickstart_http/message \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: quickstart-1" \
  -d '{"message":"Say hello from Aetheris"}'
```

The response includes a `job_id`:

```json
{
  "status": "accepted",
  "agent_id": "quickstart_http",
  "job_id": "..."
}
```

## 5. Inspect The Job

Replace `<job_id>` with the value returned above.

```bash
curl http://localhost:8080/api/jobs/<job_id>
curl http://localhost:8080/api/jobs/<job_id>/events
curl http://localhost:8080/api/jobs/<job_id>/trace
```

Open the trace page in a browser:

```text
http://localhost:8080/api/jobs/<job_id>/trace/page
```

## 6. Connect Your Real HTTP Agent

If you already have an agent in Python, JavaScript, Go, or another runtime, expose one HTTP endpoint and register it as `external_http`.

Add this under the top-level `agents` field in the active runtime config:

```yaml
agents:
  agents:
    customer_support_bot:
      type: "external_http"
      description: "Existing customer support agent"
      external:
        url: "http://localhost:9000/invoke"
        timeout: "120s"
        token_env: "CUSTOMER_BOT_TOKEN"
```

Your service should accept:

```json
{
  "message": "user request",
  "session_id": "session id",
  "metadata": {
    "agent_id": "customer_support_bot",
    "job_id": "job id",
    "idempotency_key": "stable key"
  }
}
```

And return:

```json
{
  "answer": "final response",
  "final": true,
  "metadata": {}
}
```

Restart Aetheris after changing config.

Submit to the external agent:

```bash
curl -X POST http://localhost:8080/api/agents/customer_support_bot/message \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: customer-support-demo-1" \
  -d '{"message":"Check order status for order-123"}'
```

More detail: [../adapters/external-http-agent.md](../adapters/external-http-agent.md)

## 7. Stop The Runtime

Press `Ctrl-C` in the Aetheris terminal and the mock-agent terminal.

## Next Steps

| Goal | Read |
| ---- | ---- |
| Existing HTTP agent intake | [../adapters/external-http-agent.md](../adapters/external-http-agent.md) |
| Eino examples | [../adapters/eino-examples.md](../adapters/eino-examples.md) |
| Runtime guarantees | [runtime-guarantees.md](runtime-guarantees.md) |
| Docker Compose deployment | [deployment.md](deployment.md) |
