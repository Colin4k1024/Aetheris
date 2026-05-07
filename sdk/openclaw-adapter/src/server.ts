/**
 * server.ts — Express HTTP server
 *
 * Exposes two endpoints:
 *
 *   POST /webhook/message
 *     Accepts an OpenClaw inbound message (see OpenClawInboundMessage type),
 *     routes it to Aetheris, polls for job completion, and returns the reply.
 *
 *   GET /health
 *     Simple liveness probe.
 *
 * Design note:
 *   The handler is synchronous from the caller's perspective — it waits for
 *   job completion before responding.  This keeps the integration simple for
 *   platforms that support long-lived webhooks (e.g. Telegram getUpdates).
 *
 *   For platforms that require an immediate 200 ACK (e.g. WhatsApp Cloud API),
 *   set ASYNC_REPLY=true in env and implement the callback path in handler.ts.
 */

import express, { Request, Response, NextFunction } from "express";
import { MessageHandler } from "./handler";
import { AdapterConfig, OpenClawInboundMessage } from "./types";

export function createServer(config: AdapterConfig): express.Express {
  const app = express();
  app.use(express.json({ limit: "1mb" }));

  const handler = new MessageHandler(config);

  // ------------------------------------------------------------------
  // Health probe
  // ------------------------------------------------------------------
  app.get("/health", (_req: Request, res: Response) => {
    res.json({ status: "ok", timestamp: new Date().toISOString() });
  });

  // ------------------------------------------------------------------
  // Main webhook: POST /webhook/message
  // ------------------------------------------------------------------
  app.post(
    "/webhook/message",
    async (req: Request, res: Response, next: NextFunction) => {
      try {
        const body = req.body as Partial<OpenClawInboundMessage>;

        // Validate required fields
        if (!body.platform || !body.chatId || !body.text) {
          res.status(400).json({
            error: "Missing required fields: platform, chatId, text",
          });
          return;
        }

        const msg: OpenClawInboundMessage = {
          platform: body.platform,
          chatId: body.chatId,
          senderName: body.senderName,
          text: body.text,
          timestamp: body.timestamp ?? new Date().toISOString(),
          metadata: body.metadata,
        };

        const reply = await handler.handle(msg);
        res.json(reply);
      } catch (err) {
        next(err);
      }
    }
  );

  // ------------------------------------------------------------------
  // Error handler
  // ------------------------------------------------------------------
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  app.use((err: Error, _req: Request, res: Response, _next: NextFunction) => {
    console.error("[server] unhandled error:", err);
    res.status(500).json({ error: "Internal server error" });
  });

  return app;
}

/**
 * Start the server.  Called by index.ts; exported for testing.
 */
export function startServer(config: AdapterConfig): void {
  const port = config.serverPort ?? 3100;
  const app = createServer(config);
  app.listen(port, () => {
    console.info(`[server] OpenClaw→Aetheris adapter listening on :${port}`);
    console.info(`[server] Aetheris base URL : ${config.aetherisBaseUrl}`);
    console.info(`[server] Default agent ID  : ${config.agentId}`);
  });
}
