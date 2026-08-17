"""
Aetheris Durability SDK — standalone crash recovery and idempotency for AI agents.

Zero framework dependency. Works with any Python agent.

Quick start:

    from aetheris_durability import Runner, MemoryStore

    store = MemoryStore()
    runner = Runner(store)

    job = runner.start("process-order", {"order_id": "123"})
    result = runner.execute(job.id, [
        Step("validate", validate_order),
        Step("charge", charge_payment, max_retries=3),
        Step("ship", create_shipment),
    ])
    # Process crashed? Call execute() again — resumes from last checkpoint.
"""

from .types import Job, JobState, Step, Event, EventType, Checkpoint
from .store import Store, MemoryStore
from .runner import Runner
from .idempotent import IdempotentTool

__all__ = [
    "Job",
    "JobState",
    "Step",
    "Event",
    "EventType",
    "Checkpoint",
    "Store",
    "MemoryStore",
    "Runner",
    "IdempotentTool",
]

__version__ = "0.1.0"
