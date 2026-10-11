# DSX Adapter Discovery and Architecture Discussion

**Date:** 2026-10-10. **Status:** Research verified against the reference worktrees; architecture and scope proposed, not approved. This is a discussion record, not the approved implementation plan.

**Feasibility reassessment:** See [feasibility-assessment.md](feasibility-assessment.md) before approving the candidate phases. The bridge plus bounded feed model demonstrates protocol/control behavior; proving workload flexibility requires additional workload modeling or hardware. Omniverse is an optional presentation path unless a deployed simulator/telemetry interface is verified. This is a revised recommendation, not approval to change scope.

**Hosted page verified (2026-10-10):** The correct public DSX catalog/Experience URL is https://build.nvidia.com/nvidia/omniverse-dsx-blueprint-for-ai-factories . The rendered NVIDIA blueprint catalog links to it; browser navigation showed the DSX title, Experience tab and Site/GPU/Power configurator. Static page fetching misleadingly returned a 404 for the same URL. This confirms browser-page reachability, not a headless API endpoint, function identity, command execution or metered-power channel. Opening the page automatically displayed a roughly ten-minute session countdown; the browser was promptly navigated to `about:blank`, with no key supplied or scene commands sent. GPU allocation, billing and server-side session termination were not independently verified. Do not assume merely opening Experience is a zero-allocation operation.

**Hosted NVCF configuration found (2026-10-11):** A non-executing HTTPS fetch of that actual public page, followed by JSON decoding of its embedded Next.js data, exposed `nvcfFunctionId: 0ab96455-e029-4b9c-9ce5-de0fbd5ef436`. Its referenced public catalog JavaScript passes `endpointSpec.nvcfFunctionId` into the Experience component alongside `kasPath: https://api.ngc.nvidia.com`. This is direct evidence of NVCF configuration on the deployed frontend, not merely the cloned Kit configuration. It does not independently establish the live backend's health or that our own client can allocate/control a session. Authenticated read-only GET of `/v2/nvcf/functions/0ab96455-e029-4b9c-9ce5-de0fbd5ef436/versions` at `https://api.nvcf.nvidia.com` returned HTTP 404. Treat that as unresolved metadata access/identity, not proof the frontend has no NVCF integration or definitive proof of a particular permission failure. No application scripts were executed or hosted sessions started during this check; credentials stayed inside the request process. See Checkpoint 009 in [journey.md](journey.md).

The same read-only versions lookup at the frontend-advertised `https://api.ngc.nvidia.com` base returned HTTP 403. Direct target metadata access with this credential has therefore not been established through either checked base. Listing/metadata and invocation/session permissions are separate; these results do not prove whether invocation would be allowed. No invocation was attempted.

## 1. Goal and Boundaries

Gridmetry needs to test the path from a flexibility request to AI-factory power control without having the complete DSX ecosystem or physical GPUs available. The proposed adapter will implement the DSX Flex agent role against a simulation backend and connect to the existing NVIDIA-hosted Omniverse environment through a supported interface.

MaxLPS is NVIDIA's power-management software, not a missing MQTT field or metric. Neither clone supplies its controller implementation. Our substitute must be described as a simulated controller, not as MaxLPS compatibility or replication. Actual GPU/rack behavior and workload performance remain unvalidated until compared against appropriate NVIDIA models or hardware.

Omniverse remains hosted at NVIDIA. Local deployment will cover the agreed Exchange test components and this adapter, not a GPU-backed Omniverse installation.

## 2. Research Provenance

| Source | Snapshot |
| --- | --- |
| DSX Exchange clone | `4e8b71f0584913a5f88fa728dc9fe724337aa97b`; clean worktree; commit date 2026-10-08 |
| Omniverse blueprint clone | `d940314d0593bbba1bae51e40ae7f9fd48358e18`; clean worktree; commit date 2026-05-07 |
| DSX Flex contract | AsyncAPI 3.1.0, contract version 1.7.0 |
| Shared conversation | Retrieved 2026-10-10; discusses Batsim/SimGrid, Omniverse, MaxLPS, and a simulator-neutral control interface |

The supplied conversation does not contain a finished Go bridge design, a verified hosted API contract, or runnable integration examples. Its useful principle is to separate workload/power simulation from facility visualization and physical validation.

The secret file was observed only by directory listing. Its contents were not read. No authenticated hosted requests, paid sessions, container deployment, or application tests were performed during discovery.

## 3. What Each Component Does

| Component | Responsibility | Not Established By This Research |
| --- | --- | --- |
| Gridmetry/test publisher | Issue bounded flexibility constraints and inspect response | Complete grid-market integration or reserve qualification |
| DSX Exchange | Secure IT/OT event transport; NATS with MQTT 3.1.1 | Execution of power commands by the broker |
| DSX Flex contract | Load-target commands, power-state snapshots, breach/enforcement events | A supplied implementation of the power controller |
| Proposed adapter | Validate commands, schedule intervals, control a backend, publish status | Proprietary MaxLPS optimization or physical GPU actuation |
| Proposed simulated plant | Explicitly modeled power response within configured bounds | Real GB200/GB300 behavior or workload SLA guarantees |
| Existing Omniverse blueprint | Scene interaction, facility visualization, example electrical/thermal scenarios | A verified hosted headless power-control and measured-telemetry API |

The [Omniverse README](../../omniverse-dsx-blueprint-for-ai-factories/README.md#L5) explicitly describes an extensible blueprint with sample CFD data, not a production-ready application. This does not mean the entire DSX Sim ecosystem lacks physical simulation; it means this inspected blueprint interface is not sufficient evidence of the control backend we need.

## 4. Verified DSX Flex Interface

The [AsyncAPI contract](../../dsx-exchange/schemas/asyncapi/dsx-flex/dsx-flex.yaml#L101) is the source of truth, not a topic name guessed from product descriptions.

| Direction | MQTT Topic | QoS |
| --- | --- | --- |
| Gridmetry/test ISV -> adapter | `grid/v1/isv/{identifier}/loadtarget/set` | 1 |
| Adapter -> Gridmetry/test ISV | `grid/v1/dsx-flex-agent/{identifier}/powerstate/status` | 1 |
| Flex agent -> infrastructure/ISVs | `grid/v1/dsx-flex-agent/{identifier}/powerbreach` | 2 |
| Infrastructure agent -> Flex agent/ISVs | `grid/v1/infra/{infraAgentId}/powerbreach/enforcement` | 2 |

The command topic identifies the publisher, not the destination adapter. Limit subscriptions and permissions to approved publishers. Commands select feeds in their payload. The adapter must not publish infrastructure enforcement as though it were a real BMC/Redfish agent.

### Commands and Status

- Constraints are absolute power limits in `watt`, `kilowatt`, or `megawatt`, not percentages or instructions to add GPU workload. A cap is not a guarantee that natural demand reaches that value.
- A command contains a nonempty `targets` list. Each target requires `interval` and `load_constraint`; the latter may be `null` to remove a constraint and return to default load.
- Start/end values are UTC timestamps. Omitted start means now; omitted end means the constraint remains until replaced. An empty interval object is therefore valid in the schema.
- Omitted or empty `feed_tags` means all feeds. Allocation and precedence across multiple feeds are not sufficiently specified for an aggregate optimizer; one feed avoids that ambiguity initially.
- `best_effort: true` describes nondisruptive adjustment of DPM-enabled jobs. A simple simulated feed model must not pretend to implement that workload capability.
- Status includes feed metadata, active/scheduled targets, `calculated_load`, `in_flight`, transition/heartbeat event, and compliance. `calculated_load` is calculated effective load, not proof of meter readings.
- `correlationid` is optional; when present it is echoed into target/status correlation fields. There is no dedicated application ACK/NACK for load commands. MQTT PUBACK/PUBCOMP only acknowledge protocol delivery.

See [intervals and units](../../dsx-exchange/schemas/asyncapi/dsx-flex/dsx-flex.yaml#L249), [feed state](../../dsx-exchange/schemas/asyncapi/dsx-flex/dsx-flex.yaml#L363), and [payload definitions](../../dsx-exchange/schemas/asyncapi/dsx-flex/dsx-flex.yaml#L486).

### Signing and Compatibility Questions

The [security description](../../dsx-exchange/schemas/asyncapi/dsx-flex/dsx-flex.yaml#L68) requires JWS compact serialization in `data`, using ES256 or RS256 and a key identifier. Verify the signature before validating or applying the decoded payload. JWS signs the inner payload, not the outer CloudEvent or MQTT topic; it does not encrypt the data.

Important issues to capture in executable compatibility fixtures:

1. DSX requires `specversion: "1.0.2"`; the standard CloudEvents specification uses `"1.0"`. Proposed lab profile: follow the pinned DSX wire contract without claiming generic CloudEvents SDK conformance. Raise the discrepancy with NVIDIA; do not silently rewrite it.
2. DSX prose requires signed `data`, but the event schema's `oneOf` also accepts plaintext objects. Proposed adapter profile: signed wire messages only; plaintext fixtures may represent decoded data in tests, not accepted network commands.
3. Constraint prose says greater than zero, while the schema permits zero. Proposed initial policy: reject zero as unsupported; use the contract's explicit `null` release operation. Confirm this with NVIDIA rather than silently clamping.
4. The contract does not define trusted signing-key distribution, signed freshness, replay windows, overlap precedence, or negative command acknowledgements. These are explicit integration decisions, not built-in Exchange features.
5. Unsigned envelope IDs/timestamps can be changed without invalidating the payload signature. Event-ID deduplication alone is not cryptographic replay protection. The isolated lab needs signer allowlists and durable duplicate/scheduling rules; any broader deployment needs an agreed trust/freshness contract.
6. Optional command correlation conflicts with a required correlation field in breach targets. This strengthens the case for starting with load/status only and resolving emergency semantics separately.

See [envelope requirements and wire alternatives](../../dsx-exchange/schemas/asyncapi/dsx-flex/dsx-flex.yaml#L630) and the [CloudEvents specification](https://github.com/cloudevents/spec/blob/v1.0.2/cloudevents/spec.md).

## 5. Go Assessment

**Recommendation: Go for the main adapter.** It suits concurrent MQTT I/O, typed contracts, JOSE validation, deterministic scheduling, bounded control loops, operational metrics, and a small containerized service without a local GPU dependency.

The Exchange clone already has a [Go MQTT client](../../dsx-exchange/local/mqtt-client/pkg/client/client.go#L82) using [Paho MQTT](../../dsx-exchange/local/mqtt-client/go.mod#L5). That is evidence of interoperability, not production-ready adapter code to copy wholesale: its reconnect/QoS behavior needs dedicated tests.

Candidate libraries are `eclipse/paho.mqtt.golang` for MQTT 3.1.1, `go-jose/v4` for JOSE, and a maintained JSON Schema validator. Explicit MQTT protocol version 4 selects MQTT 3.1.1. The similarly named `paho.golang` targets MQTT 5 and is not the default choice for this bus. Pin a supported patched Go release and dependency versions during implementation; do not blindly inherit the entire upstream test module.

Go is not necessarily the best language inside NVIDIA Kit. If NVIDIA permits a Kit extension gateway, Python is the natural small companion. If access requires the supported browser WebRTC SDK, a JS/TS transport component may be necessary. Do not attempt an all-Go reimplementation of NVIDIA's streaming stack merely to keep one language.

## 6. Proposed Adapter Boundary

Keep these responsibilities separate:

- **DSX transport:** MQTT connection, publisher/topic permissions, QoS checks, bounded ingress, reconnect, and status publishing.
- **Protocol codec:** DSX envelope/profile validation, JWS verification/signing, unit conversion, and trace propagation. The Omniverse credential is not a DSX signing key.
- **Scheduler/controller:** per-feed interval handling, explicit precedence, expiry/release, duplicate handling, restart recovery, feasible limits, and backend capability checks.
- **Simulation backend:** source of modeled power/state; replaceable later by a documented NVIDIA simulator or physical-controller integration.
- **Omniverse connector:** deterministic supported commands and observable responses. Presentation must not become the authoritative power measurement by accident.
- **Evidence:** run/configuration identifiers, requested constraints, modeled response, baseline, time base, health, errors, and the provenance of every reported result.

There are two materially different valid configurations:

**Model-owned mode:** Our simulated plant produces power telemetry; Omniverse displays corresponding facility state. A broken visual connection affects presentation health, not the model's mathematics. This proves our messaging/control flow, not NVIDIA power-simulator integration.

**Hosted-simulator-owned mode:** A documented NVIDIA backend executes targets and returns simulator telemetry. Backend failure makes delivery unknown/unavailable; do not silently fall back to a local model while presenting the run as NVIDIA-backed.

These modes must be chosen explicitly and recorded per run. A browser scene update alone cannot establish hosted-simulator-owned mode.

## 7. Proposed MVP Model and Time Semantics

Start with one feed, configurable minimum/default/maximum power, idle or fixed demand, controllable demand, and bounded response dynamics. These are model parameters, not facts inferred from Omniverse colors. Keep absolute watts internal and convert only at contract boundaries.

Use a deterministic clock in tests. Connected runs should default to real-time UTC; accelerated offline simulation is a separate mode and must not claim market response latency. Record simulation time and wall-clock observation time separately when they differ.

Proposed scheduling choices for review: start-inclusive/end-exclusive intervals, an explicit replacement/overlap rule, return to configured default on expiry/release, and recovery from persisted active/future constraints after restart. Preserve the upstream indefinite-interval behavior unless an explicit lab safety policy limits it. Do not treat a future command as active immediately or reapply expired retained messages.

A meaningful acceptance example is: stable modeled baseline -> scheduled lower cap -> bounded modeled transition -> status reflecting target/compliance/in-flight state -> expiry/release -> modeled recovery. Test infeasible caps, duplicates, disconnects, stale state, and unsupported `best_effort` separately. Never report success merely because a message was published or a scene changed.

Requested power limits, predicted effective power, sampled modeled plant response, and genuine metered power are different quantities. Decide and test the mapping to `calculated_load`; retain provenance outside the standard payload, for example in a run journal and isolated simulation agent identity.

Breach payloads use `measured_load`, which claims an actual measurement. Do not emit modeled emergency/enforcement events into a real operational namespace without an agreed test convention. Real infrastructure enforcement, workload scheduling, Batsim integration, thermal CFD solving, and GPU-performance/SLA prediction are outside the recommended first MVP.

The Emerald experiment is useful context, but its workload-specific GPU curves are not automatic GB200/GB300 calibration data. Any later use requires explicit hardware/workload mapping and uncertainty.

## 8. Docker Compose Scope

Exchange is a collection of services and deployment configuration, not a single all-in-one container. The upstream local environment uses Kind/Skaffold. A new Compose deployment must state which components it reproduces.

**Recommended first scope:** a single-site Exchange event-bus-compatible test subset: pinned NATS with MQTT 3.1.1 and JetStream, the Go adapter, a scenario publisher/observer, and an optional fake hosted transport for offline tests. This is not the full Exchange identity, MCP gateway, monitoring, HA, or federation stack.

Proposed deployment requirements:

- Version/digest pinning and actual image validation, not `latest`. The [reference chart](../../dsx-exchange/deploy/nats-event-bus/Chart.yaml#L11) pins NATS chart 2.12.6; image identity must also be verified explicitly.
- Persistent broker storage for stock file-backed MQTT session streams, plus independent adapter state storage. JetStream must be enabled for the client account; a volume does not make upstream memory-backed streams persistent.
- One local replica rather than upstream HA defaults. Bind listeners inside containers for service networking; publish development ports on host loopback only.
- Readiness based on broker/JetStream/MQTT usability, not only container start order. Contract tests must cover QoS 1 and QoS 2 round trips, negotiated QoS, reconnect, and broker restart.
- Explicit lab authentication and topic permissions; plaintext/anonymous settings, if allowed at all, confined to an isolated development profile. Full upstream auth-callout/OAuth/mTLS parity is separate scope.
- Offline tests require no NVIDIA credentials. A hosted profile may mount `omniverse.key` read-only as a secret; it must never enter an image, log, fixture, or generated documentation.
- No upstream repository modifications and no NVIDIA-hosted deployment changes without approval.

NATS [current MQTT documentation](https://docs.nats.io/running-a-nats-service/configuration/mqtt) describes QoS 0/1/2 support from relevant modern versions. Do not repeat the outdated assertion that NATS cannot support QoS 2. The inspected Exchange QoS 2 stream declarations still warrant a targeted compatibility test; their presence alone is not a runtime proof.

If full Exchange service/auth parity is required immediately, inventory and translate the necessary upstream services into Compose as an explicitly larger phase. A broker-only deployment must not be accepted as satisfying that requirement by relabeling it.

## 9. Hosted Omniverse Integration Gate

The [reference client](../../omniverse-dsx-blueprint-for-ai-factories/web/src/lib/OmniverseApi.ts#L51) uses a browser streaming SDK with `StreamType.DIRECT`. Its custom-message requests use application-level IDs; fire-and-forget `sendMessage` is not a command execution acknowledgement. The inspected HTTP extension focuses on agent chat and UI context, not a documented deterministic power-controller API.

The [thermal control](../../omniverse-dsx-blueprint-for-ai-factories/web/src/components/panels/simulation/SimulationPanel.tsx#L95) changes a shader load attribute. The [electrical panel](../../omniverse-dsx-blueprint-for-ai-factories/web/src/components/panels/simulation/PowerFailureTest.tsx#L134) derives displayed power/current from browser arithmetic and example capacities. Neither is evidence of GPU workload execution or a metered facility feedback channel.

The [README credential instructions](../../omniverse-dsx-blueprint-for-ai-factories/README.md#L183) describe a key for the LLM chat backend. We have not inspected the user's key and cannot infer its type, permissions, or usefulness for hosted sessions.

Choose a connector only after NVIDIA supplies the deployed contract:

| Option | Prerequisite | What It Can Prove |
| --- | --- | --- |
| Documented hosted control API | Endpoint, auth/scopes, session/function IDs, commands, telemetry/units, time/cadence, errors, limits | Whatever that API actually executes and reports |
| Approved Kit extension gateway | NVIDIA permits deployment changes and exposes an authenticated, bounded control/readback interface | Extension-defined operations; physical fidelity still depends on its backend |
| Supported SDK/custom-message client | Authorized hosted session and supported headless/sidecar use of the NVIDIA SDK | Scene/extension operations with verified readback, not automatic GPU telemetry |
| Offline model and fake connector | No hosted access required | Adapter/control contract tests only; no NVIDIA interoperability claim |

Do not use natural-language LLM chat as the power-control actuator. Do not assume an NGC/NIM key grants NVCF session access. Check evaluation/licensing conditions with the environment owner before extending or redistributing NVIDIA components.

## 10. Discussion Decisions Needed

1. **Outcome:** Is the first milestone an end-to-end messaging/model demo with Omniverse presentation, or must NVIDIA-hosted simulation be the authoritative power source?
2. **Access:** Has NVIDIA supplied a non-secret service URL and API/SDK documentation, only browser access, or permission to deploy a small Kit extension? Do not paste the key or token-bearing URLs.
3. **Docker scope:** Is a single-site Exchange event-bus test subset acceptable initially, or is the broader identity/auth/gateway stack required from day one?
4. **Model scope:** Accept one feed with configurable bounded modeled response first, or require multiple racks/workload/SLA modeling immediately?

Recommended starting choices are model-owned mode, one feed, a single-site test subset, signed command/status messages, and a hosted connector gated on documented access. Go remains the main service language under all of those choices.

### Architect Responses (2026-10-10)

- **Accepted scope:** Single-site Exchange event-bus test subset in Docker Compose, not the broader upstream services/authentication stack.
- **Accepted scope:** One feed with a bounded modeled response for the initial controller.
- **Deferred decision:** Choose whether modeled power is owned locally or by NVIDIA only after confirming NVIDIA's supported interface. This is not approval to silently choose a local fallback.
- **Research direction:** Determine the Omniverse interfaces from the supplied clone rather than expecting the architect to interpret them.

Additional direct reads establish the intended cloud route as NVCF session services and WebRTC custom messages to Kit: [architecture](../../omniverse-dsx-blueprint-for-ai-factories/ARCHITECTURE.md#L7), [NVCF application dependencies](../../omniverse-dsx-blueprint-for-ai-factories/source/apps/dsx_nvcf.kit#L20), and [frontend configuration fields](../../omniverse-dsx-blueprint-for-ai-factories/web/src/context/OVDSXAppContext.tsx#L3). Those fields provide neither the actual deployed endpoint nor a GPU power-target/readback contract. The [HTTP server](../../omniverse-dsx-blueprint-for-ai-factories/source/extensions/omni.ai.aiq.dsx/omni/ai/aiq/dsx/http_server.py#L1) supplies chat routes; its credential error explicitly identifies the key as LLM access. No key contents were inspected.

The plan should therefore verify supported session transport and power ownership before implementing the NVIDIA connector. Ask NVIDIA only for deployment-specific facts that are absent from the clone, not for facts already established by code. Without a non-secret deployed address/session contract, mocked connector tests can proceed but hosted interoperability cannot be verified.

### Candidate Phases for Plan Approval

Each phase writes a focused failing test, confirms the failure, implements the smallest passing change, reruns focused tests, and records the exact result in the journey. Phase 1's hosted-access gate may remain blocked independently of the offline transport phases; it must not be described as passed without evidence. No implementation is authorized by this discussion record.

1. **Phase 1: DSX Contract and Integration Gate**
	- **Objective:** Establish the signed DSX Flex lab profile and document the repo-supported Omniverse route, required deployed session facts, and permissible connector strategy.
	- **Candidate files:** Go module, protocol/codec package and fixtures; hosted capability record under this plan directory.
	- **Tests:** Valid signed command round trip; invalid signature/algorithm/key rejection; plaintext wire rejection; pinned envelope/version profile; topic/publisher mismatch; watts/kilowatts/megawatts conversion.
	- **Steps:** Write failing codec tests, implement verified profile with maintained JOSE/schema libraries, run tests, and record unresolved hosted capabilities separately. Approve modeled-power ownership before claiming a completed hosted backend design.
2. **Phase 2: Compose Event-Bus Test Environment**
	- **Objective:** Run the agreed single-site Exchange transport subset, not a substitute broker or full Exchange stack by assertion.
	- **Candidate files:** `compose.yaml`, NATS configuration, version pins, transport package/tests, test publisher/observer command, build configuration.
	- **Tests:** MQTT 3.1.1 QoS 1/2 signed-event round trip, granted subscription QoS, broker readiness, reconnect, restart/session persistence, loopback port binding, credential absence from image context.
	- **Steps:** Add failing broker integration tests, configure pinned NATS/JetStream and isolated credentials, implement MQTT transport, and run integration tests using Compose. Offline runs never need the Omniverse key.
3. **Phase 3: One-Feed Scheduler and Controller**
	- **Objective:** Apply verified commands at their UTC interval boundaries with explicit feasibility, overlap, release, deduplication, and restart policies.
	- **Candidate files:** Controller/scheduler, configuration, clock abstraction and persistent state packages/tests; service entry point.
	- **Tests:** Future start, exact start/end boundaries, indefinite interval, explicit release, precedence/replacement, expired replay, duplicate delivery, infeasible/unsupported strategy, restored schedule after restart.
	- **Steps:** Write deterministic-clock tests, implement one-feed state transitions and agreed policy, run focused tests and race checks. Duplicate handling does not claim cryptographic replay protection of unsigned envelope metadata.
4. **Phase 4: Bounded Simulation and DSX Status**
	- **Objective:** Exercise a replaceable one-feed simulated backend and publish status distinguishing calculated power, target, modeled transition, compliance, and presentation health.
	- **Candidate files:** Backend interface, bounded model/test implementation, status publisher, scenario fixtures and run evidence journal.
	- **Tests:** Baseline/cap/release trajectory, idle-load floor, bounded transition, no artificial demand increase to reach a cap, in-flight/compliance semantics, status correlation/heartbeat, truthful unavailable/stale state.
	- **Steps:** Write trajectory/status tests, implement explicit model parameters and provenance, run signed end-to-end load/status tests. This phase validates a test model; it does not make the final NVIDIA-backed run model-owned without the deferred decision being resolved.
5. **Phase 5: Supported Hosted Omniverse Connector**
	- **Objective:** Integrate the supported route established in Phase 1 with the existing NVIDIA deployment, without inventing a power API or using LLM chat as an actuator.
	- **Candidate files:** Omniverse connector and mocked transport tests; small SDK/Python companion only if required and authorized; hosted Compose profile and credential-file loading.
	- **Tests:** Verified command mapping/readback, session disconnect and timeout, authorization failure, missing scene target, unit-to-scene mapping, secret redaction, no silent model fallback when hosted execution fails.
	- **Steps:** Write tests against the confirmed interface, implement its connector, and run mocked integration tests. Hosted verification requires the actual deployment contract and explicit authorization for session/API use. If access is blocked, report this phase blocked, not completed by a fake connector.
6. **Phase 6: Recovery and Repeatable Acceptance**
	- **Objective:** Provide repeatable evidence for the full agreed test loop and its failure modes, with understandable operation instructions and phase checkpoints.
	- **Candidate files:** Compose-backed acceptance suite, scenario runner/reporting, service health/metrics and README updates; plan completion records after reviewed phases.
	- **Tests:** Scheduled curtailment/hold/release, transport duplicates, adapter/broker restart, corrupted persisted state, backend unavailable, stale telemetry, shut-down recovery, separation of wall-clock and accelerated simulation time.
	- **Steps:** Write acceptance failures, make only needed recovery changes, run focused acceptance/race checks, and document exact supported run commands/results and any blocked hosted requirements.

Detailed file paths, dependency pins, scheduling decisions, and acceptance criteria will be finalized in `implementation-plan.md` after plan approval. The first coding phase cannot include an unannounced expansion into multiple feeds, physical GPU control, the complete Exchange stack, or an altered NVIDIA deployment.

## 11. External References

- [Supplied shared conversation](https://chatgpt.com/share/6ac9d556-9028-83ed-8e7f-2deb18bdc2e1)
- [NVIDIA DSX platform roles](https://www.nvidia.com/en-us/data-center/products/dsx/)
- [Omniverse DSX overview](https://docs.omniverse.nvidia.com/dsx/latest/overview.html)
- [NATS MQTT documentation](https://docs.nats.io/running-a-nats-service/configuration/mqtt)
- [CloudEvents 1.0.2 specification](https://github.com/cloudevents/spec/blob/v1.0.2/cloudevents/spec.md)

External documentation was consulted on 2026-10-10 and can change independently of the pinned clones. Implementation must validate its chosen versions with executable contract tests.
