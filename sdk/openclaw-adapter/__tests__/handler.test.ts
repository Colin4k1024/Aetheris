/**
 * handler.test.ts — Unit tests for MessageHandler
 */

import { MessageHandler } from "../src/handler";
import { AetherisClient } from "../src/aetheris-client";
import { AdapterConfig, OpenClawInboundMessage } from "../src/types";

// Mock AetherisClient so MessageHandler doesn't make real HTTP calls.
// Use jest.requireActual to preserve the static extractReplyText helper.
jest.mock("../src/aetheris-client", () => {
  const actual = jest.requireActual("../src/aetheris-client") as typeof import("../src/aetheris-client");
  return {
    AetherisClient: Object.assign(jest.fn(), {
      extractReplyText: actual.AetherisClient.extractReplyText,
    }),
  };
});
const MockedAetherisClient = AetherisClient as jest.MockedClass<typeof AetherisClient>;

const baseConfig: AdapterConfig = {
  aetherisBaseUrl: "http://localhost:8080",
  aetherisToken: "test-token",
  agentId: "react",
  jobTimeoutMs: 5_000,
  pollIntervalMs: 100,
};

const baseMessage: OpenClawInboundMessage = {
  platform: "telegram",
  chatId: "chat-123",
  senderName: "Alice",
  text: "What is 2+2?",
  timestamp: "2024-01-01T00:00:00.000Z",
};

beforeEach(() => {
  MockedAetherisClient.mockClear();
});

describe("MessageHandler.handle", () => {
  it("returns reply from job_completed event", async () => {
    const mockInstance = {
      sendMessage: jest.fn().mockResolvedValue({
        status: "accepted",
        agent_id: "react",
        job_id: "job-ok",
        runtime_submission: { legacy_facade: true, canonical_api: "/api/runs", run_status: "created" },
      }),
      waitForTerminalEvent: jest.fn().mockResolvedValue({
        id: 2,
        job_id: "job-ok",
        type: "job_completed",
        payload: { result: "4" },
        created_at: "",
      }),
    };
    MockedAetherisClient.mockImplementation(() => mockInstance as unknown as AetherisClient);

    const handler = new MessageHandler(baseConfig);
    const reply = await handler.handle(baseMessage);

    expect(reply.platform).toBe("telegram");
    expect(reply.chatId).toBe("chat-123");
    expect(reply.text).toBe("4");
    expect(reply.final).toBe(true);
  });

  it("returns error reply when sendMessage throws", async () => {
    const mockInstance = {
      sendMessage: jest.fn().mockRejectedValue(new Error("connection refused")),
      waitForTerminalEvent: jest.fn(),
    };
    MockedAetherisClient.mockImplementation(() => mockInstance as unknown as AetherisClient);

    const handler = new MessageHandler(baseConfig);
    const reply = await handler.handle(baseMessage);

    expect(reply.final).toBe(true);
    expect(reply.text).toMatch(/could not reach/i);
    expect(mockInstance.waitForTerminalEvent).not.toHaveBeenCalled();
  });

  it("returns timeout reply when waitForTerminalEvent returns null", async () => {
    const mockInstance = {
      sendMessage: jest.fn().mockResolvedValue({
        status: "accepted",
        agent_id: "react",
        job_id: "job-slow",
        runtime_submission: { legacy_facade: true, canonical_api: "/api/runs", run_status: "created" },
      }),
      waitForTerminalEvent: jest.fn().mockResolvedValue(null),
    };
    MockedAetherisClient.mockImplementation(() => mockInstance as unknown as AetherisClient);

    const handler = new MessageHandler(baseConfig);
    const reply = await handler.handle(baseMessage);

    expect(reply.final).toBe(true);
    expect(reply.text).toMatch(/taking longer/i);
  });

  it("returns error message for job_failed event", async () => {
    const mockInstance = {
      sendMessage: jest.fn().mockResolvedValue({
        status: "accepted",
        agent_id: "react",
        job_id: "job-fail",
        runtime_submission: { legacy_facade: true, canonical_api: "/api/runs", run_status: "created" },
      }),
      waitForTerminalEvent: jest.fn().mockResolvedValue({
        id: 1,
        job_id: "job-fail",
        type: "job_failed",
        payload: { reason: "model overloaded" },
        created_at: "",
      }),
    };
    MockedAetherisClient.mockImplementation(() => mockInstance as unknown as AetherisClient);

    const handler = new MessageHandler(baseConfig);
    const reply = await handler.handle(baseMessage);

    expect(reply.final).toBe(true);
    expect(reply.text).toContain("model overloaded");
  });

  it("resolves agentId via function when config.agentId is a function", async () => {
    const mockInstance = {
      sendMessage: jest.fn().mockResolvedValue({
        status: "accepted",
        agent_id: "manus",
        job_id: "job-dyn",
        runtime_submission: { legacy_facade: true, canonical_api: "/api/runs", run_status: "created" },
      }),
      waitForTerminalEvent: jest.fn().mockResolvedValue({
        id: 1,
        job_id: "job-dyn",
        type: "job_completed",
        payload: { result: "dynamic agent reply" },
        created_at: "",
      }),
    };
    MockedAetherisClient.mockImplementation(() => mockInstance as unknown as AetherisClient);

    const dynamicConfig: AdapterConfig = {
      ...baseConfig,
      // Route WhatsApp messages to "manus", everything else to "react"
      agentId: (msg) => (msg.platform === "whatsapp" ? "manus" : "react"),
    };
    const handler = new MessageHandler(dynamicConfig);
    const reply = await handler.handle({ ...baseMessage, platform: "whatsapp" });

    expect(reply.text).toBe("dynamic agent reply");
    expect(mockInstance.sendMessage).toHaveBeenCalledWith(
      "manus",
      baseMessage.text,
      expect.any(String) // idempotency key
    );
  });
});
