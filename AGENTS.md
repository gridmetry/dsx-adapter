# DSX Adapter Agent Instructions

## Purpose and Scope

Build a Gridmetry-owned adapter connecting DSX Exchange / DSX Flex messages to a simulation backend and the existing NVIDIA-hosted Omniverse DSX environment. The missing MaxLPS role is a simulated power/workload controller, not an implementation of proprietary NVIDIA MaxLPS.

- Work only within this project unless the user approves otherwise. The sibling `../dsx-exchange` and `../omniverse-dsx-blueprint-for-ai-factories` clones are reference material, not modification targets.
- The architect authorized MVP progress on 2026-10-10. Execute the bounded phases in `plan/implementation-plan.md`, with review and user continuation gates. Hosted DSX identity/access is still unverified; do not start a session or modify deployments to bypass that gate.
- Use `plan/` for the implementation plan, research, decisions, discussion records, and planning/implementation checkpoints. This file and the human-facing `README.md` are the requested root-level exceptions.
- After approval, use `plan/implementation-plan.md` as the plan of record. Update `plan/journey.md` after meaningful research, decisions, implementation, validation, and blockers. Never record success without evidence.

## Secrets and Deployment

- `omniverse.key` is a secret: never read it into agent context, print it, copy it into documentation, commit it, or include it in an image build context.
- Treat the NVIDIA endpoint, key type, function/session identifiers, and access permissions as unverified until confirmed. The existence of a key does not establish API access.
- Runtime code may load the key from a read-only file/Compose secret; redact authorization headers and payload credentials from logs and errors.
- Do not start paid hosted sessions, call mutating hosted APIs, or deploy changes to NVIDIA's environment without explicit approval. Prefer offline contract tests and mocks first.

## Contracts and Modeling

- The DSX Flex AsyncAPI contract in `../dsx-exchange/schemas/asyncapi/dsx-flex/dsx-flex.yaml` is authoritative. Pin the reference revision when implementation starts.
- Use the contract's actual topics, units, signatures, intervals, and status semantics. MQTT delivery acknowledgement is not proof of command execution or delivered power.
- Do not invent NVIDIA HTTP endpoints, MaxLPS APIs, or measurement claims. Distinguish verified repository behavior, assumptions, and proposed adapter contracts.
- Separate message transport, signature verification, interval scheduling, simulation/control, telemetry, and Omniverse presentation. Keep an explicitly simulated backend independent of NVIDIA connectivity.
- Keep measured, modeled, projected, and visualization-only values distinct. Do not claim market qualification, physical GPU behavior, or SLA compliance from a scene animation.
- Make time, units, power boundary, interval expiry, restart behavior, duplicate delivery, and safety limits explicit and testable.

## Engineering Approach

- Go is the proposed service language, subject to hosted transport confirmation. Prefer established MQTT 3.1.1, JOSE, and schema-validation libraries over custom implementations.
- Follow focused TDD: failing behavior test, smallest change, focused validation immediately afterward. Use deterministic clocks for scheduling/model tests and run race checks for shared controller state.
- Keep per-feed control bounded and reject unsupported or infeasible inputs explicitly. Never silently advertise modeled delivery as measured delivery.
- Record exact test commands/results and unresolved integration prerequisites in the journey. Do not document build/run commands as working before the corresponding tooling exists and has been verified.
