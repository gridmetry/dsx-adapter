# Gridmetry DSX Adapter

Connect Gridmetry's flexibility messages to a NVIDIA-hosted Omniverse demonstration without a local GPU. The first working component checks NVCF access safely; the local Exchange bridge and hosted scene connection are subsequent MVP phases.

## Current Status

**Phase 1 (complete): a small, read-only Go CLI (`cmd/dsx-access`) that repeats the authorized NVCF discovery checks and reports results as sanitized JSON.** No session is ever started and no invocation is performed.

**Phase 2 (this update): a local, Compose-hosted DSX Flex message bridge.** One signed `grid.loadtarget.set.v1` command, published over MQTT 3.1.1 QoS 1 to an isolated NATS/JetStream broker, is verified (schema + ES256 JWS + lab policy), deduplicated, and republished as a normalized event on an authenticated local HTTP Server-Sent-Events stream. This is message delivery through an Exchange-event-bus-compatible **test subset**, not the full Exchange product, not remote execution, and not delivered/measured power. No Omniverse or NVCF connectivity is used or required by this phase. See [Phase 2 details](#phase-2-local-dsx-message-bridge) below.

Live NVCF discovery (Phase 1) succeeded, but found no identified DSX deployment. The one discovered public streaming candidate is a different, degrading fluid-simulation demonstration. The correct DSX application/session connection details remain necessary. See [Phase 1 results](plan/dsx-connectivity-mvp-phase-1-complete.md).

Research, decisions, and checkpoints belong under [plan/](plan/). The current findings and open decisions are in [plan/discovery.md](plan/discovery.md); checkpoints are in [plan/journey.md](plan/journey.md); the phased plan is in [plan/implementation-plan.md](plan/implementation-plan.md). Agent guidance is in [AGENTS.md](AGENTS.md).

## Phase 1: `dsx-access` NVCF Discovery CLI

Go module `gridmetry/dsx-adapter` (Go 1.25, standard library only, no third-party dependencies). Built and tested with `go1.25.11`.

### What it does

- Loads a bearer token from a local file (`--key-file`), bounded and validated, never printed or echoed. The value must be a single token of alphanumeric characters plus `.`, `_`, `-` only (10–4096 chars); quotes, whitespace, and shell metacharacters are rejected.
- Sends bounded, read-only `GET` requests to the fixed production host `https://api.nvcf.nvidia.com` only. There is no flag to target any other host.
- Discovery mode (default): two separate `GET /v2/nvcf/functions` requests — one with `visibility=private&visibility=authorized`, one with `visibility=public` — and reports HTTP status, a safe status classification, total function count, and any `STREAMING`-type candidates found (each with a `name_matches_dsx` heuristic boolean; the free-text name itself is never printed). A `functions` field that is missing or `null` is reported as an error, not a silent success; an empty `functions` list is a legitimate, successful result.
- Detail mode (`--function-id <uuid> --version-id <uuid>`, both required together): a single `GET /v2/nvcf/functions/{id}/versions/{version}?includeSecrets=false` request. The live API wraps the function object under a `function` key (`{"function": {"id": ..., "versionId": ..., "status": ...}}`); a missing/null/empty wrapper, or a response whose `id`/`versionId` do not match the requested values, is treated as not found with an error, never as a successful match. `function_active` is only true when the returned `status` is exactly `ACTIVE`.
- Candidate fields are validated before they are ever serialized: `id`/`version_id` must be syntactically valid UUIDs or the candidate is excluded entirely, and `function_type`/`status` are accepted only as exact-case known API enum values — any other value (including injected text) is sanitized to empty rather than echoed.
- Never issues `POST`/`PUT`/`PATCH`/`DELETE`, never invokes a function, never allocates a session or token.

### What it does not do

- It does not prove a DSX/Omniverse session exists or is reachable. `session_access_verified` and `dsx_identity_verified` are always `false` in the output.
- A name match against a candidate is a heuristic string check, not authoritative identification of a deployment. `candidates_discovered` reflects only that heuristic.
- An explicit detail lookup only confirms that the requested function/version metadata was read (`target_metadata_verified`) and whether its status is exactly `ACTIVE` (`function_active`); neither implies a DSX session is ready or that the function is related to DSX/Omniverse.
- It is not the adapter, Exchange bridge, Compose stack, or Omniverse connector described above; those remain future phases.

### Usage

```sh
go run ./cmd/dsx-access --key-file ./omniverse.key
go run ./cmd/dsx-access --key-file ./omniverse.key --function-id <uuid> --version-id <uuid>
go run ./cmd/dsx-access --help
```

`--key-file` is required. `--function-id`/`--version-id` are optional but must be supplied together as valid UUIDs. No secret value is ever accepted as a flag value — only a file path.

Output is a single JSON object on stdout, for example:

```json
{
  "credential_used": true,
  "discovery": [
    {"visibilities": ["private", "authorized"], "http_status": 200, "status_class": "ok", "function_count": 0},
    {"visibilities": ["public"], "http_status": 200, "status_class": "ok", "function_count": 226,
     "streaming_candidates": [{"id": "...", "version_id": "...", "function_type": "STREAMING", "status": "DEGRADING", "name_matches_dsx": false}]}
  ],
  "candidates_discovered": false,
  "target_metadata_verified": false,
  "function_active": false,
  "dsx_identity_verified": false,
  "session_access_verified": false
}
```

### Exit codes

- `0` — explicit detail metadata was verified against the requested ids, or discovery found a streaming candidate whose name heuristically matches DSX/Omniverse (`candidates_discovered`). Neither ever implies a DSX session or identity.
- `1` — an execution error occurred: unreadable/invalid credential file, request failure, or any non-200 HTTP status, including `404` (a non-200 response to an explicit detail request cannot be distinguished from a request/identifier problem, so it is never reported as a clean unresolved result).
- `2` — requests completed cleanly (HTTP 200) but no target was resolved; automation must not assume a connected scene. `--help`/usage errors also exit `2` and print safely to stderr.

Zero functions returned is reported as a normal, successful `status_class: "ok"` result — it is never represented as a bad credential.

### Security properties

- Credential loading is bounded (max 8 KiB read) and validated (single token, alphanumeric plus `.`/`_`/`-` only, 10–4096 chars); errors never echo file contents or paths.
- The production HTTP client disables redirect following, disables implicit proxy-environment use, enforces TLS 1.2+, and bounds request time and response body size (5 MiB list / 1 MiB detail).
- HTTP status codes are mapped to a safe classification (`auth_denied` for 401, `permissions_or_auth_unknown` for 403, `rate_limited` for 429, `service_unavailable` for 5xx) — the CLI never retries automatically and never prints raw response bodies, headers, or request URLs in errors or reports.
- Candidate output includes only `id`, `version_id`, `function_type`, `status`, and a `name_matches_dsx` boolean — never the free-text function name, container image, environment, secrets, or account data. `id`/`version_id` must be valid UUIDs and `function_type`/`status` must be exact-case known API enum values, or the field is dropped/sanitized rather than echoed raw.
- Signing-key/PEM parsing, session allocation, and invocation are explicitly out of scope for this phase.

### Tests

```sh
gofmt -l .
go test ./internal/nvcf ./cmd/dsx-access
go test -race ./...
go vet ./...
```

All acceptance tests run against an in-process `httptest` TLS server with fake fixture data (fake UUIDs, fake tokens, fake names) — no network access and no real credential is ever used in tests.

## Phase 2: Local DSX Message Bridge

Go module additions (Phase 2): `github.com/eclipse/paho.mqtt.golang` (MQTT 3.1.1 client), `github.com/go-jose/go-jose/v4` (JWS ES256), `github.com/santhosh-tekuri/jsonschema/v6` (JSON Schema), `gopkg.in/yaml.v3` (parsing the embedded contract only). Phase 1's `cmd/dsx-access` and `internal/nvcf` are unchanged.

### Scope

A single-site Exchange-event-bus-compatible **test subset**: one isolated NATS broker (MQTT 3.1.1 + JetStream), this Go bridge, and a self-contained scenario runner — not the full upstream DSX Exchange stack (no identity/auth-callout/HA/federation), and not a connection to NVIDIA/Omniverse. The accepted flow is:

```
gridmetry (scenario) --MQTT QoS1 signed command--> NATS/MQTT broker --> bridge --SSE (authenticated)--> subscriber
```

Lab policy (see `internal/flex/policy.go`): exactly one target per command, one configured feed (`main-a`), a positive bounded absolute watt constraint or explicit `null` release, no `best_effort`, no future-scheduled start, bounded envelope/decoded-payload size, and a 5-minute default envelope freshness window. Commands are validated against a build-embedded, pinned copy of the DSX Flex AsyncAPI contract (`../dsx-exchange/schemas/asyncapi/dsx-flex/dsx-flex.yaml` at commit `4e8b71f0584913a5f88fa728dc9fe724337aa97b`, copied verbatim into `internal/flex/schema/dsx-flex.yaml` with its original NVIDIA Copyright/SPDX header preserved and no header added to our own code) using `santhosh-tekuri/jsonschema/v6`, plus this narrower lab policy on top.

### Packages and binary

- `internal/flex`: CloudEvents 1.0.2 envelope validation, strict (duplicate-key-rejecting) JSON decoding, ES256 JWS sign/verify (`go-jose/v4`), the embedded-schema codec pipeline, lab policy, and a bounded 1024-entry/TTL dedup ring.
- `internal/flex/schema`: the embedded pinned contract and compiled JSON Schemas for the envelope and decoded payload.
- `internal/bridge`: a bounded-queue processing pipeline (never blocks the MQTT callback; drops and counts on a full queue) and the paho MQTT 3.1.1 subscriber (QoS 1, checks the actual granted SUBACK QoS, reconnects and resubscribes, readiness reflects only a currently-confirmed subscription). The paho client's own per-attempt connect timeout and reconnect backoff ceiling are explicitly bounded (5s / 2s) rather than left at Paho's defaults (30s / 10 minutes), so a broker restart is observed and recovered from within a bounded lab/test timeframe instead of potentially backing off for minutes.
- `internal/transport`: the SSE hub (bounded per-subscriber buffer; a slow subscriber is disconnected, never allowed to block delivery to others; an event whose `end_time` has already passed is dropped, not written, immediately before each write) and the HTTP server (`GET /events` bearer-token auth with constant-time comparison, no query-token acceptance, default-deny CORS with an exact-origin `OPTIONS` preflight for cross-origin fetch — never a wildcard, never a substitute for the bearer check on the actual `GET`; `GET /healthz` always-200 liveness plus accepted/rejected/duplicate/queue-dropped/expired counts; `GET /readyz` true only while the MQTT subscription is actually active).
- `internal/lab`: bounded config/secret-file loading, the one-time secret/key generator used by `init`, and nats.conf rendering (all interpolated values are validated against a restricted charset before substitution; no string escaping of arbitrary input). The generated config defines a single shared NATS account holding both the lab publisher (publish-only on the one command subject) and the bridge (subscribe-only on that same subject) — not two separate accounts, which would need explicit imports/exports to let the one subject flow between them at all.
- `cmd/dsx-lab`: one binary, four subcommands — `init` (idempotent; generates the ES256 signing keypair, MQTT credentials for the lab publisher and the bridge, an SSE bearer token, and nats.conf into the shared `secrets` volume; never rotates existing trust material; refuses to generate anything, and exits nonzero, if the directory holds only some of the required files — the signature of a previous run interrupted partway through), `serve` (the bridge/HTTP service), `check` (a bounded HTTP GET used as the Compose healthcheck, since the minimal images here do not ship curl), and `scenario` (subscribes to the SSE stream first, waits for its ready signal, then publishes one signed command and verifies a matching event is received).

### Running it

```sh
docker compose run --rm init
docker compose up -d --build broker bridge
docker compose run --rm scenario
```

`broker` listens on `127.0.0.1:1883` (host loopback only; never publicly exposed) and `bridge` serves `0.0.0.0:8090` inside its container, published to the host as `127.0.0.1:8090`. `scenario` and `bridge` run as uid/gid `10001:10001`; `init` runs as root only to `chown`/`chmod` the generated secrets (dirs `0700`, files `0600`/`0640`) before dropping to that uid. Generated credentials are never printed, logged, or baked into the image; they live only in the `secrets` named volume (excluded from Git/Docker build contexts). The NATS broker container itself is left running as its image default (root) so it can read the root-written/owned `0640` config and manage its own `natsdata` JetStream volume; this is a documented simplification, not a hardening claim about the broker process.

### Staged Live Test

An isolated `dsx-adapter-staged` project is available using [compose.stage.yaml](compose.stage.yaml). It publishes MQTT at `127.0.0.1:21883` and HTTP at `127.0.0.1:28090`, leaving the existing site proxy on port 8090 untouched. The override uses Compose's `!override` tag, requiring Docker Compose 2.24.4+; its merged bindings were verified with Compose 2.39.4. Generic YAML editors may flag that Compose-specific tag even though Compose accepts it.

Run from this project directory:

```sh
docker compose -p dsx-adapter-staged -f compose.yaml -f compose.stage.yaml up -d --build --wait --wait-timeout 90 broker bridge
docker compose -p dsx-adapter-staged -f compose.yaml -f compose.stage.yaml run --rm --build --no-deps scenario --watts 800000
docker compose -p dsx-adapter-staged -f compose.yaml -f compose.stage.yaml run --rm --no-deps scenario --release
```

The scenario opens an authenticated SSE connection using generated lab credentials inside its container, publishes a fresh signed command, then reports the matching receipt event. It does not need the NVIDIA key and does not connect to Omniverse.

Check the staged service and counters:

```sh
curl --fail http://127.0.0.1:28090/readyz
curl --fail http://127.0.0.1:28090/healthz
docker compose -p dsx-adapter-staged -f compose.yaml -f compose.stage.yaml ps
```

On 2026-10-12 the cap and release both passed. A signed zero-watt request was rejected without increasing accepted count; an unauthenticated `/events` request returned HTTP 401. Final counters were `accepted: 2`, `rejected: 1`, and zero duplicates, queue drops and expired commands. Counters will change as scenarios are repeated. Services were intentionally left running after this test.

To stop this staging project without deleting its credential or broker-data volumes:

```sh
docker compose -p dsx-adapter-staged -f compose.yaml -f compose.stage.yaml down
```

These results establish real local MQTT-to-SSE interoperability. `800000` is the requested constraint in watts, not measured power consumption or verified curtailment. Hosted Omniverse scene changes remain Phase 3 work.

### Tests

```sh
gofmt -l internal/flex internal/bridge internal/transport internal/lab cmd/dsx-lab integration
go build ./...
go test ./...
go test -race ./...
go vet ./...
```

These require no Docker and no credentials; they cover schema validation, envelope/JWS verification (wrong key, wrong kid, tampered payload, alg=none rejection), lab policy (zero/negative/oversized constraints, multiple targets, unknown feed, `best_effort`, future start, expired interval/envelope), duplicate-JSON-field rejection, the bounded dedup ring (exact duplicate vs. conflicting id reuse vs. TTL expiry vs. capacity eviction, and that a policy-rejected message never poisons the cache against a later valid resubmission of the same id), delivery-time `end_time` expiry re-checked with an injectable clock immediately before both the hub publish and the SSE write (not just at ingress), the SSE hub/auth/CORS-preflight/readiness, the MQTT SUBACK-driven readiness/reconnect decision (granted/denied/lowered QoS, via a fake of the narrow Paho surface used, since the concrete SUBACK token type cannot be constructed outside its package), and the bounded ingress queue/pipeline counters.

A Docker-backed suite also exists, gated behind a build tag so the default suite above never needs Docker:

```sh
go test -tags integration ./integration/...
```

It brings up an isolated `docker compose -p dsxlab-it-<random>` project (host ports disabled via `integration/override.yaml`, so it cannot collide with a human's own running demo stack), runs `init`/`up`/the `scenario` container, and asserts the scenario observed a matching received event; a second test in the same suite stops, then restarts, the `broker` container and asserts the bridge's `/readyz` (checked directly from inside the bridge container with the same `check` subcommand its Compose healthcheck uses, not inferred from Compose's health-string polling interval) drops while the broker is down and returns once resubscribed, followed by a post-reconnect `scenario` run. Both always tear their own stack down (`down -v`), never the operator's.

The integration suite requires Docker Compose 2.24+ for `ports: !reset []`. Plain `ports: []` does not clear inherited host bindings. `TestComposeOverrideClearsPublishedPorts` verifies the actual merged configuration before relying on that isolation.

A third, Docker-free variant (`integration/native_test.go`, same build tag) exercises the identical broker+bridge+SSE contract without any container runtime: a real external `nats-server` binary (MQTT 3.1.1 + JetStream) and a `dsx-lab` built fresh from this repo are started as plain, bounded, test-owned subprocesses on randomly reserved loopback ports. It is skipped, not failed, unless an external `nats-server` binary is explicitly opted in:

```sh
DSX_NATS_SERVER=/path/to/nats-server go test -tags integration ./integration -run '^TestNative' -v -count=1
```

It asserts a signed accept command and a signed release command each flow end to end over SSE; an unauthenticated `GET /events` is rejected (`401`); a policy-rejected zero-watt command is counted as `rejected` and never `accepted` or delivered; and that SIGTERM-ing the broker drops `/readyz` to `503` within a bounded 10s poll, while restarting it against the same JetStream store brings `/readyz` back to `200` within a bounded 20s poll and resumes delivery of further signed commands — all without restarting the bridge process itself. All of its subprocesses are stopped in `t.Cleanup` regardless of outcome; no secret file content is ever read into the test process or logged, only file paths and HTTP status/counts.

### Known limitations (read before relying on this for anything beyond a local demo)

- **Dedup is best-effort, not cryptographic replay protection.** The CloudEvents `id`/`time` fields sit outside the JWS signature, so they are not guaranteed authentic; the 1024-entry/TTL dedup ring is in-memory only and does not survive a bridge restart.
- **No exactly-once or durable delivery is claimed anywhere.** A full ingress queue drops and counts (`queue_dropped`) rather than blocking; this is reported honestly on `/healthz`, not hidden.
- **The bridge never claims DSX execution.** The normalized SSE event means "a signed, schema-valid, policy-accepted command was received"; there is no application-level DSX ACK, modeled/measured power, or Omniverse path in this phase.
- **This is not a fully DSX-Exchange-compliant profile.** It deliberately narrows the upstream contract (one target, one feed, no `best_effort`, no future scheduling, signed-wire-only) and documents every deviation in `internal/flex/policy.go` rather than silently reinterpreting the spec.
- **The broker's own process runs as root** in this Compose profile (see above); do not reuse this Compose file for anything beyond an isolated local lab without revisiting that choice.

### Observed results (this environment)

- `go build ./...`, `go test ./...`, `go test -race ./...`, `go vet ./...` all pass; `gofmt -l` is clean for every Phase 2 file (one pre-existing, untouched Phase 1 file — `cmd/dsx-access/main_test.go` — is flagged by `gofmt -l` from before this phase).
- The native (Docker-free) suite above was actually run live in this environment against a real `nats-server v2.12.6` binary and a freshly built `dsx-lab`: `DSX_NATS_SERVER=/workspace/bin/nats-server go test -tags integration ./integration -run '^TestNative' -v -race -count=1` passed (`TestNativeBrokerBridgeScenario`, ~2.7s), including the broker SIGTERM/restart readiness drop-and-recovery and post-restart delivery. This is separate from the successful Docker-backed verification below.
- `docker compose config` and `docker compose --profile scenario config --services` both resolve all four services correctly; `docker compose build init` succeeds against the pinned, registry-verified base images (`golang:1.25.11-bookworm`, `gcr.io/distroless/static-debian12:nonroot`, `nats:2.12.6-alpine`), producing a ~10 MB final image.
- **Docker-backed acceptance now passes (2026-10-11).** The previous `unsupported shim version (3)` failure is no longer reproducible; the daemon was already working when rechecked, and no daemon changes were made during this verification. A separate port collision exposed the ineffective integration override; replacing it with `!reset []` fixed that test setup without interrupting existing services.
- `go test -tags integration ./integration -run '^(TestComposeOverrideClearsPublishedPorts|TestScenarioAcceptsOneSignedCommand|TestBridgeReadinessDropsAndReturnsAcrossBrokerRestart)$' -v -count=1 -timeout=8m` passed all three tests in 22.610 seconds: signed MQTT-to-SSE delivery, merged-config port isolation, and broker outage/reconnect delivery. Test-owned containers/volumes were cleaned up; existing `on-site-b` and `site-proxy` remained running. `go vet -tags integration ./integration` also passed. The demo's fixed loopback ports still need to be free when starting the non-test Compose stack.

## Planned (Not Yet Built)

Phase 3 (gated): a supported NVIDIA browser streaming SDK client connected to a verified, authorized DSX hosted session, translating one approved command into a visible scene update. No hosted session allocation, scene commands, or browser SDK code exist yet; see `plan/implementation-plan.md` for the gating conditions.

## What It Is Not

This is not NVIDIA MaxLPS, and it does not provide real GPU power control. A simulated substitute can exercise the messaging and control workflow; it cannot establish physical power response, real workload SLAs, or energy-market qualification.

The cloned Omniverse blueprint demonstrates digital-twin visualization and interactive scenarios. Its visible controls do not, by themselves, establish a headless GPU power-control or metered-telemetry service.

## Proposed Flow

Gridmetry / test publisher -> isolated NATS/MQTT broker (Phase 2, implemented) -> Go bridge (Phase 2, implemented) -> authenticated local SSE stream (Phase 2, implemented) -> [Phase 3, not yet built] supported NVIDIA hosted connector -> Omniverse scene.

Omniverse will be connected through a supported hosted interface once NVIDIA confirms it. Scene updates and modeled power must remain distinguishable from measured telemetry.

## Reference Repositories

- DSX Exchange: https://github.com/dsx-ai-factory/dsx-exchange
- Omniverse DSX blueprint: https://github.com/NVIDIA-Omniverse-blueprints/omniverse-dsx-blueprint-for-ai-factories
- Supplied architecture discussion: https://chatgpt.com/share/6ac9d556-9028-83ed-8e7f-2deb18bdc2e1

The sibling clones are research inputs; this adapter is a separate project. Phase 2 adds a Compose-based local test environment (`compose.yaml`); the upstream Exchange development stack uses Kind/Skaffold, so this minimal broker deployment is not described as the full Exchange stack.

## Credentials

`omniverse.key` contains a local secret. Do not display it or commit it. It is excluded from Git and Docker build contexts. After the hosted contract is confirmed, runtime code will load credentials from a read-only file or Compose secret, not bake them into an image. A key alone does not identify the hosted endpoint or grant simulation-control access.

Phase 2's lab secrets (ES256 signing keypair, MQTT credentials, SSE bearer token, generated `nats.conf`) are unrelated to `omniverse.key`, generated locally by `docker compose run --rm init`, and live only in the `secrets` named Docker volume — never read from or written into the Git/Docker build context, never logged, and never printed by any `dsx-lab` subcommand.
