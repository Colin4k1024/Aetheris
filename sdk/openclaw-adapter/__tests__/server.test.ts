/**
 * server.test.ts — Integration tests for the Express webhook endpoint
 *
 * Uses supertest to exercise the actual Express app without starting a real
 * TCP server.  MessageHandler is mocked so no Aetheris calls are made.
 */

import request from "supertest";
import { createServer } from "../src/server";
import { MessageHandler } from "../src/handler";
import { AdapterConfig } from "../src/types";

jest.mock("../src/handler");
const MockedMessageHandler = MessageHandler as jest.MockedClass<typeof MessageHandler>;

const config: AdapterConfig = {
  aetherisBaseUrl: "http://localhost:8080",
  aetherisToken: "token",
  agentId: "react",
};

beforeEach(() => {
  MockedMessageHandler.mockClear();
});

describe("GET /health", () => {
  it("returns 200 with status ok", async () => {
    const app = createServer(config);
    const res = await request(app).get("/health");
    expect(res.status).toBe(200);
    expect(res.body.status).toBe("ok");
  });
});

describe("POST /webhook/message", () => {
  const validBody = {
    platform: "telegram",
    chatId: "chat-999",
    text: "Hello",
    timestamp: "2024-01-01T00:00:00.000Z",
  };

  it("returns 400 when required fields are missing", async () => {
    const app = createServer(config);
    const res = await request(app).post("/webhook/message").send({ platform: "telegram" });
    expect(res.status).toBe(400);
    expect(res.body.error).toMatch(/required fields/i);
  });

  it("calls MessageHandler.handle and returns reply on success", async () => {
    const mockHandle = jest.fn().mockResolvedValue({
      platform: "telegram",
      chatId: "chat-999",
      text: "4",
      final: true,
    });
    MockedMessageHandler.mockImplementation(() => ({
      handle: mockHandle,
    }) as unknown as MessageHandler);

    const app = createServer(config);
    const res = await request(app).post("/webhook/message").send(validBody);

    expect(res.status).toBe(200);
    expect(res.body.text).toBe("4");
    expect(res.body.final).toBe(true);
    expect(mockHandle).toHaveBeenCalledTimes(1);
  });

  it("returns 500 when MessageHandler throws", async () => {
    MockedMessageHandler.mockImplementation(() => ({
      handle: jest.fn().mockRejectedValue(new Error("unexpected")),
    }) as unknown as MessageHandler);

    const app = createServer(config);
    const res = await request(app).post("/webhook/message").send(validBody);
    expect(res.status).toBe(500);
  });

  it("fills in timestamp when caller omits it", async () => {
    const mockHandle = jest.fn().mockResolvedValue({
      platform: "telegram",
      chatId: "chat-999",
      text: "ok",
      final: true,
    });
    MockedMessageHandler.mockImplementation(() => ({
      handle: mockHandle,
    }) as unknown as MessageHandler);

    const app = createServer(config);
    const bodyWithoutTs = { platform: "telegram", chatId: "chat-999", text: "hi" };
    await request(app).post("/webhook/message").send(bodyWithoutTs);

    const called = mockHandle.mock.calls[0][0];
    expect(called.timestamp).toBeTruthy();
  });
});
