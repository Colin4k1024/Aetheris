/**
 * utils.ts — Small shared helpers
 */

import { createHash } from "crypto";
import { OpenClawInboundMessage } from "./types";

/**
 * Deterministic idempotency key from message fields.
 * Aetheris uses this (per agent + tenant) to deduplicate retried webhook
 * deliveries — the second POST returns the existing job_id immediately.
 *
 * Key space: platform:chatId:text:timestamp — truncated to 64 hex chars.
 */
export function buildIdempotencyKey(msg: OpenClawInboundMessage): string {
  const raw = [
    msg.platform,
    msg.chatId,
    msg.text,
    msg.timestamp,
    // If the source platform provides a stable message_id, prefer that.
    (msg.metadata?.messageId as string) ?? "",
  ].join("|");

  return createHash("sha256").update(raw).digest("hex").slice(0, 64);
}

/**
 * Simple structured logger — wraps console with a level prefix.
 * Replace with Pino / Winston in production.
 */
export function log(
  level: "info" | "warn" | "error",
  component: string,
  message: string,
  extra?: unknown
): void {
  const ts = new Date().toISOString();
  const prefix = `[${ts}] [${level.toUpperCase()}] [${component}]`;
  if (extra !== undefined) {
    console[level](`${prefix} ${message}`, extra);
  } else {
    console[level](`${prefix} ${message}`);
  }
}
