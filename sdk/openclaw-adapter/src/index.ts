/**
 * index.ts — Entry point
 *
 * Reads config from environment variables and starts the adapter server.
 *
 * Required env vars:
 *   AETHERIS_BASE_URL   Base URL of the Aetheris API (e.g. http://localhost:8080)
 *   AETHERIS_TOKEN      Bearer token (JWT or static API key)
 *   AETHERIS_AGENT_ID   Target agent ID defined in agents.yaml  (e.g. "react")
 *
 * Optional env vars:
 *   PORT                Adapter listen port                      (default: 3100)
 *   JOB_TIMEOUT_MS      Max wait for job completion in ms        (default: 120000)
 *   POLL_INTERVAL_MS    Polling interval in ms                   (default: 1500)
 */

import "dotenv/config";
import { AdapterConfig } from "./types";
import { startServer } from "./server";

function requireEnv(name: string): string {
  const value = process.env[name];
  if (!value) {
    console.error(`[startup] Required env var ${name} is not set`);
    process.exit(1);
  }
  return value;
}

const config: AdapterConfig = {
  aetherisBaseUrl: requireEnv("AETHERIS_BASE_URL"),
  aetherisToken: requireEnv("AETHERIS_TOKEN"),
  agentId: requireEnv("AETHERIS_AGENT_ID"),
  serverPort: process.env["PORT"] ? parseInt(process.env["PORT"], 10) : 3100,
  jobTimeoutMs: process.env["JOB_TIMEOUT_MS"]
    ? parseInt(process.env["JOB_TIMEOUT_MS"], 10)
    : 120_000,
  pollIntervalMs: process.env["POLL_INTERVAL_MS"]
    ? parseInt(process.env["POLL_INTERVAL_MS"], 10)
    : 1_500,
};

startServer(config);

// Re-export public API so the package can also be used as a library
export { AetherisClient } from "./aetheris-client";
export { MessageHandler } from "./handler";
export { createServer } from "./server";
export * from "./types";
