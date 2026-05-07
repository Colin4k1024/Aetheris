/**
 * handler.ts — Core message routing logic
 *
 * Receives an inbound OpenClaw message, submits it to Aetheris, polls for
 * completion, and returns an outbound reply.
 *
 * This module is deliberately framework-agnostic: the Express webhook wires
 * it up, but it can be called from any transport layer (Lambda, Cloud Run, etc.)
 */

import { AetherisClient } from "./aetheris-client";
import {
  AdapterConfig,
  OpenClawInboundMessage,
  OpenClawOutboundReply,
} from "./types";
import { buildIdempotencyKey } from "./utils";

export class MessageHandler {
  private readonly client: AetherisClient;
  private readonly config: AdapterConfig;

  constructor(config: AdapterConfig) {
    this.config = config;
    this.client = new AetherisClient(
      config.aetherisBaseUrl,
      config.aetherisToken
    );
  }

  /**
   * Handle one inbound message end-to-end:
   * 1. Resolve the target agentId
   * 2. Submit message → Aetheris (with idempotency key)
   * 3. Poll until terminal event or timeout
   * 4. Return the reply text
   */
  async handle(msg: OpenClawInboundMessage): Promise<OpenClawOutboundReply> {
    const agentId =
      typeof this.config.agentId === "function"
        ? this.config.agentId(msg)
        : this.config.agentId;

    const timeoutMs = this.config.jobTimeoutMs ?? 120_000;
    const pollIntervalMs = this.config.pollIntervalMs ?? 1_500;

    // Idempotency key: scoped to platform + chatId + message text + timestamp
    // This prevents duplicate jobs if OpenClaw retries the webhook delivery.
    const idempotencyKey = buildIdempotencyKey(msg);

    console.info(
      `[MessageHandler] → agentId=${agentId} platform=${msg.platform} chatId=${msg.chatId} key=${idempotencyKey}`
    );

    // 1. Submit message to Aetheris
    let jobId: string;
    try {
      const accepted = await this.client.sendMessage(
        agentId,
        msg.text,
        idempotencyKey
      );
      jobId = accepted.job_id;
      console.info(`[MessageHandler] Job accepted: job_id=${jobId}`);
    } catch (err) {
      console.error("[MessageHandler] sendMessage failed:", err);
      return this.errorReply(
        msg,
        "Sorry, I could not reach the AI agent right now. Please try again."
      );
    }

    // 2. Poll for terminal event
    const terminal = await this.client.waitForTerminalEvent(
      jobId,
      timeoutMs,
      pollIntervalMs
    );

    if (!terminal) {
      console.warn(
        `[MessageHandler] Job ${jobId} timed out after ${timeoutMs}ms`
      );
      return this.errorReply(
        msg,
        "The agent is taking longer than expected. Your request is still being processed — you'll get a reply soon."
      );
    }

    // 3. Build reply
    const replyText = AetherisClient.extractReplyText(terminal);
    console.info(
      `[MessageHandler] Job ${jobId} terminal event="${terminal.type}", reply length=${replyText.length}`
    );

    return {
      platform: msg.platform,
      chatId: msg.chatId,
      text: replyText,
      final: true,
    };
  }

  private errorReply(
    msg: OpenClawInboundMessage,
    text: string
  ): OpenClawOutboundReply {
    return { platform: msg.platform, chatId: msg.chatId, text, final: true };
  }
}
