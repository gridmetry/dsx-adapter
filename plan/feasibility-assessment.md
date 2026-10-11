# Gridmetry Flexibility Demonstration: Feasibility Reassessment

**Date:** 2026-10-10. **Status:** Evidence-backed recommendation for discussion, not implementation approval. The existing candidate plan remains paused. Previously accepted scope remains one feed and a single-site Exchange event-bus Compose subset; this assessment does not authorize workload-simulator integration or hosted deployment changes.

## Recommendation

**Architect clarification:** The first objective is interconnectivity with NVIDIA-hosted Omniverse without owning a GPU. See the NVCF section below: this is a valid narrower milestone and does not require a workload simulator or hardware power test. Those remain optional later evidence stages, not prerequisites for the connectivity demonstration.

We can build a credible simulated Gridmetry flexibility demonstration using DSX Exchange, with Omniverse as an optional connected facility view. We cannot currently claim that the supplied Omniverse deployment executes a GPU-power flexibility experiment merely by receiving bridged messages.

The architecture should be centered on Gridmetry's decision and evidence loop, not on the viewer:

Grid request -> Gridmetry policy -> DSX Exchange -> simulated power/workload controller -> independently evolving plant/workload state -> DSX status and response evidence -> Gridmetry.

Omniverse receives a projection of the experiment state through an authorized supported client or gateway. It should not be the source of delivered-power evidence unless a deployed solver/surrogate interface actually supplies that evidence.

The one-feed bounded model in the candidate plan is a useful first plumbing test. By itself, it does not demonstrate that AI work can be shifted or slowed while protecting business constraints. Making the latter claim needs workload modeling or hardware, an explicit change to the proposed scope.

## What Can Be Demonstrated

| Claim | Feasibility | Evidence Needed |
| --- | --- | --- |
| Gridmetry sends DSX Flex requests and receives status | Feasible with adapter and local test bus; not implemented yet | Signed wire round trips, scheduling, correlated status, reconnect/failure tests |
| A simulated feed responds to a cap and recovers | Feasible with an explicit plant model | Sampled model state, baseline, response trajectory, feasibility limits and failures |
| Omniverse displays the same experiment | Conditional on authorized client/session integration | Correct scene mapping, observable display result and presentation failure detection |
| Gridmetry chooses workload interventions while respecting modeled priorities/deadlines | Feasible with a separate workload simulator; not provided by this clone | Work arrivals, progress, throughput/power model, baseline comparison, deadline and backlog outcomes |
| NVIDIA's hosted simulator predicts facility/GPU power from our interventions | Not established | Actual deployed simulator API, command semantics, telemetry, calibration scope and validation |
| Real GPU power falls and useful AI work continues | Requires authorized hardware experiment | Measured GPU/node power, completed work, workload service metrics, baseline and repeat trials |
| The asset qualifies for FCR/mFRR or settlement | Not established by any of these demos | Applicable market product requirements, measurement boundary, performance tests and qualification process |

These are different milestones. The word "digital twin" does not make them interchangeable.

## What the Omniverse Clone Actually Provides

The following evidence uses the previously recorded reference snapshot in [discovery.md](discovery.md). It does not establish what extra components NVIDIA may have deployed privately.

1. The [client message helpers](../../omniverse-dsx-blueprint-for-ai-factories/web/src/streamMessages.ts#L60) send custom commands such as `setAttribute` and `powerFailure` over the stream. The latter carries power values supplied by the browser; it does not ask a GPU or solver to determine them.
2. The [Kit manager](../../omniverse-dsx-blueprint-for-ai-factories/source/extensions/manager/manager/extension.py#L137) routes these commands to scene operations. The inspected attribute/power-color paths do not emit a correlated execution result or power sample.
3. The [attribute helper](../../omniverse-dsx-blueprint-for-ai-factories/source/extensions/manager/manager/attribute.py#L8) writes an existing USD attribute. Its boolean success result is not returned by that manager path. The generic frontend request-ID wrapper does not create missing backend acknowledgement handlers.
4. The [whip-color code](../../omniverse-dsx-blueprint-for-ai-factories/source/extensions/manager/manager/whip_color.py#L40) maps supplied loading to colors. It is presentation logic, not an electrical load-flow solution.
5. The [electrical panel](../../omniverse-dsx-blueprint-for-ai-factories/web/src/components/panels/simulation/PowerFailureTest.tsx#L121) computes displayed power/current in the browser from a percentage, example capacity, RPP failure count and randomized voltage. These inputs are [React state](../../omniverse-dsx-blueprint-for-ai-factories/web/src/context/SimulationContext.tsx#L129), not automatically updated by USD attribute changes.
6. The [thermal slider](../../omniverse-dsx-blueprint-for-ai-factories/web/src/components/panels/simulation/SimulationPanel.tsx#L95) writes a volume shader load parameter, not an absolute-watt GPU cap.
7. No MQTT subscriber or operational GPU power-feedback connection was found in the inspected own-source controls. A stock hosted viewer must not be assumed to subscribe to our local event bus.

A useful disconfirming check, once session use is authorized, is to send known power-color values while leaving the electrical UI inputs fixed. If only colors change, this remains a presentation path. A documented solver-generated power stream, responding to interventions, would change the conclusion. No hosted test was performed.

The [official system architecture](https://docs.omniverse.nvidia.com/dsx/latest/system-architecture.html) describes simulation services, a Simulation Data Delegate and surrogate inference APIs. This is a promising intended integration architecture, not proof that those services or their endpoints are in the clone or available in the existing deployment. We should ask NVIDIA about those specific components rather than dismiss the wider DSX Sim ecosystem as visualization-only.

## What a Credible Gridmetry Experiment Needs

The demo should answer: "How much load can this asset reduce, for how long, at what workload cost, and how does it recover?" A cap setting or successful publish does not answer those questions.

Keep the controller separate from the simulated plant. Plant power and work progress must evolve from modeled actuator effects and workload state, not simply copy the requested target. Include idle/fixed demand, response lag, limits, natural workload variation and an actuator-failure case. Otherwise the controller creates its own evidence of success.

For a workload-flexibility claim, add independent remaining-work and priority/deadline state. The controller chooses allowed interventions; the simulator determines progress, backlog and power. Resource, checkpoint and restart costs must be modeled if those interventions are claimed. A single electrical feed can aggregate several workload classes without adding a multi-feed topology, but this is still extra modeling scope requiring approval.

### Illustrative Acceptance Scenario

The following numbers are assumptions for a test, not measured NVIDIA capacities or performance:

- Replay the same workload arrivals, initial state and random seed in baseline and controlled runs.
- Suppose the baseline is a stable 1 MW for the event. Request 200 kW reduction for 15 minutes. That corresponds to an 800 kW cap only under that baseline assumption; with changing baseline or insufficient controllability, the same cap does not guarantee 200 kW delivered reduction.
- Let Gridmetry select which eligible workload is slowed or deferred. Protect a declared critical-workload service threshold; do not assume protection without modeling or measuring it.
- Observe sampled modeled power, transition error, sustained reduction, curtailed energy, work progress, missed deadlines, backlog, and recovery/rebound.
- Repeat with insufficient flexibility or a failed actuator. The system must report the limitation or delivery failure instead of declaring compliance by target echo.
- Disconnect Omniverse. A presentation-only failure must not change the experiment result. Disconnect the authoritative backend and explicitly mark the evidence unavailable.

For a perfectly sustained 200 kW reduction lasting 0.25 hours, the event-window reduction is 50 kWh. Ramp time can reduce that amount. This is energy avoided in the event window, not necessarily net energy saved: deferred work can consume energy after release. Compare the full recovery horizon and completed useful work, not only the trough in the power chart.

Specify whether power is GPU, node, IT feed or utility import. CPU/storage/network overhead and cooling mean a GPU cap cannot be equated to a grid-meter reduction. Static PUE is at most an explicit approximation, not a calibrated transient facility model.

Start with a scheduled demand-response demonstration. Price optimization additionally needs prices and completion deadlines. Frequency-response qualification needs its own frequency/control dynamics, sampling, sustained response and relevant product requirements; an hourly or 15-minute cap demonstration is not that evidence.

## Alternatives and Their Fit

| Approach | Best Use | Main Limitation | Recommendation |
| --- | --- | --- | --- |
| Bounded one-feed plant + Exchange | Messaging, interval/control recovery, failure testing | No workload progress or business-constraint evidence by itself | Keep as the first offline plumbing milestone |
| Batsim/SimGrid + explicit power/performance model + Exchange | Work scheduling, resource allocation, completion delay and flexibility-policy comparisons | Requires modeled GPU curves, checkpoint assumptions and simulation-clock integration | Strongest software-only candidate for a workload-flexibility experiment |
| Documented NVIDIA simulator/surrogate connected to Exchange | Facility what-if response within the supplied model's scope | Deployment/API access and model coverage are not established | Pursue if NVIDIA supplies the real delegate/surrogate service |
| Small authorized GPU testbed + NVML/DCGM + Exchange | Real power response and workload performance | Limited scale, host permissions and measurement boundary; not MaxLPS replication | Strongest next evidence step when a controlled GPU becomes available |
| DSX Air logical twin | Authentic supported software/network/API integration | Not automatically a GPU electrical/power simulator | Useful for ecosystem integration, not a replacement for the power model |
| Recorded Emerald traces + Exchange | Replay, regression, realistic telemetry examples | Recorded samples do not react causally to new commands | Supplementary tests, never closed-loop validation |

### Batsim/SimGrid Nuances

[Batsim](https://batsim.readthedocs.io/en/latest/) is designed to test resource-management decisions independently of platform execution. Its [protocol](https://batsim.readthedocs.io/en/latest/protocol.html) describes execution, kill/progress, dynamic job registration and host power-state changes. Do not assume a turnkey GPU pause/checkpoint/resume interface: representing resumed remaining work and checkpoint overhead requires supported profile/registration behavior or additional modeling. Pin the simulator/protocol version; current and older documentation use different event formats.

The [SimGrid host-energy plugin](https://simgrid.org/doc/latest/Plugins.html#host-energy) supplies configurable host power profiles and pstate performance. It is not a ready-made model of GB200/GB300 tensor-core throughput or rack electrical transients. NVIDIA or measured workload-specific curves are needed to make those predictions credible. Separate CPU-style model assumptions from GPU characterization.

Batsim uses a synchronous decision protocol and [simulation time](https://batsim.readthedocs.io/en/latest/time-dilation.html), not automatically a real-time MQTT control clock. A coordinator must inject grid events and align times; accelerated runs cannot establish real service-response latency. This is a reason to integrate it behind the Go adapter rather than treating it as an MQTT appliance.

### Small Hardware Trial Without MaxLPS

We do not need to reproduce MaxLPS to run a bounded physical proof of concept. A supported NVIDIA GPU running useful test work can be observed through [NVML/nvidia-smi](https://docs.nvidia.com/deploy/nvidia-smi/index.html) or [DCGM](https://docs.nvidia.com/datacenter/dcgm/latest/user-guide/feature-overview.html), while an authorized host controller applies supported limits or changes workload admission/concurrency.

Power-limit management requires supported hardware and appropriate administrative permissions; ordinary shared GPU rental, vGPU or MIG access must not be assumed to permit it. NVML is preferable to fragile CLI-output parsing for maintained integration. DCGM supplies telemetry/configuration capabilities but does not itself supply Gridmetry's workload optimizer. Collect useful-work metrics as well as watts.

A small physical trial shows causal response at that device/workload boundary. It cannot prove megawatt-scale delivery, rack cooling dynamics or whole-factory performance by scaling the plotted axis.

### DSX Air Is a Different Tool

NVIDIA's [DSX Air guide](https://docs.nvidia.com/dsx-air/) describes a logical IT/network simulation environment. Its [October 7 article](https://developer.nvidia.com/blog/validate-ai-factory-changes-with-digital-twins-and-ai-agents/) explicitly distinguishes infrastructure/API validation from separate power/performance models, and discusses connecting external Brev GPU compute. Access to that compute does not establish permission to set GPU power limits. This is a useful NVIDIA discussion path, not a verified available testbed.

## NVCF for the Connectivity Milestone

**Clarified goal (2026-10-10):** Gridmetry owns no NVIDIA GPU and wants to demonstrate interconnectivity with the already NVIDIA-hosted Omniverse experience. NVCF is a credible route for that goal, conditional on access to the deployed function/session. No workload simulator is required merely to prove connected message flow and visible scene updates.

The clone explicitly supplies [an NVCF application configuration](../../omniverse-dsx-blueprint-for-ai-factories/source/apps/dsx_nvcf.kit#L11), including `omni.services.livestream.session`. NVIDIA's [DSX deployment documentation](https://docs.omniverse.nvidia.com/dsx/latest/application-streaming.html) also identifies a cloud streaming application, although its application names differ from the pinned clone. Use the deployed build's actual names/configuration rather than treating current documentation as an exact checkout match.

Separate NVIDIA-hosted NVCF access from self-managed NVCF installation. Hosted access uses the environment owner's remote compute; no local GPU is required. Self-managed NVCF still needs an underlying GPU compute plane and is not a solution to missing hardware by itself. Hosted availability, capacity, quota and cost must be confirmed, not assumed from the existence of an API key.

**Proposed connectivity flow:** Local Exchange MQTT -> Go adapter -> authenticated adapter-to-client event channel -> authorized browser/SDK client -> NVCF-hosted Omniverse session -> existing Kit scene handlers. Return client/session health and display evidence separately from the simulated DSX agent's calculated status. The adapter-to-client channel is proposed new code, not present in the stock viewer.

A browser SDK client is the conservative initial transport choice because the clone already implements browser streaming/custom messages. Do not promise a headless Go or JS sidecar until the hosted SDK/session contract supports it. The NVCF lifecycle/invocation API establishes function access; it does not automatically expose Kit's internal agent HTTP server or turn streaming scene commands into ordinary inference requests. A deployed custom HTTP gateway would be a different, explicitly approved path.

For an existing hosted session, we need its supported connection instructions and authorization. If allocating a session through NVCF, obtain the correct deployment/API base, function/version identifiers where required, account permission for that function, and the streaming SDK/session setup. Invocation, function discovery and deployment permissions are distinct; do not assume an invoke-only key can list or modify functions. The [current NVCF API overview](https://docs.nvidia.com/nvcf/overview/api) describes scope-based APIs but is framed for self-managed deployments, so its credential issuance examples must not be used as NVIDIA-hosted onboarding instructions.

**Demonstration acceptance:** An actual signed DSX command is received locally; its identifier and unit-aware mapping are recorded; an authorized client sends a supported scene command; the remote scene visibly changes; session/client disconnects and send failures are reported. Remote execution/readback must be described according to the deployed handler's real capability: a successful client send is not a Kit execution acknowledgement. A live view or screenshot can evidence presentation, not power delivery. Any published `calculated_load` must remain explicitly modeled, not inferred from rendered colors.

**Recommendation:** Verify authorized connection to the existing hosted NVCF/Omniverse session first, then bridge the agreed one-feed command/status workflow to that viewer. Do not require Gridmetry to deploy its own Omniverse GPU container, expose its local MQTT broker publicly, or install Batsim for this milestone. Backend-mediated delivery to the client can keep MQTT credentials and NVIDIA API secrets out of browser code. Session tokens may still be needed by the supported browser SDK and must be handled appropriately.

No NVCF function, session or key permission has been tested. Two generic streaming SDK documentation URLs returned HTTP 403 during this research; the exact hosted connection contract remains unresolved. No credential contents were read, and no paid or mutating cloud action was taken.

## Recommended Decision Before the Plan

Keep Go and the accepted Exchange Compose subset. Keep the adapter independent of its backend. Do not make the hosted Omniverse connector a prerequisite for proving Gridmetry's core flexibility logic.

Select the desired first claim explicitly:

- **Protocol/control proof:** stay within the accepted bounded one-feed model. Add Omniverse presentation only when the session/scene integration is supported.
- **Workload flexibility proof:** propose a separate workload-simulation backend, preferably assess Batsim/SimGrid rather than inventing a general scheduler simulator. Approve the additional model scope before coding.
- **Physical proof:** request a small supported GPU test environment with explicit control and telemetry permissions; run the same Gridmetry/DSX pathway against it later.

The most convincing longer-term demonstration combines workload/power response evidence and an optional Omniverse view. The view explains the facility context; the baseline, power and workload traces substantiate the flexibility claim.

### Precise NVIDIA Questions

1. Does the hosted experience expose the Simulation Data Delegate or a power/electrical surrogate API? What interventions, state variables, units, time base and outputs does it support?
2. Is there a workload/throughput model, or only facility response to an externally supplied load? Those support different demonstrations.
3. Can we connect an authorized custom client to the existing hosted session and use the deployed scene handlers, without changing NVIDIA's server deployment? Which operations have observable readback?
4. If the necessary model is not available there, can NVIDIA offer a small GPU/MaxLPS-enabled test environment or workload-specific power/performance characterization data?

These are deployment and model questions the public clone cannot answer. No secret value is needed for the discussion.

## Research and Verification Limits

Re-traced the actual browser message -> Kit manager -> USD attribute/color paths and their absence of power-response samples. Compared official Omniverse architecture, Batsim protocol/time, SimGrid energy, DSX Air and NVIDIA management/telemetry documentation. No live hosted session, GPU experiment or runnable adapter exists here; this is a static feasibility assessment, not an executed interoperability result.

Prior Emerald observations can support example scenarios, but power notches alone do not prove which job paused or why an hour boundary was chosen. Attribution requires associated schedule/actuator logs. Do not use earlier causal interpretations or a weighted throughput aggregate as automatic calibration or per-job SLA evidence.
