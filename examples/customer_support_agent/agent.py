#!/usr/bin/env python3
"""Realistic external_http customer support agent for Aetheris examples.

The agent uses only Python's standard library so it can run in local demos
without package installation. It exposes the Aetheris external_http contract:

  POST /invoke -> {"answer": "...", "final": true, "metadata": {...}}

It also keeps an idempotency ledger on disk. Repeating the same
Idempotency-Key returns the same ticket instead of creating a duplicate case.
"""

from __future__ import annotations

import copy
import hashlib
import http.server
import json
import os
import re
import threading
import time
from dataclasses import dataclass
from pathlib import Path
from typing import Any


ORDERS: dict[str, dict[str, Any]] = {
    "A-1042": {
        "customer": "Mia Chen",
        "tier": "pro",
        "item": "espresso machine",
        "status": "delivered",
        "days_since_delivery": 4,
        "value_usd": 189,
        "risk": "low",
    },
    "B-7781": {
        "customer": "Noah Singh",
        "tier": "standard",
        "item": "noise-cancelling headphones",
        "status": "delayed",
        "days_late": 8,
        "value_usd": 42,
        "risk": "low",
    },
    "C-3019": {
        "customer": "Arden Robotics",
        "tier": "enterprise",
        "item": "industrial sensor pack",
        "status": "delivered",
        "days_since_delivery": 19,
        "value_usd": 1280,
        "risk": "high",
    },
}


@dataclass(frozen=True)
class SupportDecision:
    intent: str
    order_id: str | None
    action: str
    customer_reply: str
    internal_note: str
    priority: str
    confidence: float


class IdempotencyLedger:
    """Small JSON ledger used to demonstrate side-effect deduplication."""

    def __init__(self, path: Path):
        self.path = path
        self._lock = threading.Lock()
        self._data = self._load()

    def _load(self) -> dict[str, Any]:
        if not self.path.exists():
            return {"responses": {}, "tickets": []}
        try:
            return json.loads(self.path.read_text())
        except json.JSONDecodeError:
            return {"responses": {}, "tickets": []}

    def _save(self) -> None:
        self.path.parent.mkdir(parents=True, exist_ok=True)
        tmp = self.path.with_suffix(self.path.suffix + ".tmp")
        tmp.write_text(json.dumps(self._data, indent=2, sort_keys=True))
        tmp.replace(self.path)

    def cached_response(self, key: str) -> dict[str, Any] | None:
        with self._lock:
            response = self._data["responses"].get(key)
            if not response:
                return None
            cached = copy.deepcopy(response)
            cached.setdefault("metadata", {})["cached"] = True
            cached["metadata"]["ledger_ticket_count"] = len(self._data["tickets"])
            return cached

    def record_response(self, key: str, response: dict[str, Any]) -> dict[str, Any]:
        with self._lock:
            stored = copy.deepcopy(response)
            self._data["responses"][key] = stored
            self._data["tickets"].append(
                {
                    "ticket_id": stored["metadata"]["ticket_id"],
                    "idempotency_key": key,
                    "order_id": stored["metadata"].get("order_id"),
                    "action": stored["metadata"].get("action"),
                    "created_at": int(time.time()),
                }
            )
            stored["metadata"]["ledger_ticket_count"] = len(self._data["tickets"])
            self._data["responses"][key] = stored
            self._save()
            return copy.deepcopy(stored)

    def ticket_count(self) -> int:
        with self._lock:
            return len(self._data["tickets"])


class CustomerSupportAgent:
    """A deterministic support agent with realistic business policy steps."""

    def __init__(self, ledger: IdempotencyLedger):
        self.ledger = ledger

    def invoke(
        self,
        message: str,
        metadata: dict[str, Any] | None = None,
        headers: dict[str, str] | None = None,
    ) -> dict[str, Any]:
        metadata = metadata or {}
        headers = headers or {}
        idempotency_key = (
            headers.get("Idempotency-Key")
            or metadata.get("idempotency_key")
            or stable_key(message)
        )

        cached = self.ledger.cached_response(idempotency_key)
        if cached is not None:
            return cached

        decision = self._decide(message)
        ticket_id = make_ticket_id(idempotency_key)
        response = {
            "answer": format_answer(ticket_id, decision),
            "final": True,
            "metadata": {
                "agent": "customer_support_agent",
                "cached": False,
                "ticket_id": ticket_id,
                "order_id": decision.order_id,
                "intent": decision.intent,
                "action": decision.action,
                "priority": decision.priority,
                "confidence": decision.confidence,
                "job_id": metadata.get("job_id"),
                "idempotency_key": idempotency_key,
                "tools_used": [
                    "extract_order",
                    "lookup_order",
                    "evaluate_policy",
                    "record_case",
                    "draft_customer_reply",
                ],
            },
        }
        return self.ledger.record_response(idempotency_key, response)

    def _decide(self, message: str) -> SupportDecision:
        order_id = extract_order_id(message)
        order = ORDERS.get(order_id or "")
        intent = classify_intent(message)

        if order is None:
            return SupportDecision(
                intent=intent,
                order_id=order_id,
                action="request_order_details",
                customer_reply=(
                    "I can help, but I need a valid order id before changing an account "
                    "or creating a refund."
                ),
                internal_note="No matching order found in the local order index.",
                priority="normal",
                confidence=0.64,
            )

        if order["value_usd"] >= 500 or order["risk"] == "high":
            return SupportDecision(
                intent=intent,
                order_id=order_id,
                action="escalate_for_human_review",
                customer_reply=(
                    f"I found order {order_id} for {order['item']}. Because this is a "
                    "high-value account action, I created a priority review ticket "
                    "instead of issuing an automatic refund."
                ),
                internal_note="High-value policy requires human approval before refund.",
                priority="high",
                confidence=0.91,
            )

        if intent == "damaged_item" and order.get("days_since_delivery", 999) <= 30:
            return SupportDecision(
                intent=intent,
                order_id=order_id,
                action="approve_replacement_and_refund",
                customer_reply=(
                    f"I found order {order_id} for the {order['item']}. The damage report "
                    "is inside the 30-day support window, so I approved a replacement "
                    "and a courtesy refund."
                ),
                internal_note="Low-risk damaged-item policy auto-approved.",
                priority="normal",
                confidence=0.95,
            )

        if intent == "late_delivery" and order.get("status") == "delayed":
            return SupportDecision(
                intent=intent,
                order_id=order_id,
                action="offer_reship_or_credit",
                customer_reply=(
                    f"Order {order_id} is delayed. I created a support case offering "
                    "either a free reshipment or store credit."
                ),
                internal_note="Delayed shipment eligible for reship or credit.",
                priority="normal",
                confidence=0.9,
            )

        return SupportDecision(
            intent=intent,
            order_id=order_id,
            action="create_followup_ticket",
            customer_reply=(
                f"I found order {order_id}. I created a follow-up ticket with the "
                "support team so they can verify the next safe action."
            ),
            internal_note="No auto-resolution policy matched.",
            priority="normal",
            confidence=0.76,
        )


def extract_order_id(message: str) -> str | None:
    match = re.search(r"\b([A-Z]-\d{4})\b", message.upper())
    return match.group(1) if match else None


def classify_intent(message: str) -> str:
    text = message.lower()
    if any(word in text for word in ("damaged", "broken", "defective", "cracked")):
        return "damaged_item"
    if any(word in text for word in ("late", "delayed", "missing", "not arrived")):
        return "late_delivery"
    if "refund" in text:
        return "refund_request"
    return "general_support"


def stable_key(message: str) -> str:
    return hashlib.sha256(message.encode()).hexdigest()[:24]


def make_ticket_id(idempotency_key: str) -> str:
    digest = hashlib.sha1(idempotency_key.encode()).hexdigest()[:8].upper()
    return f"CS-{digest}"


def format_answer(ticket_id: str, decision: SupportDecision) -> str:
    return (
        f"Ticket {ticket_id}: {decision.customer_reply}\n"
        f"Action: {decision.action}. Priority: {decision.priority}. "
        f"Confidence: {decision.confidence:.2f}.\n"
        f"Internal note: {decision.internal_note}"
    )


def make_server(host: str, port: int, ledger_path: Path) -> http.server.ThreadingHTTPServer:
    support_agent = CustomerSupportAgent(IdempotencyLedger(ledger_path))

    class Handler(http.server.BaseHTTPRequestHandler):
        def log_message(self, format: str, *args: Any) -> None:
            return

        def _json(self, status: int, payload: dict[str, Any]) -> None:
            data = json.dumps(payload).encode()
            self.send_response(status)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(data)))
            self.end_headers()
            self.wfile.write(data)

        def do_GET(self) -> None:
            if self.path == "/health":
                self._json(
                    200,
                    {
                        "status": "healthy",
                        "agent": "customer_support_agent",
                        "ledger_ticket_count": support_agent.ledger.ticket_count(),
                    },
                )
                return
            self._json(
                200,
                {
                    "name": "customer_support_agent",
                    "invoke": "/invoke",
                    "health": "/health",
                },
            )

        def do_POST(self) -> None:
            if self.path != "/invoke":
                self._json(404, {"error": "not found"})
                return
            length = int(self.headers.get("Content-Length", "0"))
            try:
                body = json.loads(self.rfile.read(length) or b"{}")
                response = support_agent.invoke(
                    body.get("message", ""),
                    body.get("metadata") or {},
                    {
                        "Idempotency-Key": self.headers.get("Idempotency-Key", ""),
                        "X-Aetheris-Job-ID": self.headers.get("X-Aetheris-Job-ID", ""),
                        "X-Aetheris-Agent-ID": self.headers.get("X-Aetheris-Agent-ID", ""),
                    },
                )
            except Exception as exc:
                self._json(500, {"error": str(exc)})
                return
            self._json(200, response)

    return http.server.ThreadingHTTPServer((host, port), Handler)


def main() -> None:
    port = int(os.environ.get("AGENT_PORT", "19002"))
    ledger_path = Path(os.environ.get("LEDGER_PATH", "artifacts/agent-ledger.json"))
    server = make_server("127.0.0.1", port, ledger_path)
    print(f"customer_support_agent listening on http://127.0.0.1:{port}", flush=True)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        pass
    finally:
        server.server_close()


if __name__ == "__main__":
    main()
