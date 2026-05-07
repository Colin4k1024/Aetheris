/**
 * types.ts — Shared domain types for the OpenClaw ↔ Aetheris adapter
 */

// ---------------------------------------------------------------------------
// Aetheris API types
// ---------------------------------------------------------------------------

/** POST /api/agents/:id/message response */
export interface AgentMessageResponse {
  status: "accepted";
  agent_id: string;
  job_id: string;
  runtime_submission: {
    legacy_facade: boolean;
    canonical_api: string;
    job_id?: string;
    run_id?: string;
    run_status: "created" | "best_effort" | "disabled";
  };
}

/** Single job event from GET /api/jobs/:id/events */
export interface JobEvent {
  id: number;
  job_id: string;
  type: JobEventType;
  payload: Record<string, unknown> | null;
  created_at: string;
}

/** Terminal and key event types emitted by the Aetheris job store */
export type JobEventType =
  | "job_created"
  | "plan_generated"
  | "node_started"
  | "node_finished"
  | "command_emitted"
  | "command_committed"
  | "tool_called"
  | "tool_returned"
  | "step_committed"
  | "job_completed"
  | "job_failed"
  | "job_cancelled"
  | "job_running"
  | "job_parked"
  | "step_finished"
  | "step_failed"
  | string; // allow future extensions

/** Subset of job event types that indicate a terminal state */
export const TERMINAL_EVENT_TYPES = new Set<JobEventType>([
  "job_completed",
  "job_failed",
  "job_cancelled",
]);

/** GET /api/jobs/:id/events response envelope */
export interface JobEventsResponse {
  job_id: string;
  events: JobEvent[];
}

// ---------------------------------------------------------------------------
// OpenClaw adapter types
// ---------------------------------------------------------------------------

/** Inbound message from an OpenClaw platform connector (Telegram, WhatsApp, etc.) */
export interface OpenClawInboundMessage {
  /** Platform identifier: "telegram" | "whatsapp" | "discord" | "slack" */
  platform: string;
  /** Unique chat/channel ID on the source platform */
  chatId: string;
  /** Human-readable sender name (optional) */
  senderName?: string;
  /** The raw text sent by the user */
  text: string;
  /** ISO-8601 timestamp */
  timestamp: string;
  /** Optional opaque metadata from OpenClaw (e.g. message_id for dedup) */
  metadata?: Record<string, unknown>;
}

/** Outbound reply to be sent back through OpenClaw */
export interface OpenClawOutboundReply {
  platform: string;
  chatId: string;
  text: string;
  /** Whether this reply is a final answer or an intermediate status update */
  final: boolean;
}

// ---------------------------------------------------------------------------
// Adapter config
// ---------------------------------------------------------------------------

export interface AdapterConfig {
  /** Base URL of the Aetheris API (e.g. http://localhost:8080) */
  aetherisBaseUrl: string;
  /** Bearer token for Aetheris API auth (JWT or static API key) */
  aetherisToken: string;
  /**
   * Which Aetheris agent to route messages to.
   * Can be a static string or a function that resolves per-message.
   */
  agentId: string | ((msg: OpenClawInboundMessage) => string);
  /**
   * How long (ms) to wait for a job to reach a terminal state before
   * returning a timeout reply. Default: 120_000 (2 min)
   */
  jobTimeoutMs?: number;
  /**
   * Polling interval (ms) while waiting for job events. Default: 1_500
   */
  pollIntervalMs?: number;
  /**
   * Port the adapter's Express server listens on. Default: 3100
   */
  serverPort?: number;
}
