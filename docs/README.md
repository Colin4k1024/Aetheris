# Documentation

This directory contains user guides, references, design notes, and historical planning material for Aetheris.

For new users, follow one path first:

```text
README.md -> guides/quickstart.md -> adapters/external-http-agent.md -> jobs/events/trace
```

## Start Here

| Goal | Document |
| ---- | -------- |
| Run Aetheris locally and submit one job | [guides/quickstart.md](guides/quickstart.md) |
| Wrap an existing HTTP agent | [adapters/external-http-agent.md](adapters/external-http-agent.md) |
| Inspect job status, events, and traces | [reference/api.md](reference/api.md) |
| Understand runtime guarantees | [guides/runtime-guarantees.md](guides/runtime-guarantees.md) |
| Deploy beyond local embedded mode | [guides/deployment.md](guides/deployment.md) |

## Common Next Steps

| Goal | Document |
| ---- | -------- |
| Use the CLI | [guides/cli.md](guides/cli.md) |
| Browse examples | [guides/examples.md](guides/examples.md) |
| Configure API, worker, model, and storage | [reference/config.md](reference/config.md) |
| Troubleshoot local setup | [guides/troubleshooting.md](guides/troubleshooting.md) |
| Add observability | [guides/observability.md](guides/observability.md), [guides/tracing.md](guides/tracing.md) |
| Run with Docker Compose | [guides/deployment.md](guides/deployment.md) |

## Reference

| Area | Document |
| ---- | -------- |
| API surface | [reference/api.md](reference/api.md) |
| API stability and compatibility | [reference/api-contract.md](reference/api-contract.md) |
| Configuration | [reference/config.md](reference/config.md) |
| Current status | [STATUS.md](STATUS.md) |
| Version history | [../CHANGELOG.md](../CHANGELOG.md) |

## Advanced / Historical

These documents remain useful, but they are not the recommended first-run path.

| Area | Document |
| ---- | -------- |
| Full feature and E2E testing | [guides/get-started.md](guides/get-started.md), [guides/test-e2e.md](guides/test-e2e.md) |
| Eino and custom tool authoring | [guides/getting-started-agents.md](guides/getting-started-agents.md) |
| Custom adapters and migration paths | [adapters/README.md](adapters/README.md), [adapters/custom-agent.md](adapters/custom-agent.md), [adapters/go-frameworks.md](adapters/go-frameworks.md) |
| Concepts and architecture | [concepts/](concepts/), [../design/](../design/) |
| Security and enterprise features | [guides/security.md](guides/security.md), [guides/security-advanced.md](guides/security-advanced.md), [guides/enterprise-integrations.md](guides/enterprise-integrations.md) |
| Release and roadmap history | [releases/](releases/), [roadmaps/](roadmaps/), [milestones/](milestones/) |
| Blog and promotion material | [blog/](blog/), [promotion/](promotion/) |

## Notes For Maintainers

- Keep [guides/quickstart.md](guides/quickstart.md) as the only beginner quickstart.
- Keep compatibility and historical documents linked, but do not present them as the main onboarding route.
- Track simplification decisions in [SIMPLIFICATION.md](SIMPLIFICATION.md).
