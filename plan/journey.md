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

## Checkpoint 012: Current Source and Images Built (2026-10-11)

**Stage:** Build-only validation requested by the architect; no new phase implementation.

**Inputs:** Inspected the current module, Dockerfile, Compose configuration and Docker exclusions before building. Preserved recent source changes; no application code was edited.

**Actual results:** `go build ./...` passed. `docker compose config --quiet` passed. `docker compose --profile scenario build` passed and produced `dsx-adapter-lab-init`, `dsx-adapter-lab-bridge` and `dsx-adapter-lab-scenario` images.

**Limits:** No containers or NVIDIA sessions were started and no credentials were read. Tests were not rerun for this build-only request. The previously observed container-runtime blocker was not rechecked; successful image construction does not establish containerized acceptance.

**Next:** Run the local Compose acceptance when its runtime prerequisite is resolved. Hosted connection work remains gated separately.

## Checkpoint 013: Docker Acceptance Blocker Resolved (2026-10-11)

**Stage:** Phase 2 local container acceptance complete. Phase 3 hosted access remains gated.

**Recheck:** `docker run --rm dsx-adapter-lab-init help` succeeded. The daemon was already changed and running existing services before this task; the old shim failure could not be reproduced. No runtime configuration or daemon restart was performed.

**New failure and diagnosis:** Both Docker-backed acceptance tests then failed on allocated port 8090. The effective Compose JSON showed `integration/override.yaml` with `ports: []` did not remove base host bindings. Existing services were not stopped to work around the test defect.

**Approved fix:** User explicitly approved the test-only repair. Added `TestComposeOverrideClearsPublishedPorts` and confirmed it failed before the fix. Changed broker/bridge overrides to `ports: !reset []`; the regression passed. No application code or demo port configuration was changed.

**Actual results:** The three-test Docker suite passed in 22.610 seconds: signed command delivery (12.83s), merged-config port isolation (0.06s), broker outage/reconnect delivery (9.71s). Parent independently reran the isolation test (0.103s) and `go vet -tags integration ./integration`, both passing. Final scoped code review returned `APPROVED`.

**Safety:** Test-owned Compose containers/volumes were cleaned up. `on-site-b` and `site-proxy` remained running throughout; the latter remained healthy. No real key was read, no hosted session started and no unrelated service changed. Record correction: earlier test-isolation descriptions reflected intent, but host ports were only actually cleared after this reset-tag fix.

**Changes:** Updated the integration override/regression and current README, phase report and plan status. Historical failed checks remain recorded; their old blockers are not presented as current failures.

**Next:** Local Phase 2 is ready for architect review/commit. Obtain supported authorization/session details before proceeding to Phase 3; no hosted interoperability success is claimed. No commit was made.

## Checkpoint 014: Persistent Live Lab Staged (2026-10-12)

**Stage:** Architect requested a real test. Staged and exercised the implemented Phase 2 path; no hosted NVIDIA session or scene command was invoked.

**Setup:** Existing `site-proxy` occupies port 8090. Added [compose.stage.yaml](../compose.stage.yaml) to replace only the lab's published ports with loopback MQTT 21883 and HTTP 28090, using the isolated project `dsx-adapter-staged`. Checked port availability and asserted the effective merged Compose JSON has exactly those bindings. The normal Compose configuration and unrelated services were not changed.

**Start:** `docker compose -p dsx-adapter-staged -f compose.yaml -f compose.stage.yaml up -d --build --wait --wait-timeout 90 broker bridge` succeeded; initialization generated isolated credentials in the staging project's volume. No credential values were printed or read into agent context. The NVIDIA key was not used.

**Actual live results:**

- Signed `scenario --watts 800000` produced matching `command_received`, feed `main-a`, `constraint_watts: 800000`; ID `03049fa8-fa2e-42ba-8811-3635e4513973`.
- Signed `scenario --release` produced matching `release: true`; ID `a293ed46-63fb-4da6-8854-d0c4de6546cb`.
- Signed `scenario --watts 0 --timeout 2s` was rejected; final accepted count stayed 2 and rejected count increased to 1.
- Readiness returned HTTP 200; unauthenticated event-stream access returned HTTP 401.
- Final liveness counters: accepted 2, rejected 1, duplicate 0, queue_dropped 0, expired 0.
- `dsx-adapter-staged-bridge-1` is healthy and `dsx-adapter-staged-broker-1` is running. Existing `on-site-b` and healthy `site-proxy` remained running throughout.

**Files:** Added only the staging Compose override and updated README/test checkpoint instructions. Compose verified the custom `!override` tags; generic editor unresolved-tag diagnostics are not Compose validation failures. No application code changed.

**Left available:** The two staging services and their generated credentials/data volumes remain available intentionally. Health is at `http://127.0.0.1:28090/healthz`; authenticated stream access is exercised by the scenario container. README includes exact replay and project-scoped shutdown commands. `down` without `-v` preserves credentials/data.

**Limits:** This is a real local container/network test, not a physical power experiment or hosted Omniverse demonstration. A cap/release receipt is neither an execution ACK nor delivered-flexibility evidence. Phase 3 access remains unverified. No commit was made.

## Checkpoint 015: Hosted Omniverse UI Steering Confirmed (2026-10-12)

**Stage:** Manual hosted Omniverse check; no adapter-to-Omniverse integration yet.

**Action:** The architect logged into the NVIDIA Build DSX Experience and manually used the existing UI controls. The relevant cloned frontend path shows `Simulations -> Electrical -> Begin Test`, `Load %`, `Fail RPP(s)` and `EDP Setting` controls emitting `powerFailure` stream messages to Kit for RPP whip coloring. The thermal path similarly emits `setAttribute` for `inputs:load_level` when running the heat-load test.

**Result:** The architect reported `Scene/visualization changed`. This confirms the hosted Omniverse Experience can be steered through its own browser/WebRTC UI path.

**Boundary:** This is not DSX Exchange steering and not the Go bridge. No local MQTT command was connected to the browser session. No NVCF function was invoked directly, no API key was used and no hosted programmatic control contract was established. It is UI-to-Omniverse steering evidence only.

**Next:** Phase 3 should bridge the existing local SSE command stream to a supported browser/SDK client or other authorized hosted-control path. Until then, our local DSX bridge and the hosted UI steering are two proven pieces that are not yet connected.

## Checkpoint 016: Hosted UI Steering Retested (2026-10-12)

**Stage:** Manual hosted Omniverse retest; no adapter-to-hosted integration yet.

**Action:** The architect retested the logged-in DSX Experience using the built-in simulation controls. The thermal test produced a red room/heat visualization. The electrical test produced visible green and violet states, consistent with de-energized and overloaded RPP/whip coloring in the cloned `PowerFailureTest` UI semantics.

**Access attempt:** Opened the public DSX Experience URL in the host browser from this environment. Static HTTP access to the public page remains possible, but this agent cannot attach to or operate the architect's logged-in browser session/cookies directly. No API key was used, no scene commands were sent by the adapter, and no hosted API invocation occurred.

**Conclusion:** Hosted Omniverse UI steering is confirmed twice by manual observation. Programmatic steering from the local DSX bridge remains Phase 3 work.

## Checkpoint 017: Repository API Capabilities Clarified (2026-10-12)

**Stage:** Focused source review and clarification requested by the architect. No connector implementation or hosted invocation.

**Verified command interface:** `web/src/streamMessages.ts` calls NVIDIA's `AppStreamer.sendMessage` with `event_type: send_message_from_event`, `payload.command_name` and a JSON-encoded `payload.message`. The Kit manager parses that envelope and routes `setAttribute`, `powerFailure`, `changeVisibility`, `changeCamera`, `changeGpu` and `rppWhipVisibility`. Scene control is therefore programmatic; manually clicking the public page is not the only possible client implementation.

**Verified connection/deployment surfaces:** `web/src/lib/OmniverseApi.ts` connects through the SDK using `StreamType.DIRECT` in the clone. `scripts/create_function.sh` registers an NVCF `STREAMING` function with a default `/sign_in` entry point, port 49100, JWT-enabled streaming and a readiness route on port 8111. These are source/configuration facts, not an instruction to create another GPU deployment or proof of access to the existing public Experience.

**HTTP distinction:** The actual agent HTTP router handles chat, health, preferences and context-state synchronization. `POST /api/agent/state` updates agent context and returns status; it is not a deterministic GPU cap or independent solver/readback interface. The manager's inspected scene-command paths do not supply an application execution acknowledgement or measured-power sample.

**Shared discussion:** Re-read the supplied ChatGPT share. Its `GetState`, `GetFlexibilityEnvelope`, `ApplyPowerTarget`, `ApplyPowerLimit`, `ReleaseConstraint` and `GetResponse` examples are explicitly conceptual adapter interfaces. They are not documented NVIDIA DSX REST endpoints or evidence of a running model.

**Corrections:** Earlier metadata 404/403 responses establish only that the tested metadata lookups did not succeed; they do not prove invocation/session access is impossible. A logged-in NVIDIA webpage is one observed session route, not inherently required by every possible authorized SDK client. The outstanding issue is supported connection/authentication to the intended hosted session, not ignorance of the repo's scene-command format.

**Evidence limits:** Prior thermal/electrical colors are architect-reported observations, not agent-inspected transport traces or power measurements. The previously supplied DOM-console connector remains an unvalidated proposal; successful CORS preflight does not prove browser host-port reachability, content-security compatibility, React event handling or command-to-scene delivery. Do not label that connector as working.

**Recommended direction:** Map accepted DSX events to allowlisted SDK scene messages through a client whose session/authentication contract is verified. Prefer that interface over brittle DOM manipulation. `powerFailure` receives supplied presentation values; it does not itself predict GPU power or replicate MaxLPS. No key was read or used in this source review, and no NVIDIA deployment/session was changed.

## Checkpoint 018: Anonymous Demo and Official SDK Investigated (2026-10-12)

**Stage:** Anonymous startup/client research requested by the architect. No connector implementation or session allocation.

**Architect observation:** The public DSX Experience is usable without an account. Treat login as optional for that demo, not a prerequisite for steering its own UI. Anonymous page availability does not establish an anonymous REST scene-control service or permission to allocate arbitrary custom clients.

**Verified SDK distribution:** The repo's registry configuration and dependency lock identify NVIDIA's `https://edge.urm.nvidia.com/artifactory/api/npm/omniverse-client-npm/` registry. An unauthenticated GET of the referenced `@nvidia/omniverse-webrtc-streaming-library` 5.17.0 tarball returned HTTP 200. Inspected the archive in memory without extracting or running code. Authentication settings from registry files and all actual keys were suppressed/not exposed.

**Verified public type contract:** `AppStreamer.connect` accepts streaming props and custom messages are supported. SDK 5.17.0 defines `DIRECT`, `GFN` and `NVCF` stream types, not `CLOUD`; `NVCFConfig` aliases `DirectConfig`. Remote configuration includes `backendUrl`, `sessionId`, `signalingServer`, `signalingPath`, `signalingPort`, `signalingQuery`, optional `accessToken` and `authenticate`, plus connection callbacks. These fields demonstrate supported remote streaming, not a verified configuration for the public DSX deployment.

**Documentation caution:** The SDK README's NVCF example uses direct streaming and a `server` placeholder; it is not a session-bootstrap/authentication recipe. Its example typing also differs from the type declarations, so do not copy it into a connector without reconciling the actual SDK contract. Prior suggestions of a generic `CLOUD`/catalog configuration branch were unverified and must not be implemented as facts.

**Remaining boundary:** Identify the public Experience's anonymous session bootstrap and selected client configuration. Metadata 404/403 does not determine invocation permission. The hosted page advertises an NVCF function ID, but that alone is not the required connection tuple or a session credential issuer.

**Safety and evidence:** Only public HTML/package metadata and static assets were inspected. No SDK or hosted application JavaScript ran, no function was invoked, no tokens issued, no scene commands sent and no user API key used. Do not label SDK research as successful hosted connectivity.

## Checkpoint 019: Anonymous Bootstrap Boundary Narrowed (2026-10-12)

**Stage:** Phase 3 connectivity research remains blocked on the hosted session configuration, not on a requirement to log in. No connector implementation or hosted command delivery is complete.

**Static evidence:** A targeted anonymous HTML parse found generic page/state records, not a usable streaming configuration. A bounded Oracle review verified that Build's `00jw9q5dp3jlz.js` dispatcher selects the custom playground with `e.A(424122)` and passes `attributes`, `nvcfFunctionId` and generic `kasPath`. The runtime implements `A` by resolving a registered loader factory; the number is not a directly usable client URL. The generic route asset did not resolve that factory within the inspected excerpts. This is not an exhaustive absence claim or evidence that anonymous streaming is impossible.

**Browser diagnostic:** Installed isolated Playwright tooling and its ARM64 Chromium runtime under `/root/.cache`, outside adapter dependencies. A static-only probe denies API/session calls, WebSockets and WebRTC, uses a fresh anonymous context, and suppresses credentials. Its first pre-browser assertion caught an overly broad asset allowlist; restricted the allowlist to the exact public document and Build's static directory. The corrected probe and a separate launch-only check could not start Chromium: its normal runtime path lacks GLib and other shared libraries. An existing Nix GLib library was located, but a compatible complete browser runtime was not established. No browser page or streaming session opened. No system dependency packages or mounted environment files were changed.

**Validation:** `node /root/.cache/dsx-browser-research/probe.mjs` failed first at the safety assertion before browser launch, then at browser initialization after the allowlist correction. A standalone launch-only diagnostic identified `libglib-2.0.so.0`; `ldd` on the located browser executable reported 21 unresolved libraries. These are diagnostic failures, not evidence of NVIDIA access denial. No Go production files changed and no new adapter behavior was tested.

**Required diagnostic:** From the architect's existing working anonymous Network trace, obtain the first non-static request that supplies streaming/session configuration: method, sanitized hostname/path, HTTP status, response field names and initiator JavaScript asset hostname/path. Exclude headers, cookies, query values, credential values and full HAR exports; redact session IDs in paths. Inspect its public initiator asset before considering a supported connector or authorized hosted test. Do not replay the bootstrap or infer arbitrary session-creation permission from anonymous UI access.

## Checkpoint 020: Documented User Profile Access Checked (2026-10-12)

**Stage and authorization:** The architect requested a read-only check of NVIDIA's documented user-profile endpoint using the existing local API key while investigating the anonymous demo trace. This does not authorize session allocation, deployments, profile changes or other mutating operations.

**Verified contract:** `https://docs.api.nvidia.com/ngc/reference/getuserprofile` specifies `GET https://api.ngc.nvidia.com/v2/user-profile`. Its response schema contains `userProfile` and `requestStatus`. The operation declares `security: [{"ssa": ["getUserProfile"]}]`, but the published components omit the corresponding security-scheme definition. Do not infer an OAuth issuer, credential type or token-exchange scope from the name alone. The linked authentication page contains placeholder text rather than a usable authentication recipe.

**Executed check:** Ran a bounded Node standard-library probe from terminal input, loading the local key inside the process without exposing it to agent context or command arguments. Compared one unauthenticated GET with one API-key Bearer GET to the fixed documented HTTPS origin/path. TLS verification remained enabled; redirects were not followed; each request had a 15-second deadline and a 1 MiB response limit. Only HTTP status, allowlisted request-status enums and recognized field names were emitted; no personal values, credentials or raw errors were printed or saved.

**Captured results:** The unauthenticated control returned HTTP 401, `requestStatus.statusCode: UNAUTHORIZED`, and only `requestStatus`. The API-key Bearer request returned HTTP 500, `requestStatus.statusCode: INTERNAL_ERROR`, and only `requestStatus`. Neither returned `userProfile`. This independently confirms the previously reported Bearer 500 and adds a discriminating control. Different handling with an Authorization header is not proof that the key is valid for this route, and a 500 is not proof that the key is invalid.

**Authentication review:** A bounded Oracle review checked the official profile reference, authentication page, NGC user guide and Private Registry user guide. General personal/service-key support, NIM Public API Endpoints permissions, registry `$oauthtoken` login and registry bearer-token practices do not establish authentication for this user-profile route. No route-specific API-key token exchange was found in those checked sources; this is a documentation gap, not an exhaustive claim about NVIDIA's capabilities.

**Outcome and next boundary:** The documented endpoint is reachable, but profile access using the tested Bearer-key flow remains unverified and produced no additional profile or streaming information. Do not retry indefinitely or guess a registry-token exchange for profile access. A complete definition of `securitySchemes.ssa` and NVIDIA's supported credential/acquisition flow would resolve the profile-authentication question. The anonymous demo's sanitized session-bootstrap trace remains the direct next evidence for the Omniverse connector. No Go production files changed; no key was rotated and no hosted compute session or scene command was started.

## Checkpoint 021: NVIDIA Forum Authentication Examples Researched (2026-10-12)

**Stage:** Architect-requested public forum research on using an existing API key. No credential use, token exchange or hosted invocation during this research.

**Search evidence:** Read the supplied phone-verification thread and queried the public Discourse JSON search index for `"authn.nvidia.com/token"`, `"user-profile"` and `"NGC" "Bearer"`. These searches returned HTTP 200 with 18, 1 and 38 topic records respectively; these are returned records, not exhaustive forum totals. The sole `user-profile` match concerned the Windows Control Panel, not the NGC API. Search excerpts were redacted before output. A separate bounded Oracle pass searched streaming-specific discussions. A subsequent terminal thread read did not expose captured results; conclusions below rely on explicit webpage results and the Oracle report rather than that uncaptured output.

**Key issuance is separate:** `https://forums.developer.nvidia.com/t/363923` reports phone-verification rate limits during NIM key issuance. It does not provide an authentication solution for an existing key, NGC user profiles or DSX streaming. Do not apply its account-verification complaints as evidence of our key's validity or permissions.

**Concrete historical key-to-JWT example:** `https://forums.developer.nvidia.com/t/login-issue-jwt-on-tao-api-with-jupyter/257773` (2023) shows TAO code performing `GET https://authn.nvidia.com/token` with `Accept: application/json` and `Authorization: ApiKey <API_KEY>`, then consuming the response's `token` field. A community reply reports HTTP 200 and an NGC-audience JWT. Replies explain a server-side token change that broke TAO's audience validation. This establishes a reported historical NGC token-exchange pattern, not verified compatibility with the current Build key or `/v2/user-profile`. The thread's local NodePort/arbitrary-token workaround, disabled TLS verification and printed credentials must not be copied into our connector or applied to hosted services.

**Scoped artifact-download example:** `https://forums.developer.nvidia.com/t/aerial-installation-failure/343133` (2025) proposes requesting an NGC token from the same issuer using HTTP Basic authentication, literal username `$oauthtoken`, the API key as password, `service=ngc` and an Aerial-specific group scope. The reply describes a temporary AODT fix, not a DSX streaming bootstrap. The reporting user's follow-up still failed, so the thread is not evidence of a successful end-to-end remedy. Do not copy another organization's scope, email safe-link wrapper or shell commands that expose keys through arguments/logs.

**Streaming-specific findings:** The Oracle pass found `https://forums.developer.nvidia.com/t/342276/2`, where NVIDIA employee/moderator VickNV distinguishes AppStreamer targeting OKAS/NVCF signaling/session management from plain Isaac Sim streaming. It supplies no hosted DSX session issuer or API-key recipe. Community thread `https://forums.developer.nvidia.com/t/383953/1` reports bearer-key authorization plus `function-id` metadata for AI inference at `grpc.nvcf.nvidia.com:443`; audio/animation-data streaming is not a rendered Omniverse WebRTC session. `https://forums.developer.nvidia.com/t/368331` reports a USD Search Bearer-key 403 and discusses account permissions; that advice is not evidence that the anonymous DSX demo requires an enterprise account.

**Decision and next discriminating check:** Distinguish raw Bearer-key inference, an API-key-to-JWT authentication exchange and SDK session authorization rather than treating all NVIDIA keys/transports as interchangeable. Forum evidence supplies a candidate fixed-origin NGC token exchange that could be tested separately with authorization, reporting only status and token-field presence and keeping any token in memory. It does not justify requesting guessed profile scopes, reusing an unverified JWT for DSX or allocating a hosted session. Profile acceptance and the DSX bootstrap remain separate checks; no forum source inspected establishes either for our key.

**Safety and verification:** Only public forum searches/pages were accessed. No API key, session token or personal profile value is recorded here; no production files, account permissions or running services changed. Forum examples are dated reports, not validated current integration behavior.

## Checkpoint 022: Forum API-Key-to-Token Flow Tested (2026-10-12)

**Authorization and hypothesis:** The architect requested trying the sourced token-exchange example. Tested whether `GET https://authn.nvidia.com/token` with `Accept: application/json` and `Authorization: ApiKey <API_KEY>` would issue a usable token for the existing local key. A successful issuance would allow a separate read-only `/v2/user-profile` test with the returned token; no compute-session creation or function invocation was authorized or attempted.

**Execution:** Ran a Node standard-library probe through `node --input-type=module -e`, loading the key from the local file only inside the process. The key was bounded and format-checked without printing it or passing its value through process arguments. The request used the exact forum origin/path without guessed service/group scopes, redirects, retries or other authentication variants. TLS verification remained enabled, with a 15-second request deadline and a 1 MiB response limit. Output was restricted to HTTP status, allowlisted request-status enums and token-presence/format booleans; raw bodies, headers and credentials were suppressed.

**Captured result:** The token issuer returned HTTP 401 with a non-JSON response. No token was obtained: `tokenFieldPresent: false`, `tokenUsableAsBearer: false`, `jwtShaped: false`. The conditional profile check was explicitly skipped. The probe completed and emitted its safety summary; this is a captured rejection, not a missing-output or network-timeout result.

**Decision and limits:** The historical `Authorization: ApiKey` flow did not work with this key in this test. This does not establish that the key is invalid for NIM, NVCF or other supported services, and does not rule out separately documented authentication mechanisms. No JWT was issued or reused, no profile information was returned and no hosted session was started. Do not repeat the rejected flow indefinitely, copy another organization's scope or treat an untested Basic-authentication variant as proven. The hosted DSX session bootstrap remains unverified.

## Checkpoint 023: Other NVIDIA Services Checked Without Inference (2026-10-12)

**Authorization and scope:** The architect requested trying other services with the existing key. Limited the checks to Cloud Functions discovery, an NGC authentication variant and hosted model-list metadata. No inference, uploads, downloads of model artifacts, account changes or compute-session creation were performed. The staged Compose configuration and running services were not changed.

**NVCF discovery:** Ran `/root/sdk/go1.25.11/bin/go -C /workspace/dsx-adapter run ./cmd/dsx-access --key-file /workspace/dsx-adapter/omniverse.key`, reusing the existing bounded read-only client. Both private/authorized and public function-list requests returned HTTP 200. The private/authorized list contained zero functions; the public list contained 226. The report identified one streaming candidate whose name did not match DSX/Omniverse and whose API status was `DEGRADING`. DSX identity, target metadata and session access remained unverified; the CLI printed `exit status 2` despite the successful list responses. No function was invoked.

**Discriminating authentication control:** A separate unauthenticated GET to the same NVCF private/authorized listing returned HTTP 401. In combination with the authenticated HTTP 200, this confirms that the key is accepted for this discovery operation, rather than merely accompanying a public request. It does not establish invocation permission, access to every function or connectivity to the hosted DSX scene.

**NGC Basic authentication variant:** Tested `GET https://authn.nvidia.com/token?service=ngc` using HTTP Basic with literal username `$oauthtoken` and the key as password, encoded only inside the process. This follows the sourced authentication pattern without copying another organization's group scope. The issuer returned HTTP 401 with a non-JSON response and no token. The conditional profile request was not made. Together with Checkpoint 022, both tested token-exchange schemes failed; that does not invalidate the successful NVCF Bearer-key discovery result.

**Verified model-list route:** A guessed documentation reference resolved to an unrelated model page and a GitHub source URL returned 404, so neither was treated as endpoint evidence. Instead, inspected the published `langchain-nvidia-ai-endpoints` 1.4.3 wheel from PyPI in memory, checking its SHA-256 against package metadata without installing or executing it. Its `_common.py` defaults to `https://integrate.api.nvidia.com/v1`, lists models at `{base_url}/models`, and uses a GET in `available_models`. Package metadata identifies the source repository as `https://github.com/langchain-ai/langchain-nvidia`.

**Hosted model metadata result:** Compared `GET https://integrate.api.nvidia.com/v1/models` without credentials and with `Authorization: Bearer <API_KEY>`. Both returned HTTP 200 and 80 model records. This verifies accessible model metadata, not key validity, inference authorization or DSX streaming access. Only status, response-shape presence and counts were emitted; no model was invoked.

**Execution safeguards and conclusion:** All live probes used fixed NVIDIA origins, verified TLS, refused redirects, bounded deadlines and response sizes, and suppressed raw bodies/errors and credentials. Keys and any potential tokens were confined to runtime memory. Six service requests were made in this checkpoint, including unauthenticated controls; no retries or paid operations were performed. No Go production files changed. Use direct Bearer-key authorization for the confirmed NVCF discovery path; do not require an NGC JWT exchange for that operation or generalize its success to hosted DSX session access. The missing hosted session-bootstrap contract remains a separate prerequisite.

## Checkpoint 024: Hosted DSX Signaling Route Supplied (2026-10-12)

**Architect evidence:** The architect supplied a WebSocket URL from the working browser demo. Record its sanitized form only: `wss://api.ngc.nvidia.com/v2/predict/websocket/blueprints/omniverse-dsx-blueprint-for-ai-factories/sign_in?peer_id=<generated>&version=2`. The actual peer identifier is not retained here. This identifies a blueprint-specific hosted signaling gateway, not a power-control REST endpoint or the NVCF metadata URL used in discovery.

**Verified SDK match:** Inspected the previously identified public 5.17.0 SDK archive in memory without installing or executing it. Its signaling implementation constructs a URL from server address/port and optional resource path, then appends `/sign_in?peer_id=` with the client name and `&version=2`. It creates a browser WebSocket with a subprotocol list, and nearby code optionally includes session-identifying subprotocols. This matches the reported URL shape, but does not establish that the hosted page uses exactly this SDK version or that an independent client is authorized.

**Partial configuration, not a working connection recipe:** The observed route suggests `signalingServer: api.ngc.nvidia.com`, `signalingPort: 443` and base `signalingPath: /v2/predict/websocket/blueprints/omniverse-dsx-blueprint-for-ai-factories`. The SDK appends `/sign_in` itself; do not put that suffix into the base path or hard-code the architect's peer identifier. Stream mode, session configuration, media routing and authentication still require confirmation. The SDK declares `accessToken` as a client JWT, and `authenticate` adds an `Authorization-Bearer.<access token>` WebSocket subprotocol. Do not assume the successfully tested NVCF API key is interchangeable with this credential.

**Documentation limits:** A bounded Oracle search did not verify an official contract for this exact hosted gateway. Candidate-page/source lookups failed and a search page required JavaScript; these retrieval failures do not invalidate the architect's working-browser observation or prove that guidance is absent. A legacy function-invocation documentation URL resolved to current self-managed NVCF guidance, which is not evidence for this hosted route. A second narrow SDK inspection did not expose captured terminal results and is not counted as an offline URL-check success.

**Next discriminating browser evidence:** From this existing WebSocket entry, request the HTTP status, presence of request/response `Sec-WebSocket-Protocol`, authorization scheme only if an Authorization header exists, and whether a Cookie header exists. Share header names and redacted protocol prefixes only; do not share full values, tokens or cookies. If needed next, inspect message direction and event/field names only, never full SDP, ICE credentials or TURN configuration. An HTTP 101 verifies upgrade, not receipt of rendered frames or scene-command delivery.

**Safety and phase status:** No request was made to the supplied predict/WebSocket endpoint, no API key was used and no session or scene command was started. The gateway route is now browser-reported evidence; authenticated external-client connectivity and the SDK datachannel remain unverified. Production code and staged Compose configuration were not changed.

## Checkpoint 025: Browser Handshake Confirms Credential-Free Upgrade (2026-10-12)

**Architect evidence:** The architect supplied request and response headers for the already observed blueprint-specific signaling connection. The response is HTTP 101 Switching Protocols. The supplied request has `Origin: https://build.nvidia.com` and WebSocket upgrade headers, but no Authorization, Cookie or Sec-WebSocket-Protocol header. The supplied response likewise has no Sec-WebSocket-Protocol header. Treat this as credential-free HTTP upgrade evidence provided the header capture is complete; do not infer that later signaling messages cannot carry credentials or that arbitrary clients are authorized.

**Offline validation:** Used Node's standard cryptographic API to compare the supplied Sec-WebSocket-Accept value against Base64(SHA-1(request nonce + RFC 6455 GUID)). The captured output was `matches: true`, `networkRequests: 0`. This checks consistency of the handshake pair, not an independent live NVIDIA connection. The WebSocket request nonce is protocol handshake material, not an API key. Neither the nonce, peer identifier nor request ID is copied into this checkpoint.

**Origin and transport interpretation:** The response advertises Access-Control-Allow-Origin for `https://build.nvidia.com` and Access-Control-Allow-Credentials. These response fields do not prove cookies were sent, nor do they establish whether other origins are permitted. Browser WebSocket origin checks are distinct from Fetch/XHR CORS enforcement. Do not forge the Build Origin header, conclude localhost will work from these headers alone, or weaken browser/server security controls. HTTP 101 does not establish WebRTC media, datachannel readiness or scene-command execution.

**Updated hypothesis:** API-key/token exchange is not demonstrated as a prerequisite for the captured initial signaling upgrade. The partial SDK configuration from Checkpoint 024 now has evidence of no HTTP/subprotocol credential on this specific connection. Keep later message authentication, session/media configuration, supported custom-client access and hosted SDK/version compatibility unverified rather than adding a key to a guessed header.

**Next evidence and gated test:** Inspect the first few messages in the architect's existing WebSocket capture for direction and top-level/nested event or field names only; do not request full SDP, ICE passwords, TURN credentials, token values or raw frame dumps. This can distinguish immediate WebRTC negotiation from a later session/authentication exchange. Then prepare the official SDK client and, with explicit approval for any new hosted-session allocation, test from its real local origin with a generated peer ID. Success requires received rendered frames and a ready application-message channel, followed by a separately authorized reversible scene command. The Go Exchange bridge remains independent of these browser/session checks.

**Safety and phase status:** No NVIDIA requests, API-key use, new session allocation or scene changes occurred in this checkpoint. Only the documentation changed. The supplied working-browser signaling handshake is verified offline for internal consistency; adapter-to-hosted scene connectivity is still not complete.

## Checkpoint 026: ICE Signaling Frames Match SDK Protocol (2026-10-12)

**Architect evidence:** The existing browser capture includes `ackid` and `peer_msg` envelopes, with `from`, `to` and a serialized `msg`. The nested message contains `sdpMLineIndex`, `sdpMid` and an ICE `candidate` with UDP transport and `typ host`. Corresponding `ack` values match the envelope identifiers. No addresses, port values, peer IDs, timestamps or candidate strings are retained here. These frames demonstrate that the observed signaling connection progressed into ICE candidate exchange, not that ICE connectivity, rendered video or application control completed.

**Captured source verification:** Inspected the public SDK 5.17.0 archive in memory using a bounded `python3 -c` probe. Its `package/dist/omniverse-webrtc-streaming-library.js` dispatches `peer_msg.from` and `peer_msg.msg` to a peer-message handler, tracks `ackid`, and removes buffered signaling messages whose identifiers are covered by an acknowledgement. These are signaling reliability acknowledgements, not correlated acknowledgements of DSX commands or scene execution. The inspected excerpts also contain browser `addIceCandidate` compatibility logic; the complete RTC candidate handler was not traced or executed. The separate Oracle package fetch failed and is not counted as source-verification evidence.

**Owning integration surface:** The reference `web/src/lib/OmniverseApi.ts` delegates connection establishment to `AppStreamer.connect` and application messages to `AppStreamer.sendMessage`. Reuse the official streaming SDK for signaling/RTC rather than reimplementing raw `peer_msg`, hard-coding observed addresses/ports or sending scene commands directly into the signaling WebSocket. Its generic application request IDs are distinct from signaling `ackid`; a request wrapper alone does not supply a server-side scene execution acknowledgement.

**Updated boundary:** The supplied blueprint gateway, credential-free HTTP upgrade and ICE signaling are sufficient to prepare an offline-tested SDK connector with the observed host/base path and a generated peer ID. They do not prove acceptance of a local browser Origin, the hosted SDK version, complete session/media configuration, datachannel readiness or execution of a command from Exchange. Do not request additional raw SDP, ICE or TURN secrets simply for reassurance.

**Next gate:** A bounded live connection from a client using its real local Origin is the discriminating check. Obtain explicit approval before any new hosted-session allocation; do not forge Build's Origin or inject an unverified API key. First require actual rendered frames and an application-message-ready connection; authorize a reversible scene command separately before testing Exchange-to-scene behavior. Treat origin rejection or signaling failure as a result, not permission to bypass controls.

**Safety and status:** Only public static SDK source and the sibling reference wrapper were read; the SDK was not installed or executed. No NVIDIA signaling endpoint was contacted, API key used, hosted session started or scene changed. No production files or Compose settings changed. The remaining task is an authorized SDK connection test, not additional NGC token-exchange retries.

## Checkpoint 027: Authorized Single SDK Connection Attempt Completed (2026-10-12)

**Authorization:** The architect explicitly approved one live connection of at most 60 seconds from the real localhost Origin, without an API key, retries or scene changes. That single live-test grant is consumed. Any additional hosted attempt requires fresh authorization; no scene-command authorization was granted.

**Implementation and prerequisites:** Added `plan/hosted-connection-probe.mjs` as an experimental standalone diagnostic, not a production Exchange connector. Installed Chromium runtime dependencies inside this container with the Playwright dependency installer; no read-only Nix mounts, Docker services or staged Compose settings were changed. Installed official SDK 5.17.0 and Playwright in `/root/.cache/dsx-browser-research`, outside the Go module, with npm package scripts disabled. SDK code is served unmodified on a random loopback port. No credential file is opened by the probe.

**Safety controls:** Only the observed DSX host/base path, HTTPS port and generated peer-ID/version query are allowed. Browser WebSockets are limited to one forwarded connection, with no subprotocol credentials, remote HTTP requests or Origin spoofing. CSP denies workers and frames. SDK reconnects are configured to zero; a separate socket guard prevents internal retry forwarding regardless of SDK behavior. All outgoing RTC datachannel payloads are intercepted and counted without forwarding, preventing application/input commands. The probe owns the Chromium server process and has a 60-second hard-kill timer, awaited cleanup and measured process exit. It never requests backend application termination or claims that client exit proves backend instance termination.

**Focused validation:** The initial `node plan/hosted-connection-probe.mjs --self-test` failed at the intentionally unimplemented route gate. After implementation, self-tests passed. Actual-SDK `--offline` runs verified the generated gateway URL while forwarding zero sockets. A pre-live Code-Review pass found worker escape coverage, misplaced Playwright timeout arguments, unmeasured outgoing sends, unconditional cleanup reporting and stale telemetry; these were repaired immediately. The repeated offline check passed, and re-review returned `APPROVED`. Passive handshake diagnostics were added and validated offline before live execution. No hosted request was needed for these repairs.

**Single live command and captured result:** Executed `node /workspace/dsx-adapter/plan/hosted-connection-probe.mjs --live` once. Captured `socketForwarded: 1`, `networkSocketCount: 1`, `localOriginVerified: true`, and no Authorization, Cookie or Sec-WebSocket-Protocol header. The socket did not open successfully: `socketOpened: false`, `socketError: true`, `socketClosed: true`, `handshakeStatus: 0`. There were no ICE/description frames, rendered video frames, created/open datachannels or outgoing datachannel payloads. The SDK tried a second socket construction despite its zero-reconnect setting; the separate guard denied it, leaving only one actual network attempt. Cleanup reported `clientClosed: true`, `cleanupFailed: false`, `deadlineKilledBrowser: false`, `clientLifetimeMs: 1553`, `apiKeyUsed: false`.

**Reporting correction:** The original run's outcome label was `socket_policy_rejected`, because the later blocked retry masked the initial failure. Added a regression check and changed outcome precedence to `connection_failed_before_upgrade` for this state. This is a reporting correction verified using synthetic assertions and offline runs, not a second live attempt. Added a fixed-label parser for future Chromium WebSocket errors that emits only allowlisted error classes or an HTTP rejection status, never raw diagnostic strings. It cannot recover the first run's uncaptured failure reason.

**Non-session control:** Ran a single standard Node TLS handshake to the NGC hostname with verified certificates and no HTTP or WebSocket request. It succeeded with TLS 1.3 and reported zero HTTP/WebSocket requests and no key use. This establishes basic Node DNS/TCP/TLS reachability, not Chromium's certificate behavior, successful gateway upgrade, localhost-Origin permission or compatible hosted SDK configuration. Do not attribute the failed live attempt to Origin policy or key validity without evidence.

**Final offline result:** `node /workspace/dsx-adapter/plan/hosted-connection-probe.mjs --offline` passed after the reporting changes: expected route, one SDK socket construction, zero forwarded/network sockets, zero outgoing datachannel payloads, no page errors, verified browser exit and no cleanup failure; final client lifetime was 668 ms. No screenshot was produced because the live acceptance criteria of rendered nonblank frames plus an open datachannel were not met. Go production code was not changed or retested in this slice.

**Outcome and next boundary:** The official SDK can construct the observed route and the single-attempt safeguards work, but adapter-to-hosted connectivity remains blocked before upgrade. The remaining discriminating evidence is Chromium's sanitized handshake failure class/status from a newly authorized test, or NVIDIA's supported external-client/origin guidance. Do not rerun automatically, forge Origin, inject keys into guessed fields or send scene commands. The local Exchange bridge remains usable and untouched; Phase 3 acceptance is not complete.

## Checkpoint 028: Browser Context Compared Before Forum Escalation (2026-10-12)

**Architect direction:** Investigate why the working anonymous browser connection succeeds while the probe fails before proposing a forum escalation. No forum post was published, new markdown file created, credential used or additional streaming connection attempted in this investigation. Inspected the current probe files first to preserve intervening edits.

**Concrete context differences:** The working capture has the genuine Build Origin in the architect's Mac browser. The probe uses a genuine random loopback Origin in container Chromium and blocks every nonlocal HTTP request before signaling. Browser, network/IP and preceding page activity therefore differ as well as Origin. An uncredentialed upgrade does not mean the request has no security context, and HTTP 403 alone identifies none of these differences as the cause. The earlier reported retry's raw rejection response was not retained in the current diagnostic fields; no hidden rejection reason was recovered from local saved-output searches.

**Public startup inspection with sessions disabled:** The existing static-page probe now ran successfully after browser dependencies were installed. All WebSockets, RTC connections and API/session requests remained blocked. It observed an AWS WAF challenge-script load and an unauthenticated blueprint-spec request in the real page startup; neither is present in the minimal local SDK probe. These are possible context differences, not evidence that a WAF token or prior spec request is required for signaling. A separate read-only GET of the exact observed blueprint spec endpoint returned HTTP 200 without credentials.

**Verified hosted-client mismatch:** Public Build asset `0r6l7vo9rrq13.js` configures the DSX wrapper with `nvcf: true`, server `api.ngc.nvidia.com`, port 443 and `/v2/predict/websocket/blueprints/<blueprint-id>`. The actual hosted component in `1vpuzpk80-j-6.js` selects NVCF versus DIRECT from that flag and supplies `authenticate: false` with no access token. Our probe had used DIRECT, following the sibling repository rather than this hosted build. Added a focused failing assertion, then changed the selector to NVCF. The actual-SDK offline check passed with the expected route, one socket construction, zero forwarded sockets or datachannel payloads, and verified browser cleanup in 656 ms. This repairs client compatibility but is not proof that stream mode caused the earlier HTTP 403 or that the corrected client can connect.

**Read-only Origin control:** Compared browser-generated OPTIONS requests from genuine localhost and Build pages using the same Chromium/network, with scripts, cookies, credentials, WebSocket upgrades and all other remote requests blocked. The captured CORS preflight response was HTTP 204 with the matching Access-Control-Allow-Origin and credentials permission for both origins. The subsequent fetch did not complete successfully, so do not advertise this as a successful signaling HTTP call. This does not prove WebSocket Origin acceptance, but it does not support treating the problem as a blanket localhost Fetch-CORS denial either. The first diagnostic stopped at a local trailing-slash URL-allowlist mismatch; the corrected diagnostic produced these results without any session allocation.

**Conclusion and next discriminating test:** The route targets the intended DSX blueprint, a real hosted/probe stream-mode mismatch is repaired, and key authentication is not demonstrated as a prerequisite for the captured handshake. Exact WebSocket rejection policy remains unresolved: Origin-specific upgrade handling, browser/network policy or other request context may still matter. The clean next live comparison is the corrected SDK client in the architect's normal Mac browser/network, preserving its real local Origin and one-attempt/no-command limits. Any new hosted connection still requires explicit authorization. Do not spoof Origin, solve/evade WAF challenges, rotate keys without evidence or assert that external clients are unsupported based on the incomplete comparison.

## Checkpoint 029: Shared Mac Browser Scene Verified Read-Only (2026-10-12)

**Architect request:** Inspect the shared VS Code browser page on the architect's Mac to investigate the failed independent connector and assess reusing the page for control. This request authorizes inspection, not scene actuation. No new session was opened or page reloaded during this check.

**Verified existing scene:** Browser runtime reports `MacIntel`, the genuine Build Origin, and a playing `remote-video` stream at 1920 by 1080 with readyState 4. Decoded frames increased from 10,393 to 16,799 during inspection. A screenshot showed the DSX facility scene; a read-only video pixel sample reported brightness range 247 and 13,636 non-dark pixels out of 14,400 sampled pixels. This is direct evidence of an active, nonblank scene in the shared browser, not success of the container's SDK probe or a physical power measurement.

**Available controls and interface limit:** The page exposes Thermal/Electrical tabs, a Load-percent slider and Begin Test. Browser tools can operate these existing UI controls through the page's established connection. No `AppStreamer`, `OmniverseAPI`, `OmniverseApi`, `dsxBridge`, `dsxInspector` or `OVDSX` window global was found in the targeted check. This does not prove that no internal interface exists, but prevents claiming a verified direct SDK-call interface on window. UI automation would be a demonstrator path, not a stable headless simulator API.

**Diagnostic evidence and gaps:** Retained resource timing metadata shows successful unauthenticated blueprint-spec and catalog requests in this browser. The full earlier handshake/closed container error response cannot be reconstructed from these resource entries, and this working tab does not by itself identify the separate container connection's 403 cause. Use sanitized current page/session state and permitted future observations rather than reading cookies, account storage, tokens, SDP or ICE/TURN secrets.

**Exchange integration boundary:** The existing staged Compose override allows the Build Origin for the bridge, but this is not proof of host port forwarding, private-network browser permission, CSP compatibility or authenticated event-stream connectivity. Reusing this page could carry accepted local events into allowlisted UI controls without opening another WebRTC connection. That bridge is not implemented or validated yet, and scene actuation still requires explicit approval and an agreed mapping. Do not equate the demo's percentage controls with measured watt delivery or inject RPP failures as a power-target mapping.

**Safety and outcome:** Only page snapshots, video/DOM metadata, retained public request metadata, a screenshot and a local pixel sample were read. No cookies/keys, private storage or raw signaling payloads were collected; no slider, radio, test button or camera was operated. Existing UI settings were preserved. The shared browser's rendered DSX scene is verified; command execution through it and Exchange-to-scene delivery remain unverified.

## Checkpoint 030: Reversible Electrical Load-Reduction Scenario Executed (2026-10-13)

**Authorization and scope:** The architect requested simulating the previously described healthy electrical scenario, reducing Load from 80% to 60%. Used the existing shared Mac-browser session and ordinary UI controls, without opening another session, injecting credentials, calling a hidden SDK interface or connecting Exchange events.

**Captured baseline:** The page initially showed Configurator/Site and an active video stream. Opened Simulations/Electrical and recorded stopped state, Load 0%, failed RPP count 0 and EDP 1.5. These actual current values differ from the unexecuted tests proposed on the previous day. Kept failed RPPs at zero and EDP unchanged throughout the scenario.

**Executed scenario and results:** Set the native Load control to 80% through its React-compatible input/change events, verified the visible 80% label, and clicked Begin Test. The power row displayed 306 kW on each of four RPPs, totaling 1,224 kW; compute-service feeder loading was 52%. Reduced only Load to 60%, verified the visible label and changed power row, and captured 230 kW per RPP, totaling 920 kW; feeder loading was 39%. Both states reported Running flawlessly. The displayed reduction was 304 kW, or 24.84% of the rounded displayed baseline. Screenshots captured the 80% controls and the 60% power table.

**Restoration verified:** Clicked Stop Test, restored Load 0%, and checked that stopped state, failed RPP count 0 and EDP 1.5 exactly matched the captured baseline. Returned to Configurator and explicitly reselected Site after the panel reopened on GPU. The final page snapshot verified Site selected. Removed the temporary in-page baseline/result variables; no persistent browser storage was used. Decoded video frames advanced from 27,666 before the scenario to 33,359 by restoration.

**Evidence limits and outcome:** This is a successfully executed UI-driven lower-load scenario with observed displayed electrical values and continued streamed video. The power values are frontend/demo calculations, not measured cluster consumption. No correlated remote scene execution acknowledgement or independent GPU/workload response was captured, and changes in whip colors were not separately verified. The result does not prove a controller can achieve a requested kW cap, workload/SLA behavior, MaxLPS operation or signed Exchange-to-scene delivery. Unlike the prior attempts caught in terminal-marker loops, this checkpoint records actual browser control operations and verified restoration.

## Checkpoint 031: Visible Repeat Paused at 80% (2026-10-13)

**Authorization:** The architect requested watching the scenario in stages. The previous public session had expired; obtained explicit approval to click Start a new session and show 80% before waiting for the architect's cue to reduce Load to 60%.

**Execution and captured state:** Started one new session through the page's normal UI and verified decoded video frames before opening Simulations/Electrical. Captured fresh stopped defaults: Load 0%, failed RPP count 0, EDP 1.5. Set Load to 80%, verified its visible label, clicked Begin Test and scrolled the power row into view. The panel displays 306 kW per RPP, totaling 1,224 kW, with 52% feeder loading and Running flawlessly status. Failed RPP count and EDP were unchanged; no keys or hidden SDK interfaces were used.

**Intentional pause:** Leave the Electrical panel and 80% scenario running for the architect to inspect. The 60% step and restoration have not been performed in this repeat. Temporary `__dsxVisibleCurtailment` page state retains the baseline and observed 80% result for the next step; it contains UI values only, not credentials or persistent storage. Wait for the architect before reducing the load, then restore the captured stopped defaults when requested. These figures remain demo-calculated power, not measured cluster demand or Exchange-driven execution.

## Checkpoint 032: Visible Reduction to 60% Paused for Inspection (2026-10-13)

**Architect cue and operation:** After the architect requested continuing the visible scenario, verified the same session was active at 80%, playing, with failed RPP count 0 and EDP 1.5. Reduced only the Load control to 60%, verified the visible label and updated power row, and kept the table in view. No session restart, credential use, direct SDK call or Exchange command was involved.

**Observed result:** The panel displays 230 kW per RPP, totaling 920 kW, compared with the captured 1,224 kW at 80%. The displayed reduction is 304 kW, or 24.84% of the rounded baseline. Feeder loading is 39% rather than 52%, and status remains Running flawlessly. The video continues playing. These are demo-calculated electrical values, not physical cluster measurements or independent curtailment verification.

**Intentional viewing state:** Leave the Electrical scenario running at 60% for the architect to inspect. Temporary `__dsxVisibleCurtailment` state now retains stage `reduced60`, the original stopped defaults and both observations. Restoration to stopped, 0% load, zero failed RPPs and EDP 1.5 is pending the architect's cue; do not claim that this visible repeat has already been restored.

## Checkpoint 033: Visible Scenario Restored (2026-10-13)

**Architect request and execution:** On the architect's restore cue, used the saved baseline from the visible repeat, clicked Stop Test and restored Load 0%. Verified failed RPP count 0 and EDP 1.5 against that saved baseline. The browser check returned `matches: true`; Begin Test is visible, status and electrical table values are cleared, and the scenario is not playing.

**Final state and cleanup:** Left the Electrical panel visible in the same active browser session and removed `__dsxVisibleCurtailment`. No new session, credentials, direct SDK operation, Exchange event or production-file change was needed. The visible 80%-to-60% demonstration and its requested restoration are now complete; its power values remain calculated visualization inputs rather than measured cluster power or proof of controller-enforced curtailment.

## Checkpoint 034: Saved Lab Token Authentication Verified (2026-10-13)

**Architect request and execution:** Verify the token saved locally in `lab.token`. A bounded Node process opened that file with an 8 KiB read limit, validated its format and used it only in memory for fixed local GET requests to `http://127.0.0.1:28090/events`. No token value, credential fingerprint, raw headers or event payload was printed or recorded. Every stream was closed after its result or a five-second deadline.

**Captured controls and success:** The file was present and its token format valid. A request without a token returned HTTP 401. A request with an altered token returned HTTP 401. The saved token returned HTTP 200 with the expected event-stream content type and the `: connected` marker. This verifies that the saved token authenticates a client to the Go adapter's read-only event stream.

**Scope and limits:** This is not direct MQTT authentication to the Exchange-compatible NATS broker; publisher/subscriber MQTT credentials and signed DSX Flex payloads remain separate. It also does not verify the browser receiver's live UI readiness, NVIDIA session access or target execution. No MQTT command or NVIDIA request was made and no scene was changed. A previous generic browser receiver failure is not evidence that this token is invalid.

## Checkpoint 035: Automatic Signed Target and Release Verified End to End (2026-10-13)

**Scope:** Reused the architect's authenticated page-side receiver and existing NVIDIA scene. Commands passed through the owned Exchange-compatible NATS/MQTT lab, signature/policy validation, Go visualization mapper and authenticated SSE, then the installed receiver operated the Electrical controls. No per-command manual slider changes or hidden SDK calls were used, and no credential values were inspected.

**First target and timeout:** Command `e1b51143-f477-4d46-9ba7-01dbb5ab9a17` carried a 920,000 W target. The receiver reported that exact ID as target_applied, and the UI displayed 60% with four 230 kW RPP values, totaling 920 kW. Release `7c5ea4ba-cebd-41e4-be48-86eca948ab15` was verified in the adapter, but browser delivery was not observed: the receiver reached its two-minute limit with only one applied event and restored the original stopped 0% settings. This release is not counted as an end-to-end pass.

**Completed second pair:** With explicit approval for a quick target/release repeat, replaced the idle widget using the fingerprint-verified deployed script, and the architect entered the local token directly. Verified receiver state receiving, authenticated true, HTTP 200 and an 80% baseline. Published target `8a3d45d5-01cb-43d8-a90e-2c19d7765b75` at 07:34:09 UTC through `dsx-lab scenario --watts 920000`. The adapter and browser reported the same ID; the receiver applied count was one and the UI displayed 60%, 230 kW per RPP and 920 kW total.

**Release verification:** Published signed null release `0e00c0d8-f82d-4eca-b9af-6634298a46db` at 07:34:39 UTC through `dsx-lab scenario --release`. The adapter mapped it to the configured unconstrained 80% baseline. The receiver reported that exact release ID, state release_applied and two applied commands; the observed UI was 80%, 306 kW per RPP and 1,224 kW total. This verifies the release pathway before receiver timeout.

**Cleanup verified:** Disconnected the receiver after observing release. Its report showed restored true and connected false. The actual Electrical UI was stopped, Load 0%, zero failed RPPs and EDP 1.5. The stream's resulting ERR_ABORTED event was expected client teardown, not a failed test. The existing page and staged services were left available.

**Outcome and limits:** Automatic signed-command-to-hosted-UI interoperability and release-to-baseline both passed. The deployed receiver script was pinned to SHA-256 `3290b6a7488ec13bc033ba16181574871c3356688863896be6e08568bf91169b`. The values are demo-calculated visualization inputs, not measured cluster demand, workload response or physical curtailment. No correlated remote Kit execution acknowledgement or independent whip-color change was captured. This is an installed, session-bound UI POC, not a persistent standalone NVIDIA client or the full upstream Exchange stack. No production source changed or unit tests were rerun in this live verification slice.

## Checkpoint 036: Receiver Lifetime Configurable, Default Eight Minutes (2026-10-13)

**Architect request and implementation:** Replace the hard-coded two-minute receiver lifetime with a configurable duration defaulting to eight minutes. Added a Duration (minutes) numeric input to the Connect form and a `mount({ durationMinutes })` override; `mount()` defaults to 8. Whole-minute values from 1 to 10 are validated before mount and before authentication. The selected duration is reported publicly and its timer starts when Connect is submitted. This does not alter or extend NVIDIA's own session timer.

**Interval semantics and cleanup:** Removed the independent two-minute cap on signed command expiry. Command intervals now use their actual end time, bounded to the browser timer's supported range; receiver deadline/manual disconnect still clear that timer and restore the captured UI baseline. Authentication, freshness, no-reconnect behavior and expiry-generation ownership are unchanged.

**Focused validation:** Four new regressions failed before implementation: default eight-minute deadline/restoration, mount/form override, invalid durations and an interval beyond two minutes. All twelve Node receiver tests passed after implementation. Go tests for presentation, transport and the lab command passed, editor diagnostics were clean, and the focused lifetime review returned APPROVED. Deadline behavior used deterministic fake timers, not an eight-minute hosted test.

**Deployment and visible verification:** Rebuilt only the owned staged bridge after confirming the browser receiver was disconnected. The served receiver SHA-256 is `667e34ff4b4925e1cc4614232951c874a939f1f809b0fb001a0da64d767276e9`. Replaced the idle shared-page widget with this verified source. Browser inspection returned durationMinutes 8, Duration field value 8, minimum 1, maximum 10, state idle and connected false. No token was entered/read, NVIDIA session restarted, target published or scene test started; the existing NVIDIA session remained expired.

## Checkpoint 037: Persistent Interactive Flex Publisher Implemented (2026-10-13)

**Architect request and implementation:** Replace repeated container starts with a dynamic terminal publisher for multiple power targets during the browser receiver window. Added `scenario --interactive` to the existing lab binary, reusing its mounted credentials, ES256 signing and receipt schema. One MQTT connection and one authenticated SSE reader serve the entire process. Each numeric target or release gets a new command ID, traceparent, timestamp and signature; the next prompt follows its matching adapter receipt.

**Interface and safety:** Accept positive finite watt values up to 1,000,000, plus release, help and quit/exit. Invalid input is bounded and not echoed or published. Interactive duration defaults to 8m and is capped at 10m; command publish/receipt timeout defaults to 10s and is capped at 1m. Duration is independent of the widget and NVIDIA timers. EOF, quit, Ctrl+C and session timeout close connections without implicitly releasing a constraint; release is an explicit command. MQTT or SSE loss stops the session, with no automatic reconnect or publishing after loss of receipt observation.

**Focused tests and review:** Parser/loop and shared-reader regressions failed before their implementations, then passed. The first review identified idle SSE closure allowing a later publish and parent deadlines being collapsed into generic send errors; added regressions and repaired both. All default Go unit tests, the changed command package's race tests and vet passed. Focused re-review returned APPROVED; editor diagnostics were clean. Existing one-shot flags and receipt/visualization output remain supported.

**Real-broker acceptance:** Captured the final tagged Compose test output in a temporary generated log after earlier terminal results showed only test startup. `TestScenarioAcceptsOneSignedCommand` passed in 10.45s and `TestInteractiveScenarioAcceptsMultipleSignedCommands` passed in 5.33s; total 15.786s. The interactive test used one process with 920000, an invalid entry, 650000, 800000, release and quit. It verified one MQTT/SSE setup, four distinct matching receipts, correct target values and null release, and no echo of invalid input. Tests used isolated project names, internal networks and test-owned volumes with cleanup; no staged or NVIDIA command was issued.

**Delivery and run command:** Built the staged scenario image without starting it or restarting the staged broker/bridge. From the project directory, run `docker compose -p dsx-adapter-staged -f compose.yaml -f compose.stage.yaml run --rm --no-deps scenario --interactive` and type commands at `flex>`. README documents the interactive and scripted forms, duration override and cleanup semantics. The publisher confirms adapter delivery, not browser execution; keep the widget receiving and verify its applied state for live UI evidence. No hosted session was allocated and no physical-power claim is made.

## Checkpoint 038: Token-Free Console Loader Saved (2026-10-13)

**Architect request and artifact:** Saved the previously supplied injection snippet as `plan/inject-browser-receiver.js`. It validates the target page, fetches the public receiver source with credentials omitted and a five-second deadline, checks the current reviewed SHA-256 fingerprint, then disconnects/removes an older widget and mounts a new idle receiver. It contains no token and does not authenticate, start a session or publish commands.

**Validation:** Confirmed the served receiver fingerprint is `667e34ff4b4925e1cc4614232951c874a939f1f809b0fb001a0da64d767276e9`. Node syntax checking and offline mocked fetch/DOM checks passed: valid source mounts the eight-minute default; fingerprint mismatch preserves the old widget; wrong-page execution is rejected before fetching. No live browser or credential operation was performed. README links the loader and explains that reviewed source changes require updating the fingerprint.

## Checkpoint 039: User Profile Restarted With Full-Load Targets (2026-10-13)

**Architect request and preserved settings:** The architect changed the staged profile to EDP 1.2, retained an 80% baseline, and requested a Compose restart plus interactive targets up to 100%. Preserved both settings, all loopback bindings and persistent secret/data volumes. Did not start a NVIDIA session, operate the hosted scene or modify unrelated projects.

**Implementation:** Targets now map over 0-100% independently of startup/release baseline; release still restores the configured baseline. The adapter generates a browser client with its actual feed, EDP, baseline and capacity, replacing the pinned 1.5/80 restriction. Browser startup, validation, release, expiry and EDP readiness use that profile. Publisher input accepts a validated `DSX_MAX_WATTS`/`--max-watts` ceiling, retaining the default lab's 1,000,000 W limit. One staged YAML anchor sets bridge and publisher to 1,316,000 W, matching EDP 1.2's full displayed load.

**Validation:** Mapper, browser and publisher regressions failed before their implementations. All default Go tests, changed-package race tests, vet and fourteen Node receiver tests passed; focused review returned APPROVED. The isolated real-broker staged test accepted signed 1,316,000 W as 100% / 1,316 kW and release as 80% / 1,052 kW, passing in 11.854 seconds with test-owned cleanup. Generated served-client validation and saved-loader fingerprint checks also passed offline. The one-shot publisher remains bridge-policy-protected, but only interactive input performs the new pre-publish ceiling validation.

**Restart and delivery:** Rebuilt/recreated only `dsx-adapter-staged` broker/bridge without dependencies and rebuilt its scenario image. Services became healthy and `/readyz` returned ready. The generated EDP 1.2 client fingerprint is `5c49c53bcb94dda54f0ffba0cc305ebaeda78ee932043461f004a7a750875fc8`; updated only the saved loader's fingerprint. A live staged help/quit-only publisher check advertised 1,316,000 W and issued no target. Existing page-memory widgets must be re-injected and set to matching EDP before live use. No full-load NVIDIA UI test was run, and all power evidence remains visualization-only.

## Checkpoint 040: Live Electrical Readiness Mismatch Corrected (2026-10-13)

**Diagnosis:** The architect reported Simulation not ready after Connect. Read only the public receiver report and scene/profile metadata: authentication succeeded with HTTP 200, video was playing at readyState 4, Electrical was selected and failed RPP count was zero. The receiver expected the configured EDP 1.2, but the page had EDP 1.5 selected. This directly identifies the failing readiness predicate; the token and session expiry were not the cause in this capture.

**Authorized correction and verification:** After explicit approval, selected EDP 1.2 and replaced the failed widget using the verified deployed client fingerprint. The resulting browser report was idle/disconnected, expected EDP 1.2, baseline 80 and duration 8; checked radios were rpp-0 and edp-1.2, video remained ready/playing and Load was 0%. No token was read or entered, command published, NVIDIA session started or production source modified. The architect must enter the same lab token in the new widget to reconnect; the readiness guard was preserved rather than bypassed.

## Checkpoint 041: Flex Receiver Adapts to Manual EDP and Fault Changes (2026-10-13)

**Diagnosis and approved behavior:** Code inspection identified that the earlier healthy/configured-EDP guard threw simulation_not_ready and disconnected SSE on subsequent Flex updates. The currently shared page had no receiver/ready video, so the reported earlier live failure was not independently captured. The architect explicitly approved adapting to current EDP/failure selections and keeping Flex connected, rather than silently resetting faults or bypassing receipt validation.

**Implementation:** Retained the accepted watt constraint and added a bounded-lifecycle UI monitor. Recompute an integer load using the current EDP capacity and healthy RPP count, including each RPP's rounding, without altering radio selections. Receipt validation remains against the authenticated adapter profile; observed UI power is reported separately. Unique applied IDs and local adaptations have separate counters. All-four-RPP failure and temporary UI unavailability pause application without closing SSE; the newest valid pending target resumes when possible. Backend publisher/bridge limits, token handling and receiver duration are unchanged.

**Expiry and cleanup:** The first focused review found a lost expiry-baseline retry when UI/video was unavailable. Added a failing regression and an independent pending baseline-restore operation; it retries after UI recovery and is canceled by a newer command or disconnect. Generation guards still protect newer targets. Stop clears monitor/interval/lifetime timers and restores load/running state while preserving manual EDP/failure selections. Expired unapplied targets are not replayed.

**Validation and delivery:** Nineteen offline Node tests passed, covering EDP/fault remapping and later commands, conservative rounding, all-failed recovery, transient video loss, interval-expiry recovery, newer-target ownership, release, duration, authentication and restoration. Go presentation/transport/lab tests and focused vet passed; re-review returned APPROVED. Rebuilt only the owned staged bridge, verified ready and served fingerprint `0d51b5bd1edb8330954c5ade951653976ddaf1e8c2465554a31a3631743809c4`, and updated only the loader's pin. No NVIDIA session or fault was created, no live target was published, and the changed fault-adaptation behavior was not tested on the hosted scene. Re-inject the current loader to replace an older widget before live use.

## Checkpoint 042: Independent Operator How-To Added (2026-10-13)

**Architect request and deliverable:** Consolidate a guide so other developers can run the POC without this chat. Added How To Run The POC near the README start, with prerequisites, project/container context, idempotent credential initialization, staged startup/readiness, private browser-side port forwarding, correct local-token retrieval, NVIDIA scene selection, saved loader injection, widget connection, repeated signed targets, explicit release, restoration, volume-preserving shutdown and troubleshooting. Clarified that this browser path needs no local GPU or NVIDIA API key, while NVIDIA access/terms/session restrictions still apply.

**Validation and limits:** Checked all eight guide sections, local links, merged staged profile and equal 1,316,000 W limits, private port 28090 binding, and saved-loader match to the currently served script. Markdown diagnostics were clean. This was documentation-only: no secrets were read, services restarted, command published or NVIDIA session created. The walkthrough assembles previously verified operations; a separate fresh-machine operator rehearsal was not performed.

## Future Checkpoint Format

For each checkpoint record date, stage, scope, evidence, decisions, files changed, exact validation results, blockers, and next action. Use explicit `proposed`, `verified`, `approved`, and `blocked` labels. Keep secret values and sensitive endpoint details out of this log.
