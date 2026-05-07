/**
 * aetheris-client.ts — Thin HTTP client for the Aetheris REST API
 *
 * Covers the two endpoints needed for Phase 1:
 *   POST /api/agents/:id/message  → submit a user message, get back a job_id
 *   GET  /api/jobs/:id/events     → poll the event stream until terminal
 */

import axios, { AxiosInstance, AxiosError } from "axios";
import {
  AgentMessageResponse,
  JobEventsResponse,
  JobEvent,
  TERMINAL_EVENT_TYPES,
} from "./types";

export class AetherisClient {
  private readonly http: AxiosInstance;

  constructor(baseUrl: string, token: string) {
    this.http = axios.create({
      baseURL: baseUrl.replace(/\/$/, ""),
      headers: {
        Authorization: `Bearer ${token}`,
        "Content-Type": "application/json",
      },
      timeout: 10_000,
    });
  }

  /**
   * POST /api/agents/:agentId/message
   *
   * Submits a user message to an Aetheris agent. Returns the accepted job_id.
   * Passes an optional Idempotency-Key header so repeated deliveries are safe.
   */
  async sendMessage(
    agentId: string,
    message: string,
    idempotencyKey?: string
  ): Promise<AgentMessageResponse> {
    const headers: Record<string, string> = {};
    if (idempotencyKey) {
      headers["Idempotency-Key"] = idempotencyKey;
    }
    const response = await this.http.post<AgentMessageResponse>(
      `/api/agents/${encodeURIComponent(agentId)}/message`,
      { message },
      { headers }
    );
    return response.data;
  }

  /**
   * GET /api/jobs/:jobId/events
   *
   * Returns all events recorded so far for the given job.
   */
  async getJobEvents(jobId: string): Promise<JobEventsResponse> {
    const response = await this.http.get<JobEventsResponse>(
      `/api/jobs/${encodeURIComponent(jobId)}/events`
    );
    return response.data;
  }

  /**
   * Polls GET /api/jobs/:jobId/events until a terminal event is found or
   * the timeout expires.
   *
   * @param jobId          The job to watch
   * @param timeoutMs      Give up after this many milliseconds (default 120 s)
   * @param pollIntervalMs How often to poll (default 1.5 s)
   * @returns              The terminal event, or null if timed out
   */
  async waitForTerminalEvent(
    jobId: string,
    timeoutMs = 120_000,
    pollIntervalMs = 1_500
  ): Promise<JobEvent | null> {
    const deadline = Date.now() + timeoutMs;
    while (Date.now() < deadline) {
      let events: JobEvent[];
      try {
        const res = await this.getJobEvents(jobId);
        events = res.events ?? [];
      } catch (err) {
        // Transient network error — log and retry
        const axiosErr = err as AxiosError;
        if (axiosErr.isAxiosError && axiosErr.response?.status === 404) {
          // Job doesn't exist yet — wait a bit then retry
        } else {
          console.warn(`[AetherisClient] poll error for job ${jobId}:`, err);
        }
        await sleep(pollIntervalMs);
        continue;
      }

      const terminal = events.find((e) => TERMINAL_EVENT_TYPES.has(e.type));
      if (terminal) {
        return terminal;
      }

      await sleep(pollIntervalMs);
    }
    return null; // timed out
  }

  /**
   * Extracts the textual reply from a terminal event payload.
   * job_completed payloads typically carry { result: "...", output: "..." }.
   */
  static extractReplyText(event: JobEvent): string {
    if (event.type === "job_failed") {
      const payload = event.payload as Record<string, unknown> | null;
      const reason =
        (payload?.reason as string) ?? (payload?.error as string) ?? "";
      return reason
        ? `Sorry, the agent encountered an error: ${reason}`
        : "Sorry, the agent encountered an error while processing your request.";
    }

    if (event.type === "job_cancelled") {
      return "The request was cancelled.";
    }

    // job_completed
    const payload = event.payload as Record<string, unknown> | null;
    if (!payload) {
      return "Task completed successfully.";
    }
    // Try common field names used by Aetheris runners
    const text =
      (payload.result as string) ??
      (payload.output as string) ??
      (payload.answer as string) ??
      (payload.response as string) ??
      JSON.stringify(payload);
    return text;
  }
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
