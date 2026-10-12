# Phase 2 Implementation: Local DSX Message Bridge

**Code/review status:** Implemented and `APPROVED`. **Container acceptance:** `PASSED` on 2026-10-11 after fixing the test-only port override. The hosted Omniverse phase and overall MVP remain incomplete.

## Delivered

- Go bridge: MQTT 3.1.1 QoS 1 input -> pinned DSX Flex schema and ES256 signature validation -> one-feed lab policy -> authenticated SSE `command_received` event.
- Explicit cap/release handling, absolute power-unit conversion, duplicate JSON rejection, signer/topic/source checks, expiry checks, bounded ingress/fanout and slow-client handling.
- HTTP liveness/readiness and counters, exact-origin CORS preflight, bounded MQTT connection/reconnect and subscription-grant checks.
- Idempotent local lab credential generation, fail-closed partial setup, a single NATS account with separate publisher/bridge permissions, Compose deployment and a minimal Go image.
- Scenario runner and automated unit, native-broker and Compose integration tests. No NVCF key use, GPU actuation, measured/modelled power status or Omniverse connection in this phase.

## Files and Functions

New components are under `internal/flex/` (codec, policy, JWS, schema and dedup), `internal/bridge/` (pipeline/MQTT), `internal/transport/` (SSE hub/server), `internal/lab/` (config, generated credentials/NATS configuration and counters), `cmd/dsx-lab/` (init/serve/check/scenario), and `integration/` (native and Compose tests). Deployment files are `compose.yaml` and `Dockerfile`; dependencies, README and ignore files were updated. Existing NVCF/access CLI code was left untouched.

The embedded reference contract is pinned to Exchange commit `4e8b71f0584913a5f88fa728dc9fe724337aa97b`, DSX Flex 1.7.0. The original upstream license header is preserved. The lab profile is deliberately narrower: one immediate target/feed, positive bounded cap or `null` release; unsupported future schedules, workload strategies and other feeds are rejected, not silently executed.

## Tests and Review Repairs

Implementation followed failing tests -> minimal code -> focused passing tests. Review repairs included:

- Policy-rejected messages no longer enter the accepted-command dedup window.
- Expiry is rechecked before hub publish and before SSE write.
- Authenticated browser fetch streams have an exact-origin Authorization preflight.
- Reconnect tests cover denied/lowered/missing SUBACK and readiness transitions.
- Publisher/bridge now share one NATS account; the original separate accounts could not exchange commands without imports/exports.
- Partial credential initialization fails closed and never rotates a running lab's material.
- Reconnect backoff is bounded to two seconds and connection attempts to five seconds; the previous Paho default backoff could exceed the scenario timeout.
- Duplicate editor-visible package declarations were removed; remaining IDE warnings concern excluded `integration` build tags, not syntax failures.

Queued commands are revalidated for age at processing time, deliberately failing closed if queue delay makes them stale. `received_at` therefore represents policy acceptance/processing, not proof of original MQTT arrival or remote execution.

## Actual Verification

### Container Gate Resolved

On recheck, the existing Docker daemon successfully ran `dsx-adapter-lab-init help`; the original shim failure was no longer reproducible. No daemon restart, runtime modification or interruption of existing services was performed by this task.

The first Compose rerun exposed a distinct test defect: `ports: []` left the base host bindings in place, colliding with an existing service on port 8090. After explicit user approval, changed only the integration override to `ports: !reset []` and added `TestComposeOverrideClearsPublishedPorts`, which JSON-parses the effective Compose configuration. The regression failed before the fix and passed afterward.

`go test -tags integration ./integration -run '^(TestComposeOverrideClearsPublishedPorts|TestScenarioAcceptsOneSignedCommand|TestBridgeReadinessDropsAndReturnsAcrossBrokerRestart)$' -v -count=1 -timeout=8m` passed all three tests in 22.610 seconds. Signed-command delivery passed in 12.83 seconds, port isolation in 0.06 seconds and broker outage/reconnect in 9.71 seconds. Parent independently reran the isolation regression and tagged vet successfully. Final two-file fix review returned `APPROVED`.

The tests started actual Compose containers and cleaned up their own projects/volumes. Existing `on-site-b` and `site-proxy` services remained running. Docker Compose 2.39.4 was used; the reset-tag override requires Compose 2.24+. No NVIDIA credentials or hosted sessions were used.

### Earlier Checks

Parent-run final checks on 2026-10-11:

- `go test ./...`: passed for all eight default packages.
- `go test -race ./...`: passed.
- `go vet ./...`: passed.
- `DSX_NATS_SERVER=/workspace/bin/nats-server go test -tags integration ./integration -run '^TestNative' -race -count=1`: passed in 3.658 seconds.
- `docker compose config --quiet`: passed.
- `docker compose build init`: passed for the final implementation.
- `go test -tags integration ./integration -run '^TestScenarioAcceptsOneSignedCommand$' -count=1 -timeout=4m`: failed before the init container started with `failed to start shim: unsupported shim version (3): not implemented`.

The native integration test used real `nats-server v2.12.6` and a freshly built lab binary, temporary generated credentials and loopback ports. It verified signed cap/release delivery, unauthenticated stream HTTP 401, invalid zero-watt command rejection without increasing accepted count, broker shutdown -> readiness HTTP 503, restored broker -> HTTP 200 and successful post-reconnect delivery. This is real protocol interoperability evidence, not a substitute claim that Compose ran.

Manual real-broker cap and release scenarios also passed, with `constraint_watts: 800000` and a separate `release: true` event. The temporary manual broker/bridge processes were stopped and their PIDs checked. The automated native test cleans up its own subprocesses/files. No hosted session was started.

The earlier runtime diagnosis found Docker's daemon-associated containerd `v2.2.0` alongside a system `containerd-shim-runc-v2` reporting `2.2.1`, consistent with mixed runtime binaries. That earlier failure is preserved as history, not a current blocker. The daemon had changed before the successful recheck; the precise external repair was not established or attributed to this task.

## Review Status and Remaining Gates

Final code review and the subsequent test-isolation fix review returned **APPROVED**, with no material findings. Reviewer source/diagnostics checks are separate from the parent/implementer's terminal results above. Container acceptance is now complete for the tested local Phase 2 flow. Do not proceed to a hosted scene claim without the supported DSX session/authentication contract.

The normalized SSE event proves a validated command was received. It is not a DSX execution ACK or delivered-power measurement. Dedup is in-memory/best-effort, not durable or cryptographic replay protection; the lab broker remains plaintext/root-process and is not a production deployment.

## Suggested Commit Message

feat: add signed local DSX message bridge

- Validate signed one-feed commands and stream receipt events
- Add isolated Compose lab and scenario runner
- Test real MQTT delivery, rejection and broker recovery
- Record Docker runtime and hosted-access verification gates
