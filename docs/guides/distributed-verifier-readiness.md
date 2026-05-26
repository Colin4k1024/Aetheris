# Distributed Verifier Readiness

The distributed verifier is a prototype reserve for multi-organization evidence validation. It should not be promoted merely because root-hash comparison code exists.

## Promotion Gate

Promotion requires evidence for all of the following:

- single-node saturation or verifier bottleneck evidence
- lease failure-mode tests
- recovery failure-mode tests
- multi-org root hash release drill

The code-level gate is `distributed.AssessPromotionReadiness`.

## Current Decision

Current decision: keep `pkg/distributed` as `prototype`.

Reason: the release drill can prove accepted and divergent root-hash behavior, but the repository does not yet contain operational evidence that single-node verification is saturated or that distributed verification is needed in production.

## Root Hash Drill

The release drill covers:

- accepted case: multiple organizations report the same root hash
- divergent case: organizations report different root hashes
- missing hash / empty stream / pull failure cases
- readiness assessment remains conservative without saturation, lease, and recovery evidence

## Non-Goals

- Do not add distributed runtime complexity before the single-node runtime bottleneck is proven.
- Do not use quorum semantics to hide root-hash divergence.
- Do not promote this package to stable API without storage, recovery, and operational ownership decisions.
