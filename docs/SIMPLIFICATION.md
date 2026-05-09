# Simplification Plan

This plan keeps the project focused on the fastest user path:

1. Start Aetheris locally in embedded mode.
2. Register an existing agent through `external_http`.
3. Submit a job.
4. Inspect job status, events, and trace output.

## Current Decision

Primary audience: users who want to run or wrap an agent quickly.

Primary path:

```text
embedded API runtime -> external_http agent -> /api/agents/{id}/message -> /api/jobs/{job_id}/trace
```

The `/api/agents/{id}/message` endpoint is still the practical facade for agent submission. It returns runtime mapping metadata and writes the canonical job/run records behind the scenes.

Agent definitions must be present in the active runtime config. `configs/agents.yaml` is useful as a reference, but it is not the current quickstart entrypoint.

Second-round decision: documentation entrypoints now point to one beginner path. `docs/README.md` is a navigation page, while `guides/get-started.md`, `guides/getting-started-agents.md`, and `docs/README_zh.md` are retained but explicitly marked as complete testing, advanced authoring, or historical/compatibility material.

Third-round decision: examples are now grouped by maturity and intent, and configuration docs now state that agent definitions must be added to the active runtime config. Embedded config files include comments that point users toward the correct place without changing runtime behavior.

## Keep Prominent

- [README.md](../README.md)
- [docs/guides/quickstart.md](guides/quickstart.md)
- [docs/adapters/external-http-agent.md](adapters/external-http-agent.md)
- [docs/reference/api.md](reference/api.md)
- [configs/agents.yaml](../configs/agents.yaml) as a reference for available agent fields
- Embedded configs: [configs/api.embedded.yaml](../configs/api.embedded.yaml), [configs/worker.embedded.yaml](../configs/worker.embedded.yaml)

## De-Emphasize

These areas may remain in the repo, but should not be on the main onboarding path:

- Legacy `/api/query` examples.
- Legacy `/api/agent/*` examples.
- Long marketing narratives in the root README.
- Historical roadmaps and release checklists.
- Prototype vector adapters such as Milvus and Pinecone placeholders.
- Deep design notes that are useful for maintainers but not for first-run users.

## Candidate Cleanup Backlog

| Area | Proposed action | Why |
| ---- | --------------- | --- |
| Quickstart docs | Keep one canonical quickstart and redirect older guides to it | Avoid three competing first-run paths |
| README | Keep it short and action-oriented | Users should know what to run in under a minute |
| Docs archive | Move historical roadmap and release snapshots under `docs/archive/` | Preserve context without making it look current |
| API docs | Split stable path from compatibility path | Makes `/api/runs`, `/api/jobs`, and agent facade boundaries clear |
| Examples | Continue adding maturity labels to individual example READMEs | Reduces guesswork inside example folders |
| Config UX | Consider loading `configs/agents.yaml` explicitly or renaming it as reference-only | Removes the active-config vs reference-file trap |
| Prototype adapters | Gate or mark placeholder implementations clearly | Prevents accidental production use |
| Legacy facade | Keep compatibility docs, but remove from main narrative | Reduces conceptual load |

## Do Not Remove Yet

- Runtime code paths that are covered by tests.
- Compatibility API endpoints used by existing examples or adapters.
- Historical docs before links are audited.
- Generated protobuf files.

## Verification Checklist

Before each simplification PR:

```bash
make fmt-check
make test
```

If Go is not available in the local shell, record that in the PR and run at least:

```bash
git diff --check
```
