import tempfile
import unittest
from pathlib import Path

from agent import CustomerSupportAgent, IdempotencyLedger


class CustomerSupportAgentTest(unittest.TestCase):
    def make_agent(self) -> CustomerSupportAgent:
        tmp = tempfile.TemporaryDirectory()
        self.addCleanup(tmp.cleanup)
        return CustomerSupportAgent(IdempotencyLedger(Path(tmp.name) / "ledger.json"))

    def test_damaged_low_risk_order_is_auto_approved(self) -> None:
        agent = self.make_agent()
        response = agent.invoke(
            "Order A-1042 arrived damaged. Please refund and replace it.",
            headers={"Idempotency-Key": "unit-damaged-001"},
        )

        self.assertTrue(response["final"])
        self.assertEqual(response["metadata"]["intent"], "damaged_item")
        self.assertEqual(response["metadata"]["order_id"], "A-1042")
        self.assertEqual(response["metadata"]["action"], "approve_replacement_and_refund")
        self.assertIn("Ticket CS-", response["answer"])

    def test_high_value_order_requires_human_review(self) -> None:
        agent = self.make_agent()
        response = agent.invoke(
            "Order C-3019 is defective. Approve a refund now.",
            headers={"Idempotency-Key": "unit-high-value-001"},
        )

        self.assertEqual(response["metadata"]["action"], "escalate_for_human_review")
        self.assertEqual(response["metadata"]["priority"], "high")

    def test_idempotency_reuses_ticket_without_duplicate_case(self) -> None:
        agent = self.make_agent()
        first = agent.invoke(
            "Order A-1042 arrived damaged. Please refund and replace it.",
            headers={"Idempotency-Key": "unit-repeat-001"},
        )
        second = agent.invoke(
            "Order A-1042 arrived damaged. Please refund and replace it.",
            headers={"Idempotency-Key": "unit-repeat-001"},
        )

        self.assertEqual(first["metadata"]["ticket_id"], second["metadata"]["ticket_id"])
        self.assertFalse(first["metadata"]["cached"])
        self.assertTrue(second["metadata"]["cached"])
        self.assertEqual(second["metadata"]["ledger_ticket_count"], 1)


if __name__ == "__main__":
    unittest.main()
