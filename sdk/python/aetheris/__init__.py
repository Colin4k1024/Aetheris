"""
Aetheris Python SDK
~~~~~~~~~~~~~~~~~~~~

Minimal client for the Aetheris durable execution runtime.

Usage::

    from aetheris import AetherisClient

    client = AetherisClient("http://localhost:8080")

    # Send a message to an agent (creates a durable job)
    job = client.run("my-agent", "Summarise the Q3 report")

    # Poll until the job completes (or raises on failure)
    result = job.wait()
    print(result.output)

"""

from .client import AetherisClient, Job, JobStatus, AetherisError

__all__ = ["AetherisClient", "Job", "JobStatus", "AetherisError"]
__version__ = "0.1.0"
