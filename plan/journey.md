# DSX Adapter Journey

## Checkpoint 001: Discovery Started (2026-10-10)

**Stage:** Planning; no application implementation authorized yet.

**Goal:** Agree a Go adapter that bridges local DSX Exchange / DSX Flex and NVIDIA-hosted Omniverse, with an explicitly simulated substitute for the missing MaxLPS control role. Deliver a local Docker Compose test environment after plan approval.

**Inputs inspected:** Supplied shared conversation; sibling Exchange and Omniverse clones; adapter directory listing. The directory initially contained only `omniverse.key`. The key contents were not accessed.

**Initial findings:**

- The shared conversation proposes a simulator-neutral control boundary and distinguishes workload simulation, facility visualization, and physical validation. It does not supply a finished bridge protocol.
- Exchange uses NATS with MQTT 3.1.1 and a signed DSX Flex AsyncAPI contract. Its upstream development deployment is Kind/Skaffold, not a ready-made Compose stack.
- The Omniverse clone provides streaming/data-channel scene controls and an agent-chat HTTP surface. A hosted, deterministic power-control and metered-telemetry contract has not been verified.
- Go is a promising service language because the Exchange clone already uses Go MQTT tooling. The hosted Omniverse transport could still require a companion SDK component.

**Local hypothesis:** A DSX Flex-compatible simulated agent can exercise the messaging/control loop, but the cloned blueprint alone does not establish that this loop can drive and measure physical GPU power through the hosted environment.

**Discriminating check:** Compare the authoritative DSX Flex command/status schemas with the Omniverse control implementation, then obtain NVIDIA's deployed endpoint/authentication/control/telemetry contract. A documented hosted power-control interface would change the backend design.

**Documentation created:** Root project README and agent instructions; this checkpoint log; Git/Docker secret exclusions. These are planning safeguards, not application scaffolding.

**Next:** Complete the contract analysis, record verified evidence and decisions, and discuss scope with the architect before issuing an approved implementation plan.

## Checkpoint 002: Contracts and Boundaries Researched (2026-10-10)

**Stage:** Planning; implementation remains unapproved.

**Evidence:** Read-only exploration of each sibling repo, followed by a focused integration research pass, direct inspection of Flex/Omniverse controls, and official NVIDIA/NATS documentation. Recorded clean reference revisions in [discovery.md](discovery.md).

**Findings that sharpen the design:**

- MaxLPS is an external controller capability, not a missing schema field. A local substitute must be identified as simulated.
- Exchange has usable command/status topics and absolute power/UTC interval schemas. Application ACKs, overlap rules, signing trust, and cryptographic replay protection are not fully specified.
- The wire contract has CloudEvents-version and signed-versus-plaintext inconsistencies; they require explicit tested compatibility decisions.
- Modern NATS supports QoS 2. Broker/JetStream configuration and reconnect behavior still need executable tests; none were run during discovery.
- Inspected Omniverse controls change scene attributes or browser-derived electrical values. Hosted dynamic power control/telemetry and credential scopes remain unverified.
- Go fits the core service; Python or JS/TS may be appropriate for a small supported NVIDIA connector, if required and permitted.

**Proposed, not approved:** One-feed modeled controller; command/status MVP; strict signed-message lab profile; single-site Compose event-bus subset; separate hosted integration gate. Keep modeled power distinct from measured power and from scene presentation.

**Changes:** Added the evidence-backed architecture discussion in [discovery.md](discovery.md) and linked it from the project README. No Go module, application code, Compose deployment, or hosted extension created.

**Validation so far:** VS Code diagnostics reported no errors for the initial README, agent instructions, checkpoint log, and ignore files. `git log -1` and `git status --short` established the reference snapshots and clean worktrees. The key was not read and neither sibling repo was changed.

**Next:** Review the research documentation for factual errors, present candidate phases, and resolve the four scope/access questions with the architect. Write the detailed plan of record after approval, then implement and checkpoint one tested phase at a time.

## Checkpoint 003: Architect Scope Responses and Review (2026-10-10)

**Stage:** Planning; candidate six-phase sequence presented for approval, not implemented.

**Accepted scope:** Single-site Exchange event-bus Compose subset and one feed with bounded modeled response. **Deferred:** Authoritative power ownership pending verification of NVIDIA's interface. The architect directed us to establish repository-supported interfaces ourselves.

**Follow-up evidence:** Directly inspected Omniverse architecture, NVCF Kit configuration, frontend stream/catalog fields, and the agent HTTP server. The intended hosted route is streaming/session services plus custom messages, not an established GPU power-control REST API. Deployment-specific addresses/scopes are absent from those generic interfaces.

**Review:** Documentation review returned `APPROVED`, with no material factual or security issues. This approves research accuracy, not implementation scope or hosted access.

**Validation:** VS Code diagnostics reported no errors for the research documents. A first local-link check failed because terminal `rg` is unavailable; the equivalent `grep` check passed with `Local Markdown link targets exist; credential exclusions are present.` No secret content was read and no hosted/container test was claimed.

**Changes:** Added the architect's responses, source-backed hosted-interface gate, and candidate six-phase TDD sequence to [discovery.md](discovery.md). No application or deployment files created.

**Next:** Obtain approval or revisions for the phase sequence. Then write `implementation-plan.md` under this directory and start only the approved phase. Resolve missing deployment-specific NVIDIA facts through supplied non-secret connection details or the environment owner; do not infer them from the key.

## Checkpoint 004: Flexibility Demonstration Reassessed (2026-10-10)

**Stage:** Planning paused at the architect's request; no phase implementation authorized.

**Question:** Can Exchange plus NVIDIA-hosted Omniverse actually demonstrate Gridmetry's flexibility approach, and what alternatives fit better?

**Evidence:** Focused read-only research traced browser custom messages through Kit's attribute/color handlers. Those paths do not emit solver-generated power samples or correlated execution results. Official Omniverse architecture describes separate delegate/surrogate services, but their deployed availability remains unverified. Consulted Batsim protocol/time, SimGrid energy modeling, DSX Air and NVIDIA power-management/telemetry documentation.

**Revised recommendation:** Keep the accepted single-feed/Exchange scope for an offline protocol/control proof. Omniverse can be an optional connected view with supported authorized integration. A substantive workload-flexibility experiment needs an independent workload/power model or hardware; a target-echoing model cannot substantiate it. Batsim/SimGrid is a promising software-only backend candidate, while a small permitted NVML/DCGM GPU trial is a later causal physical check. Neither extra scope is approved.

**Changes:** Added [feasibility-assessment.md](feasibility-assessment.md) and linked it from the discovery record. Recorded baseline/energy/rebound and workload-evidence acceptance criteria, model-clock caveats, and precise questions about NVIDIA's simulation services.

**Verification limits:** No credentials read, hosted sessions invoked, GPU limits changed, containers launched, or application tests run. Conclusions are static feasibility findings and documentation-backed alternatives.

**Next:** Discuss which claim the first demo should prove before revising or approving the implementation plan. The existing candidate sequence is not an approved plan.

## Checkpoint 005: NVCF Connectivity Goal Clarified (2026-10-10)

**Stage:** Planning/discussion; implementation remains paused.

**Architect clarification:** Gridmetry owns no NVIDIA GPU. The first intended demonstration is interconnectivity through Exchange to NVIDIA-hosted Omniverse, not a physical power or workload-SLA validation. This narrower milestone does not require Batsim or a hardware trial.

**Evidence:** Rechecked the clone's explicit NVCF Kit application/session dependency and actual browser custom-message helpers. Consulted official DSX deployment and NVCF API documentation. Older NVCF links now resolve to self-managed documentation; distinguish that from permission to use NVIDIA's hosted function. Two generic streaming SDK links returned HTTP 403.

**Recommendation:** Use the existing authorized NVCF-hosted Omniverse session, local Exchange/Go adapter, and a supported browser SDK bridge for scene updates. Establish session connectivity before making it an implementation dependency. NVCF provides remote hosting/session access, not a new GPU power-control API. Any calculated DSX power remains explicitly modeled.

**Changes:** Added the clarified first objective, proposed connection flow, hosted-versus-self-managed distinction, required function/session access facts and connectivity acceptance criteria to [feasibility-assessment.md](feasibility-assessment.md). No new implementation or deployment code.

**Verification limits:** No key contents accessed; no NVCF session, function discovery, paid compute, deployment or live command invoked. Hosted availability and credentials remain unverified.

**Next:** Obtain the non-secret hosted connection instructions or application URL and establish an authorized session connectivity check before revising the detailed plan.

## Checkpoint 006: Live Read-Only NVCF Discovery (2026-10-10)

**Stage:** Implementation authorized for an access-first connectivity MVP; hosted scene integration not established.

**Architect request:** Establish access and proceed in an MVP way. Narrow the previous candidate sequence to safe access checking, local Exchange bridge, and verified hosted scene integration in [implementation-plan.md](implementation-plan.md).

**Actual requests:** HTTPS GET private/authorized function list returned HTTP 200 with zero functions. Public list returned HTTP 200 with 226 functions and one streaming candidate. GET details for that candidate with `includeSecrets=false` returned HTTP 200: `ai-digital-twins-for-fluid-simulation`, `STREAMING`, `DEGRADING`. It is not confirmed as DSX and must not be selected as the desired deployment merely because it streams.

**Credential handling:** With the architect's authorization, request processes loaded the credential from its local file without displaying it or passing its value as a terminal argument. TLS verification was enabled; redirects and environment proxies were disabled; response size was bounded and only allowlisted metadata/counts were displayed. The credential was not read into agent context. Earlier checkpoints refer to earlier non-use, not a permanent prohibition on runtime file loading.

**Constraints:** `/workspace/secretspec.toml` could not be consulted because it is absent in the current workspace. No credential values were exposed. No function invocation, token issuance, session allocation, deployment or mutation was performed.

**Conclusion:** NVCF discovery API access verified. Intended DSX function identity, invocation permission, hosted session health and scene connectivity remain unverified. Public streaming candidate status is degrading and its name identifies a different demonstration.

**Next:** Implement and test a small repeatable Go discovery CLI under Phase 1, then review and report. The actual DSX application URL or function/version identifiers remain required for Phase 3 if discovery cannot find them.

## Checkpoint 007: Access-Check MVP Implemented and Reviewed (2026-10-10)

**Stage:** Phase 1 of 3 complete and reviewed; awaiting architect continuation for Phase 2.

**Implementation:** Standard-library Go `dsx-access` CLI for fixed-host, read-only NVCF discovery and explicit version metadata checks. Runtime credential loading is bounded, output is sanitized, and candidate metadata is never called a connected DSX session. No MQTT/Compose/browser application or hosted invocation is implemented yet.

**Repairs and review:** Corrected function detail parsing to the verified `function` wrapper, validated output UUIDs/enums, made null/missing list structures errors, tightened bearer input, and separated discovery/metadata/activity from DSX/session verification. Review requested duplicate package-header and valid-enum fixes; both were addressed with focused validation. Final re-review returned `APPROVED` and editor diagnostics are clean.

**Validation:** Implementation reported passing `go test ./...`, `go test -race ./...`, `go vet ./...` and scoped formatting checks. Parent focused `go test ./internal/nvcf ./cmd/dsx-access` passed. The IDE test tool found no tests, so command-line Go testing was used.

**Live CLI results:** Discovery reproduced HTTP 200 / zero private-authorized functions and HTTP 200 / 226 public functions, with one unrelated degrading streaming function. Exit 2 correctly indicates unresolved DSX candidate. Explicit detail lookup returned matching metadata, inactive function status, and false DSX identity/session verification. No session or paid GPU workload started.

**Record:** See [dsx-connectivity-mvp-phase-1-complete.md](dsx-connectivity-mvp-phase-1-complete.md) for files, tests, review provenance, blocker and suggested commit message.

**Next:** Architect confirms continuation to the local Exchange/Go message bridge phase. For the hosted phase, obtain correct DSX connection details; current discovery does not identify that target. No commit or new Git branch was made.

## Checkpoint 008: Correct Public DSX Experience URL Found (2026-10-10)

**Stage:** Hosted URL research only; no Phase 2 implementation or headless session integration.

**Verified URL:** https://build.nvidia.com/nvidia/omniverse-dsx-blueprint-for-ai-factories

**Evidence:** NVIDIA's rendered blueprint catalog links to this exact page. Integrated-browser navigation confirmed the DSX page title, selected Experience tab and Site/GPU/Power configurator. A text fetcher returned a misleading 404, while its catalog extraction also surfaced an alternative publisher path that was not usable. Browser evidence resolved that discrepancy; do not label a search-index or static-fetch failure as proof the hosted demo is absent.

**Unexpected session behavior:** Opening the public page automatically displayed `Session Time Remaining: 9:52`. No key was supplied, login completed, terms button clicked or scene command sent. The browser was promptly navigated away to `about:blank` to end its page connection. Actual GPU allocation/billing and server-side session release were not inspected. Future visits must account for automatic timed-session behavior rather than being described as guaranteed allocation-free.

**Conclusion:** Public DSX browser Experience reachability is verified. Its headless streaming/API connection contract and mapping to an accessible NVCF function remain unverified. Earlier discovery results describe that credential's visible NVCF functions, not proof that this separate catalog experience is unavailable.

**Changes:** Recorded the verified URL and session caveat in [discovery.md](discovery.md) and this checkpoint. No application code changed.

**Next:** Give the architect the working page URL. Any subsequent network/session integration needs its own explicitly bounded verification, separate from this URL-finding task.

## Checkpoint 009: NVCF Configuration on the Hosted DSX Page (2026-10-11)

**Stage:** Read-only hosted-configuration research requested by the architect. No Phase 2 implementation or hosted-session invocation.

**Question:** Is NVCF present on the actual NVIDIA-hosted DSX experience, and can the supplied key access that target?

**New evidence:** HTTPS-fetched the public DSX page without executing JavaScript. Parsed its inline Next.js data using JSON decoding: `nvcfFunctionId` is `0ab96455-e029-4b9c-9ce5-de0fbd5ef436`. Inspected same-origin public script references; catalog asset `/_next/static/chunks/00jw9q5dp3jlz.js` passes `endpointSpec.nvcfFunctionId` and `kasPath: https://api.ngc.nvidia.com` into the Experience component. The public configuration is therefore tied to this hosted page, not inferred solely from the local clone.

**Key-based verification:** Authenticated HTTPS GET to `https://api.nvcf.nvidia.com/v2/nvcf/functions/0ab96455-e029-4b9c-9ce5-de0fbd5ef436/versions` returned HTTP 404. Credentials were loaded only by the runtime process, redirects/proxies disabled, TLS verified, response size bounded and response/error bodies suppressed. This result does not distinguish stale/removed identifiers, API routing or account visibility; it establishes no invocation or session permission.

**Advertised-base follow-up:** The same read-only versions route at `https://api.ngc.nvidia.com` returned HTTP 403 with this credential. Response bodies were suppressed. Direct target metadata access remains unestablished; separate invocation/session scopes were not tested. The bounded request processes finished, and no cloud session or background workload remained from these checks.

**Conclusion:** NVCF configuration is present in the hosted frontend. Operational backend verification and direct programmatic access with this key are still unestablished. A generic function-discovery list missing this ID does not invalidate the public-page evidence. No generic streaming SDK mode or undocumented invocation endpoint has been inferred from the field name alone.

**Safety:** This check did not load the Experience in a browser, run its JavaScript, start a timed session, invoke a function, issue tokens, send scene commands or modify a deployment. Public asset inspection exposed only selected fields, identifiers and service bases. Raw credential values never entered agent context.

**Changes:** Added the direct hosted-configuration evidence to [discovery.md](discovery.md) and this checkpoint. No application code or deployment configuration changed.

**Next:** Resolve the supported hosted session/authentication contract before integrating a client. Metadata lookup failure must not be worked around by guessing invocation paths or using an unrelated public function.

## Checkpoint 010: Local Message Bridge Implementation Started (2026-10-11)

**Stage:** Phase 2 of 3, implementation authorized by the architect's request for code, test cases and review.

**Local hypothesis:** A signed immediate one-feed DSX Flex command can pass through the Exchange-compatible event-bus subset and reach an authenticated browser-facing stream without any hosted GPU/session access.

**Discriminating check:** A Compose-backed scenario subscribes to the stream, publishes a signed MQTT command and verifies the matching normalized event. Invalid signatures, bad identities/units, duplicates, expired commands, unsupported schedules/strategies and unavailable connections must not be reported as delivered or executed.

**Scope:** MQTT 3.1.1 and ES256 lab signing, reference-schema validation plus explicit immediate-only policy, bounded ingress/event delivery, locally generated credential files and private/loopback Compose networking. No Omniverse key use, power simulator, invented DSX ACK, cloud session or upstream repo modification.

**Prerequisites checked:** `docker compose version` reports 2.39.4; Docker daemon server reports 28.5.1. The pinned reference contract requires `targets`, `interval` and `load_constraint`, with watt/kilowatt/megawatt units and `null` release.

**Next:** Implement test-first, run focused validation after edits, exercise the containerized scenario and failure cases, then review and checkpoint Phase 2 before proceeding further.

## Checkpoint 011: Local Bridge Reviewed and Real-Broker Verified (2026-10-11)

**Stage:** Phase 2 implementation reviewed `APPROVED`; containerized acceptance remains `BLOCKED`. Phase 3 not started.

**Implemented:** Signed/schema-validated immediate one-feed MQTT-to-authenticated-SSE bridge, cap/release policy, duplicate/expiry and bounded delivery handling, readiness/counters/CORS, isolated lab credentials, Compose/image and scenario runner. Details are in [dsx-connectivity-mvp-phase-2-complete.md](dsx-connectivity-mvp-phase-2-complete.md).

**Review/verification repairs:** Fixed policy-before-dedup order, delivery expiry, browser preflight, SUBACK coverage, shared NATS account routing, partial initialization, deterministic reconnect tests and bounded client backoff. Kept processing-time freshness checks as an explicit fail-closed policy. Removed duplicated package headers rather than dismissing editor syntax errors.

**Actual parent results:** Default unit tests, race checks and vet passed. Automated native integration against NATS 2.12.6 passed with race in 3.658 seconds, covering accepted cap/release, HTTP 401, invalid command rejection, readiness HTTP 503 on broker loss and HTTP 200 plus delivery after restart. Temporary manual test processes were stopped. Credentials were generated locally and never printed; the Omniverse key was not used.

**Docker gate:** Final Compose syntax validation and image build passed. Dedicated Compose acceptance failed before init execution with `unsupported shim version (3)`. Docker/containerd/shim version evidence is recorded in the phase report; shared runtime modification was not attempted. Native protocol success is not labelled container success.

**Next:** Operator resolves the local Docker execution prerequisite and reruns Compose acceptance. Architect reviews Phase 2 and confirms continuation; hosted SDK integration remains separately gated by access/connection details. No commit or new branch was made.

## Future Checkpoint Format

For each checkpoint record date, stage, scope, evidence, decisions, files changed, exact validation results, blockers, and next action. Use explicit `proposed`, `verified`, `approved`, and `blocked` labels. Keep secret values and sensitive endpoint details out of this log.
