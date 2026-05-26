# ADR-0001: Runtime-First Prototype Promotion Policy

Date: 2026-05-25

Status: Accepted

## Context

Aetheris contains production runtime code, integrated forensics/compliance surfaces, and several 3.0 candidate packages. Without a clear promotion policy, documentation can imply a broader product surface than the runtime can prove.

The architecture review identified the main risk: users may read technical reserve packages as production guarantees.

## Decision

Aetheris keeps **reliable execution runtime** as the 2.x north star.

Prototype capabilities are promoted only by vertical slice:

1. A runtime path exists.
2. The API or CLI contract is documented.
3. Config and storage ownership are documented.
4. Unit, integration, and failure-path tests exist.
5. Operational runbooks and release gates exist.

The first recommended 3.0 slice is signed evidence export because it strengthens the existing runtime/evidence story without creating a separate product lane.

## Consequences

- `pkg/signature`, `pkg/distributed`, `pkg/ai_forensics`, `pkg/monitoring`, and `pkg/compliance` remain prototype unless a slice-specific artifact proves otherwise.
- Forensics and compliance APIs stay gated by `api.forensics.experimental` until their contracts and operations evidence are ready.
- Documentation must distinguish stable runtime exports from experimental query/report surfaces.

## Alternatives Considered

### Promote all enterprise packages together

Rejected. It would create a large release surface without matching tests, storage contracts, and operational drills.

### Freeze all 3.0 work

Rejected. A narrow signed-evidence slice directly supports the runtime evidence plane and can be promoted incrementally.

### Treat documentation as sufficient promotion evidence

Rejected. Promotion changes user expectations and therefore requires code, tests, and release gates.
