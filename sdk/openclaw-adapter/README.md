# OpenClaw → Aetheris Adapter

A lightweight Node.js adapter that bridges **OpenClaw** (the self-hosted messaging gateway) with **Aetheris** (the durable agent execution runtime).

## Architecture

```
User (Telegram / WhatsApp / Discord / Slack)
        │
        ▼
   OpenClaw Gateway
        │  POST /webhook/message
        ▼
  openclaw-adapter  ←── this package
        │  POST /api/agents/:id/message
        ▼
   Aetheris API
        │  creates durable Job
        ▼
   Aetheris Worker  (Eino ADK agent, tools, replays)
        │  polls GET /api/jobs/:id/events
        ▼
  openclaw-adapter
        │  returns reply JSON
        ▼
   OpenClaw Gateway  → platform reply
```

### Phase integration map

| Phase | Link | Description |
|-------|------|-------------|
| **Phase 1** ✅ | OpenClaw → Aetheris | This package |
| Phase 2 | Aetheris → Hermes API Server | Aetheris HTTP tool calls `http://hermes:8642/v1/chat/completions` |
| Phase 3 | Hermes MCP bridge | `hermes mcp serve` exposes messaging tools to Aetheris jobs |

---

## Quick Start

### 1. Install

```bash
cd sdk/openclaw-adapter
npm install
```

### 2. Configure

```bash
cp .env.example .env
# Edit .env with your values
```

```
AETHERIS_BASE_URL=http://localhost:8080
AETHERIS_TOKEN=your-jwt-or-api-key
AETHERIS_AGENT_ID=react
PORT=3100
JOB_TIMEOUT_MS=120000
POLL_INTERVAL_MS=1500
```

### 3. Start Aetheris

```bash
# from repo root
make run
```

### 4. Start the adapter

```bash
npm run dev          # development (ts-node)
# or
npm run build && npm start   # production
```

### 5. Test manually

```bash
curl -X POST http://localhost:3100/webhook/message \
  -H 'Content-Type: application/json' \
  -d '{
    "platform": "telegram",
    "chatId": "chat-123",
    "senderName": "Alice",
    "text": "What is 2+2?"
  }'
```

Expected response:
```json
{
  "platform": "telegram",
  "chatId": "chat-123",
  "text": "4",
  "final": true
}
```

---

## Webhook API

### `POST /webhook/message`

Accepts an inbound message and returns the agent reply after job completion.

**Request body**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `platform` | string | ✅ | Source platform (`telegram`, `whatsapp`, `discord`, `slack`, …) |
| `chatId` | string | ✅ | Chat or channel ID on the source platform |
| `text` | string | ✅ | User's message text |
| `senderName` | string | — | Human-readable sender name |
| `timestamp` | string | — | ISO-8601; defaults to `now()` if omitted |
| `metadata` | object | — | Opaque platform metadata (e.g. `messageId` for dedup) |

**Response** (synchronous — waits for job completion)

```json
{
  "platform": "telegram",
  "chatId": "chat-123",
  "text": "<agent reply>",
  "final": true
}
```

**Error replies** (HTTP 200, `final: true`)

The adapter never crashes the webhook — transient errors and timeouts are returned as a user-friendly text reply.

### `GET /health`

Liveness probe. Returns `{ "status": "ok", "timestamp": "..." }`.

---

## Idempotency

Every inbound message is hashed into a deterministic **Idempotency-Key** (platform + chatId + text + timestamp).  The key is forwarded to the `POST /api/agents/:id/message` call.

If OpenClaw retries a webhook delivery, Aetheris detects the duplicate key and returns the existing `job_id` immediately — no duplicate jobs are created.

---

## Dynamic agent routing

To route different platforms or users to different Aetheris agents, set `agentId` to a function:

```ts
import { startServer } from "@aetheris/openclaw-adapter";

startServer({
  aetherisBaseUrl: "http://localhost:8080",
  aetherisToken: process.env.AETHERIS_TOKEN!,
  agentId: (msg) => {
    if (msg.platform === "whatsapp") return "manus";
    if (msg.chatId === "vip-chat-999") return "deer";
    return "react";
  },
});
```

---

## Docker / docker-compose

See [`docker-compose.yml`](./docker-compose.yml) for a full local stack:

```bash
docker compose up
```

This starts:
- `aetheris-api` (port 8080)
- `aetheris-worker`
- `openclaw-adapter` (port 3100)
- PostgreSQL (port 5432)

---

## Running tests

```bash
npm test
```

Test coverage:
- `AetherisClient` — sendMessage, getJobEvents, waitForTerminalEvent, extractReplyText
- `MessageHandler` — happy path, error, timeout, dynamic agentId
- Express server — health, webhook validation, error handling

---

## Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `AETHERIS_BASE_URL` | — | **Required.** Aetheris API base URL |
| `AETHERIS_TOKEN` | — | **Required.** Bearer token (JWT or static API key) |
| `AETHERIS_AGENT_ID` | — | **Required.** Agent ID from `configs/agents.yaml` |
| `PORT` | `3100` | Adapter listen port |
| `JOB_TIMEOUT_MS` | `120000` | Max wait for job completion |
| `POLL_INTERVAL_MS` | `1500` | Polling interval |
