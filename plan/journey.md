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

## Future Checkpoint Format

For each checkpoint record date, stage, scope, evidence, decisions, files changed, exact validation results, blockers, and next action. Use explicit `proposed`, `verified`, `approved`, and `blocked` labels. Keep secret values and sensitive endpoint details out of this log.
