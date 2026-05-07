/**
 * aetheris-client.test.ts — Unit tests for AetherisClient
 */

import axios from "axios";
import { AetherisClient } from "../src/aetheris-client";
import { JobEvent } from "../src/types";

// Mock axios at module level so we don't make real HTTP calls
jest.mock("axios");
const mockedAxios = axios as jest.Mocked<typeof axios>;

// axios.create returns an axios-like instance; we mock it here
const mockGet = jest.fn();
const mockPost = jest.fn();
const mockHttpInstance = {
  get: mockGet,
  post: mockPost,
};

beforeAll(() => {
  (mockedAxios.create as jest.Mock).mockReturnValue(mockHttpInstance);
});

beforeEach(() => {
  mockGet.mockReset();
  mockPost.mockReset();
});

// ---------------------------------------------------------------------------
// sendMessage
// ---------------------------------------------------------------------------
describe("AetherisClient.sendMessage", () => {
  it("POSTs to /api/agents/:id/message and returns accepted response", async () => {
    const client = new AetherisClient("http://localhost:8080", "test-token");
    const fakeResponse = {
      data: {
        status: "accepted",
        agent_id: "react",
        job_id: "job-abc-123",
        runtime_submission: {
          legacy_facade: true,
          canonical_api: "/api/runs",
          run_status: "created",
        },
      },
    };
    mockPost.mockResolvedValueOnce(fakeResponse);

    const result = await client.sendMessage("react", "Hello world");

    expect(mockPost).toHaveBeenCalledWith(
      "/api/agents/react/message",
      { message: "Hello world" },
      { headers: {} }
    );
    expect(result.job_id).toBe("job-abc-123");
    expect(result.status).toBe("accepted");
  });

  it("passes Idempotency-Key header when provided", async () => {
    const client = new AetherisClient("http://localhost:8080", "test-token");
    mockPost.mockResolvedValueOnce({
      data: {
        status: "accepted",
        agent_id: "react",
        job_id: "job-idem-456",
        runtime_submission: { legacy_facade: true, canonical_api: "/api/runs", run_status: "created" },
      },
    });

    await client.sendMessage("react", "Idempotent message", "my-key-123");

    expect(mockPost).toHaveBeenCalledWith(
      "/api/agents/react/message",
      { message: "Idempotent message" },
      { headers: { "Idempotency-Key": "my-key-123" } }
    );
  });
});

// ---------------------------------------------------------------------------
// getJobEvents
// ---------------------------------------------------------------------------
describe("AetherisClient.getJobEvents", () => {
  it("GETs /api/jobs/:id/events and returns events array", async () => {
    const client = new AetherisClient("http://localhost:8080", "test-token");
    const fakeEvents = {
      data: {
        job_id: "job-abc-123",
        events: [
          { id: 1, job_id: "job-abc-123", type: "job_created", payload: null, created_at: "2024-01-01T00:00:00Z" },
          { id: 2, job_id: "job-abc-123", type: "job_completed", payload: { result: "42" }, created_at: "2024-01-01T00:00:01Z" },
        ],
      },
    };
    mockGet.mockResolvedValueOnce(fakeEvents);

    const result = await client.getJobEvents("job-abc-123");
    expect(result.job_id).toBe("job-abc-123");
    expect(result.events).toHaveLength(2);
    expect(result.events[1].type).toBe("job_completed");
  });
});

// ---------------------------------------------------------------------------
// waitForTerminalEvent
// ---------------------------------------------------------------------------
describe("AetherisClient.waitForTerminalEvent", () => {
  it("returns terminal event when job_completed is present", async () => {
    const client = new AetherisClient("http://localhost:8080", "test-token");
    mockGet.mockResolvedValueOnce({
      data: {
        job_id: "job-1",
        events: [
          { id: 1, job_id: "job-1", type: "job_created", payload: null, created_at: "" },
          { id: 2, job_id: "job-1", type: "job_completed", payload: { result: "done" }, created_at: "" },
        ],
      },
    });

    const result = await client.waitForTerminalEvent("job-1", 5_000, 100);
    expect(result).not.toBeNull();
    expect(result!.type).toBe("job_completed");
  });

  it("returns terminal event for job_failed", async () => {
    const client = new AetherisClient("http://localhost:8080", "test-token");
    mockGet.mockResolvedValueOnce({
      data: {
        job_id: "job-2",
        events: [
          { id: 1, job_id: "job-2", type: "job_failed", payload: { reason: "LLM error" }, created_at: "" },
        ],
      },
    });

    const result = await client.waitForTerminalEvent("job-2", 5_000, 100);
    expect(result).not.toBeNull();
    expect(result!.type).toBe("job_failed");
  });

  it("returns null when timeout expires before terminal event", async () => {
    const client = new AetherisClient("http://localhost:8080", "test-token");
    // Always return non-terminal events
    mockGet.mockResolvedValue({
      data: {
        job_id: "job-3",
        events: [{ id: 1, job_id: "job-3", type: "job_running", payload: null, created_at: "" }],
      },
    });

    const result = await client.waitForTerminalEvent("job-3", 200, 50);
    expect(result).toBeNull();
  });

  it("retries after transient 404 from Aetheris", async () => {
    const client = new AetherisClient("http://localhost:8080", "test-token");
    const notFound = Object.assign(new Error("not found"), {
      isAxiosError: true,
      response: { status: 404 },
    });
    mockGet
      .mockRejectedValueOnce(notFound) // first call: 404
      .mockResolvedValueOnce({          // second call: success
        data: {
          job_id: "job-4",
          events: [
            { id: 1, job_id: "job-4", type: "job_completed", payload: { result: "ok" }, created_at: "" },
          ],
        },
      });

    const result = await client.waitForTerminalEvent("job-4", 5_000, 50);
    expect(result).not.toBeNull();
    expect(result!.type).toBe("job_completed");
    expect(mockGet).toHaveBeenCalledTimes(2);
  });
});

// ---------------------------------------------------------------------------
// extractReplyText
// ---------------------------------------------------------------------------
describe("AetherisClient.extractReplyText", () => {
  const makeEvent = (type: string, payload: Record<string, unknown> | null): JobEvent => ({
    id: 1,
    job_id: "j",
    type,
    payload,
    created_at: "",
  });

  it("extracts result field from job_completed payload", () => {
    const e = makeEvent("job_completed", { result: "42 is the answer" });
    expect(AetherisClient.extractReplyText(e)).toBe("42 is the answer");
  });

  it("falls back to output field", () => {
    const e = makeEvent("job_completed", { output: "output text" });
    expect(AetherisClient.extractReplyText(e)).toBe("output text");
  });

  it("returns generic message for null payload on job_completed", () => {
    const e = makeEvent("job_completed", null);
    expect(AetherisClient.extractReplyText(e)).toBe("Task completed successfully.");
  });

  it("returns error message for job_failed with reason", () => {
    const e = makeEvent("job_failed", { reason: "timeout" });
    expect(AetherisClient.extractReplyText(e)).toContain("timeout");
  });

  it("returns cancelled message for job_cancelled", () => {
    const e = makeEvent("job_cancelled", null);
    expect(AetherisClient.extractReplyText(e)).toContain("cancelled");
  });
});
