# Plan: DSX Connectivity MVP

**Status:** The architect requested access establishment and MVP progress on 2026-10-10. Proceed with the bounded access-check slice of the previously discussed connectivity work. Review each completed phase before proceeding. Hosted session allocation, service deployment, and scene commands remain gated by identification of the correct DSX deployment and explicit authorization for its use. This plan supersedes the broader six-phase candidate for this first milestone only.

## Goal

Demonstrate one signed DSX Flex command passing through a local Exchange event-bus subset and reaching an authorized NVIDIA-hosted Omniverse scene. No local GPU, workload simulator, physical power actuation, or MaxLPS replication is required. Display evidence and transport health must not be advertised as delivered power.

## Phase 2 Execution Scope (2026-10-11)

**Execution result:** Code implemented and reviewed `APPROVED`; native real-broker acceptance, unit/race/vet checks passed. Containerized signed-command delivery, test port isolation and broker outage/reconnect acceptance also passed on 2026-10-11 after correcting the integration override to `!reset []`. The earlier Docker shim error was no longer reproducible; no daemon repair was made by this task. See [dsx-connectivity-mvp-phase-2-complete.md](dsx-connectivity-mvp-phase-2-complete.md). The hosted Omniverse gate remains separate and unverified.

The architect requested implementation, test cases and review. Phase 2 is authorized; Phase 1 is complete. The next runnable acceptance check is a signed immediate one-feed loadtarget command through Compose-hosted NATS/MQTT 3.1.1 into the Go bridge and out through an authenticated local event stream. This is message delivery, not remote execution or power delivery.

Initial lab policy: one immediate target per command, one configured feed, positive bounded absolute power or explicit `null` release. Reject unsupported future scheduling, multiple targets/unknown feeds, and `best_effort: true` rather than silently acting on them. Require the pinned DSX envelope and signed data; validate payloads against the reference schema plus this narrower policy. Include expiry checks at ingress and event delivery, bounded duplicate/queue handling, topic/signer authorization, broker reconnect/readiness and safe shutdown.

Compose generates isolated local MQTT identities, an ES256 signing key and an event-channel token into ignored runtime secret files without printing values. Do not reuse the Omniverse key, expose MQTT publicly, or publish fabricated power status. The browser-facing channel uses authenticated HTTP streaming compatible with a later fetch-based client. A containerized scenario runner receives an event and publishes its matching signed command, providing a repeatable offline demonstration.

The NVCF configuration advertised by the public DSX page was identified during later research, but metadata lookup failed with HTTP 404 at the NVCF API and HTTP 403 at the frontend-advertised NGC base. Hosted-session client authorization remains an independent Phase 3 gate. No hosted allocation or scene commands are part of Phase 2.

## Phase 3 Connection Check (2026-10-12)

The architect explicitly approved one connection test of at most 60 seconds to the browser-observed DSX blueprint gateway, using the real local Origin, no API-key injection, no retries and no scene changes. That single authorization has been consumed; it does not authorize another live attempt or a scene command.

An experimental official-SDK probe under `plan/` passes offline route, credential, one-attempt and reporting checks. Its pre-live safety review was `APPROVED` after correcting worker/frame escape coverage, timeouts, outgoing datachannel suppression and measured browser cleanup. The one live attempt failed before successful WebSocket upgrade/media; one socket was forwarded and an SDK retry was blocked. Client shutdown was verified after 1.553 seconds. Node TLS connectivity to NGC succeeded independently, but the browser failure's cause was not captured. Hosted connection and scene demonstration acceptance remain unmet. See `plan/journey.md`, Checkpoint 027, for exact evidence; obtain fresh authorization before any additional live diagnostic.

## Phase 3 Browser Receiver POC Verification (2026-10-13)

The architect approved reusing the working NVIDIA page, an authenticated page-side event receiver, signed target/release demonstrations and UI restoration. The receiver is installed into the existing authorized page session; it does not establish a new standalone SDK connection. The staged profile enables the conservative mapper for main-a, EDP 1.5 and an 80% unconstrained baseline. The default Compose lab remains receipt-only.

The live second target/release pair passed with exact command-ID correlation between adapter and browser: target `8a3d45d5-01cb-43d8-a90e-2c19d7765b75` produced 60% / 920 kW, and release `0e00c0d8-f82d-4eca-b9af-6634298a46db` produced 80% / 1,224 kW. Disconnect restored the captured stopped 0% state. The first target passed, but its release was not observed before the two-minute receiver limit; it is not counted as a complete pair. See `plan/journey.md`, Checkpoint 035.

The UI-interoperability POC is verified. Keep the remaining acceptance limits explicit: no metered power, GPU/workload response, remote execution acknowledgement or separately verified scene-color transition; the upstream Exchange deployment and persistent standalone hosted client are not demonstrated. Browser reload/session expiry and manual token connection remain lifecycle limitations. Further sessions and scene tests still require authorization; this record is not an unlimited allocation grant.

## Verified Access State

Read-only requests against `https://api.nvcf.nvidia.com` using process-loaded credentials succeeded on 2026-10-10:

- Private/authorized discovery: HTTP 200, zero functions returned.
- Public discovery: HTTP 200, 226 functions returned, one streaming candidate.
- Streaming candidate details: HTTP 200, name `ai-digital-twins-for-fluid-simulation`, type `STREAMING`, status `DEGRADING`. It is not established as the DSX deployment.
- No function was invoked, session allocated, deployment modified, or GPU workload started. Full response bodies and credentials were suppressed.

Credential acceptance for discovery is not proof of invocation permission, session connectivity, or access to the intended DSX scene. The operator should supply the correct DSX application/session connection instructions or function/version identifiers if the service is not discoverable with this credential. Do not substitute the unrelated public fluid-simulation function.

## Phases

1. **Phase 1: Safe NVCF Access Check**
   - **Objective:** Deliver a small Go CLI that repeats the read-only discovery checks and truthfully distinguishes API access, discovered candidates, and confirmed DSX identity. No session allocation capability.
   - **Files/Functions:** `go.mod`, `cmd/dsx-access/main.go`, `internal/nvcf/` client/credential/report code and tests, README run instructions, plan checkpoints.
   - **Tests:** File credential validation without disclosure; authenticated GET; visibility queries; redirect refusal; TLS verification; bounded response parsing; JSON/schema errors; HTTP 401/403/429/5xx; sanitized errors/reports; no POST/PUT/DELETE; empty private result; unrelated streaming candidate not claimed as DSX; explicit function/version query when supplied.
   - **Steps:** Write minimal failing client tests, run to confirm expected failure, implement the bounded standard-library client/CLI, rerun focused tests and race/vet checks, verify only the already authorized read-only discovery live, then review and record exact results.
   - **Acceptance:** Useful structured output and exit behavior; credential values never printed or put in command arguments; no raw response bodies, environment, account, registry, secret or deployment data in reports; fixed verified NVIDIA host in production CLI. Discovery success must remain distinct from DSX session readiness.
2. **Phase 2: Local DSX Message Bridge**
   - **Objective:** Deliver the agreed single-site Exchange event-bus Compose subset and a one-feed signed-command path to an authenticated browser-facing event channel. Do not require Omniverse connectivity to test local transport.
   - **Files/Functions:** `compose.yaml`, NATS configuration, adapter/scenario commands, Flex codec and bridge packages/tests, container build files, documented example configuration without credentials.
   - **Tests:** Signed command acceptance/rejection, topic/publisher matching, absolute power units and `null` release, MQTT 3.1.1 QoS 1 round trip, duplicates, expired commands, broker readiness/reconnect, authenticated local client delivery, bounded queues, loopback publication and secret exclusion.
   - **Steps:** Write failing codec/transport tests, implement the smallest loadtarget-only bridge with explicit lab policy, run Compose-backed integration tests, and publish an exact runnable local scenario. Add status only with explicitly modeled semantics; do not invent a DSX execution ACK.
   - **Acceptance:** Repeatable local message flow and honest failure reporting. Identify the deployment as an Exchange-compatible event-bus test subset, not the full Exchange product. No public exposure of MQTT or browser embedding of long-lived NVIDIA credentials.
3. **Phase 3: Authorized Hosted Scene Demonstration**
   - **Objective:** Connect a supported NVIDIA browser streaming SDK client to the verified DSX session, translate one approved command, and demonstrate a visible scene update.
   - **Files/Functions:** Minimal viewer/SDK bridge, session integration selected from actual SDK contract, command mapper, session/client result reporting and tests; hosted configuration only after its contract is verified.
   - **Tests:** Supported session config, allowlisted scene operation and unit mapping, disconnected/unauthorized session, send timeout/failure, secret redaction, no claim of remote acknowledgement where the handler provides none; end-to-end local command to visible remote scene when authorized.
   - **Steps:** Obtain the real deployed function/session contract and supported SDK configuration; write mocked failing integration tests; implement the smallest permitted client connection; rerun tests. After authorization for the identified service, perform the bounded hosted connectivity/scene demonstration and collect display evidence.
   - **Acceptance:** A real DSX scene is connected and changes through the Exchange pathway; observations are labelled presentation/interoperability evidence. If correct service identity, access or healthy capacity is unavailable, mark this phase blocked and leave the local bridge usable; never replace it with an unrelated service or label a mock as live success.

## Decisions and Gates

- Use Go for the headless adapter/access client and the supported NVIDIA JS/TS SDK for viewer streaming when required.
- Keep the one-feed, Exchange event-bus subset scope previously accepted. No Batsim, multi-rack model, GPU control or full Exchange stack in this MVP.
- Use the pinned DSX Flex contract and explicitly capture its version/signing inconsistencies before network command acceptance.
- The clone's active frontend is direct-streaming only; unused cloud configuration fields do not establish a complete NVCF client. Do not implement an assumed SDK enum/configuration without confirming its actual contract.
- Read-only NVCF discovery is authorized by the current request. It does not authorize creating functions, editing account permissions, deploying images, or launching an unrelated billable function.
- At each phase, report review status, tests, remaining blockers, and a suggested commit message. The user controls commits and confirmation to continue.

## Open Questions

1. Which NVIDIA-hosted DSX application or function/version is the intended destination, since current discovery does not identify it?
2. Does its owner provide existing-session access, supported session allocation, and permission for the scene commands needed by the demo?
3. What observation/readback mechanism does that deployed scene support beyond successful client sends?
