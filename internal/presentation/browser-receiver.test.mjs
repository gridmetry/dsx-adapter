import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';
import vm from 'node:vm';

const context = vm.createContext({});
const source = await readFile(new URL('./browser-receiver.js', import.meta.url), 'utf8');
vm.runInContext(source, context);
const validate = context.GridmetryDSXReceiver.validate;
const now = Date.parse('2026-10-13T00:00:00Z');

function configuredSource(profile) {
  return source.trim().replace(/\}\)\(globalThis\);$/, '})(globalThis,' + JSON.stringify(profile) + ');');
}

function receipt() {
  return {
    type: 'command_received', id: '147c9f7f-a3db-4a1c-b3d1-9656ed58babf', feed_tag: 'main-a',
    release: false, constraint_watts: 920000, received_at: new Date(now).toISOString(),
    visualization: {
      type: 'visualization_target', id: '147c9f7f-a3db-4a1c-b3d1-9656ed58babf', feed_tag: 'main-a',
      load_percent: 60, edp_setting: '1.5', failed_rpps: 0, displayed_power_watts: 920000,
      requested_watts: 920000, release: false, apply_before: new Date(now + 300000).toISOString(),
      evidence: 'visualization_input',
    },
  };
}

test('validates the correlated signed-receipt handoff', () => {
  assert.equal(validate(receipt(), now).load_percent, 60);
});

test('accepts full load above startup baseline and honors generated profile', () => {
  const full = receipt();
  full.constraint_watts = full.visualization.requested_watts = 1532000;
  full.visualization.load_percent = 100;
  full.visualization.displayed_power_watts = 1532000;
  assert.equal(validate(full, now).load_percent, 100);
  const profile = { feedTag: 'main-a', edpSetting: '1.2', baselineLoadPercent: 60, maximumKW: 1316 };
  const profileContext = vm.createContext({});
  vm.runInContext(configuredSource(profile), profileContext);
  full.constraint_watts = full.visualization.requested_watts = 1316000;
  full.visualization.edp_setting = '1.2';
  full.visualization.displayed_power_watts = 1316000;
  assert.equal(profileContext.GridmetryDSXReceiver.validate(full, now).load_percent, 100);
  full.release = full.visualization.release = true;
  delete full.constraint_watts;
  delete full.visualization.requested_watts;
  full.visualization.load_percent = 60;
  full.visualization.displayed_power_watts = 788000;
  assert.equal(profileContext.GridmetryDSXReceiver.validate(full, now).load_percent, 60);
  full.visualization.load_percent = 80;
  full.visualization.displayed_power_watts = 1052000;
  assert.throws(() => profileContext.GridmetryDSXReceiver.validate(full, now));
});

test('rejects stale, mismatched and unsafe visual commands', () => {
  for (const mutate of [
    event => { event.type = 'other'; },
    event => { event.visualization.id = 'different'; },
    event => { event.feed_tag = 'other'; },
    event => { event.visualization.failed_rpps = 1; },
    event => { event.visualization.edp_setting = '1.2'; },
    event => { event.visualization.load_percent = 61; },
    event => { event.visualization.apply_before = new Date(now).toISOString(); },
    event => { event.received_at = new Date(now - 360000).toISOString(); },
    event => { event.visualization.requested_watts = 900000; },
    event => { event.visualization.load_percent = 81; },
    event => { event.visualization.release = true; },
  ]) {
    const event = receipt();
    mutate(event);
    assert.throws(() => validate(event, now));
  }
});

test('release restores the configured baseline, not zero power', () => {
  const event = receipt();
  event.release = true;
  delete event.constraint_watts;
  event.visualization.release = true;
  delete event.visualization.requested_watts;
  event.visualization.load_percent = 80;
  event.visualization.displayed_power_watts = 1224000;
  assert.equal(validate(event, now).load_percent, 80);
});

function lifecycle(statusCode = 200, options = {}, profile) {
  const configuration = profile || { edpSetting: '1.5', baselineLoadPercent: 80, maximumKW: 1531 };
  const model = { playing: false, percent: 0, defer80: false, mounted: false, removed: false, videoReady: 4, videoPaused: false, fetchFailure: false, edpSetting: configuration.edpSetting, failedRpps: 0 };
  let clock = now;
  let timerID = 0;
  const timers = new Map();
  const intervals = new Map();
  const chunks = [];
  let waiting;
  const handlers = {};
  class Input {
    constructor(value) { this.stored = String(value); }
    get value() { return this.stored; }
    set value(value) { this.stored = String(value); }
    closest(selector) { return selector === 'form' ? form : panel; }
    dispatchEvent() { if (this === range && !(model.defer80 && Number(this.value) === 80)) model.percent = Number(this.value); }
  }
  const range = new Input(0);
  const token = new Input('synthetic-readonly-token-1234567890');
  const duration = new Input(options.durationMinutes ?? 8);
  const begin = { textContent: 'Begin Test', click: () => { model.playing = true; } };
  const stop = { textContent: 'Stop Test', click: () => { model.playing = false; } };
  const panel = {
    querySelector: selector => selector.includes('rpp-' + model.failedRpps) || selector.includes('edp-' + model.edpSetting) ? {} : null,
    querySelectorAll: selector => {
      if (selector === 'button') return [model.playing ? stop : begin];
      if (selector.includes('[role="radio"]')) return [{ id: 'rpp-' + model.failedRpps }, { id: 'edp-' + model.edpSetting }];
      const maximumKW = model.edpSetting === '1.5' ? 1531 : 1316;
      const healthy = 4 - model.failedRpps;
      const power = healthy ? Math.round(maximumKW * model.percent / (100 * healthy)) : 0;
      return [{ textContent: 'Power (kW)', querySelectorAll: () => [{ textContent: 'Power (kW)' }, ...Array.from({ length: 4 }, (_, index) => ({ textContent: model.playing && index >= model.failedRpps ? String(power) : '-' }))] }];
    },
  };
  const form = { addEventListener: (name, callback) => { handlers[name] = callback; }, remove: () => { model.removed = true; } };
  const status = { textContent: '' };
  const output = { textContent: '' };
  const disconnect = { disabled: true, addEventListener: () => {} };
  const box = {
    style: {},
    querySelector: selector => ({ form, input: token, output, '#gridmetry-dsx-status': status, '#gridmetry-dsx-disconnect': disconnect, '#gridmetry-dsx-duration': duration })[selector],
  };
  const fixtureContext = vm.createContext({
    Date: class extends Date { static now() { return clock; } },
    location: { origin: 'https://build.nvidia.com', pathname: '/nvidia/omniverse-dsx-blueprint-for-ai-factories' },
    document: {
      createElement: () => box,
      body: { appendChild: () => { model.mounted = true; } },
      getElementById: name => name === 'remote-video' ? { readyState: model.videoReady, paused: model.videoPaused } : model.mounted ? box : null,
      querySelector: () => range,
    },
    HTMLInputElement: Input,
    Event: class { constructor(type) { this.type = type; } },
    AbortController, TextDecoder,
    setTimeout: (callback, delay) => { const identity = ++timerID; timers.set(identity, { callback, delay }); return identity; },
    clearTimeout: identity => timers.delete(identity),
    setInterval: callback => { const identity = ++timerID; intervals.set(identity, callback); return identity; },
    clearInterval: identity => intervals.delete(identity),
    fetch: async (url, options) => {
      assert.equal(url, 'http://127.0.0.1:28090/events');
      assert.equal(options.credentials, 'omit');
      assert.equal(options.headers.Authorization, 'Bearer synthetic-readonly-token-1234567890');
      if (model.fetchFailure) throw new Error('synthetic-private-network-error');
      options.signal.addEventListener('abort', () => { waiting?.({ done: true }); waiting = undefined; });
      return {
        status: statusCode,
        headers: { get: () => 'text/event-stream; charset=utf-8' },
        body: { getReader: () => ({ read: () => chunks.length ? Promise.resolve(chunks.shift()) : new Promise(resolve => { waiting = resolve; }) }) },
      };
    },
  });
  vm.runInContext(profile ? configuredSource(profile) : source, fixtureContext);
  const receiver = fixtureContext.GridmetryDSXReceiver.mount(options);
  const drain = () => new Promise(resolve => setImmediate(resolve));
  return {
    model, token, duration, receiver, drain,
    monitor: async () => { for (const callback of intervals.values()) callback(); await drain(); },
    activeIntervals: () => intervals.size,
    timerDelays: () => [...timers.values()].map(timer => timer.delay),
    start: () => handlers.submit({ preventDefault() {} }),
    emit: text => {
      const chunk = { done: false, value: new TextEncoder().encode(text) };
      if (waiting) { const resolve = waiting; waiting = undefined; resolve(chunk); } else chunks.push(chunk);
    },
    polls: async () => {
      for (const [identity, timer] of [...timers]) if (timer.delay === 40) { timers.delete(identity); timer.callback(); }
      await drain();
    },
    expire: async delay => {
      const [identity, timer] = [...timers].find(([, timer]) => timer.delay === delay);
      timers.delete(identity);
      clock += delay;
      timer.callback();
      await drain();
    },
  };
}

async function connectedFixture(options = {}, profile) {
  const fixture = lifecycle(200, options, profile);
  fixture.running = fixture.start();
  await fixture.drain();
  fixture.emit(': con');
  fixture.emit('nected\n\n');
  await fixture.drain();
  assert.equal(fixture.receiver.report().state, 'receiving');
  assert.equal(fixture.model.percent, profile?.baselineLoadPercent ?? 80);
  assert.equal(fixture.token.value, '');
  assert.equal(fixture.model.removed, true);
  return fixture;
}

async function disconnectFixture(fixture) {
  const stopped = fixture.receiver.disconnect();
  await fixture.polls();
  await stopped;
  await fixture.running;
  assert.equal(fixture.receiver.report().restored, true);
  assert.equal(fixture.model.playing, false);
  assert.equal(fixture.model.percent, 0);
}

test('stream lifecycle applies once, releases and restores without leaking the token', async () => {
  const fixture = await connectedFixture();
  const event = receipt();
  fixture.emit('data: ' + JSON.stringify(event).slice(0, 45));
  fixture.emit(JSON.stringify(event).slice(45) + '\n\n');
  await fixture.drain();
  assert.equal(fixture.receiver.report().loadPercent, 60);
  fixture.emit('data: ' + JSON.stringify(event) + '\n\n');
  await fixture.drain();
  assert.equal(fixture.receiver.report().applied, 1);
  const release = receipt();
  release.id = release.visualization.id = '147c9f7f-a3db-4a1c-b3d1-9656ed58babe';
  release.release = release.visualization.release = true;
  delete release.constraint_watts;
  delete release.visualization.requested_watts;
  release.visualization.load_percent = 80;
  release.visualization.displayed_power_watts = 1224000;
  fixture.emit('data: ' + JSON.stringify(release) + '\n\n');
  await fixture.drain();
  assert.equal(fixture.receiver.report().state, 'release_applied');
  assert.equal(fixture.receiver.report().applied, 2);
  assert.equal(JSON.stringify(fixture.receiver.report()).includes('synthetic'), false);
  await disconnectFixture(fixture);
});

test('configured EDP lifecycle starts at baseline and supports full load', async () => {
  const profile = { feedTag: 'main-a', edpSetting: '1.2', baselineLoadPercent: 60, maximumKW: 1316 };
  const fixture = await connectedFixture({}, profile);
  assert.equal(fixture.receiver.report().baselineLoadPercent, 60);
  assert.equal(fixture.receiver.report().edpSetting, '1.2');
  const full = receipt();
  full.constraint_watts = full.visualization.requested_watts = 1316000;
  full.visualization.edp_setting = '1.2';
  full.visualization.load_percent = 100;
  full.visualization.displayed_power_watts = 1316000;
  fixture.emit('data: ' + JSON.stringify(full) + '\n\n');
  await fixture.drain();
  assert.equal(fixture.model.percent, 100);
  assert.equal(fixture.receiver.report().displayedPowerWatts, 1316000);
  const release = JSON.parse(JSON.stringify(full));
  release.id = release.visualization.id = '147c9f7f-a3db-4a1c-b3d1-9656ed58babe';
  release.release = release.visualization.release = true;
  delete release.constraint_watts;
  delete release.visualization.requested_watts;
  release.visualization.load_percent = 60;
  release.visualization.displayed_power_watts = 788000;
  fixture.emit('data: ' + JSON.stringify(release) + '\n\n');
  await fixture.drain();
  assert.equal(fixture.model.percent, 60);
  assert.equal(fixture.receiver.report().state, 'release_applied');
  await disconnectFixture(fixture);
});

test('manual EDP and RPP changes remap the retained watt target without disconnecting', async () => {
  const fixture = await connectedFixture();
  fixture.emit('data: ' + JSON.stringify(receipt()) + '\n\n');
  await fixture.drain();
  fixture.model.edpSetting = '1.2';
  fixture.model.failedRpps = 1;
  await fixture.monitor();
  assert.equal(fixture.receiver.report().connected, true);
  assert.equal(fixture.model.edpSetting, '1.2');
  assert.equal(fixture.model.failedRpps, 1);
  assert.equal(fixture.model.percent, 69);
  assert.equal(fixture.receiver.report().displayedPowerWatts, 909000);
  assert.equal(fixture.receiver.report().applied, 1);
  const next = receipt();
  next.id = next.visualization.id = '147c9f7f-a3db-4a1c-b3d1-9656ed58bab1';
  next.constraint_watts = next.visualization.requested_watts = 650000;
  next.visualization.load_percent = 42;
  next.visualization.displayed_power_watts = 644000;
  fixture.emit('data: ' + JSON.stringify(next) + '\n\n');
  await fixture.drain();
  assert.equal(fixture.receiver.report().connected, true);
  assert.equal(fixture.receiver.report().lastId, next.id);
  assert.ok(fixture.receiver.report().displayedPowerWatts <= 650000);
  await disconnectFixture(fixture);
  assert.equal(fixture.model.failedRpps, 1);
  assert.equal(fixture.activeIntervals(), 0);
});

test('EDP-only changes keep the accepted watt ceiling while preserving the UI', async () => {
  const profile = { feedTag: 'main-a', edpSetting: '1.2', baselineLoadPercent: 80, maximumKW: 1316 };
  const fixture = await connectedFixture({}, profile);
  const target = receipt();
  target.constraint_watts = target.visualization.requested_watts = 1316000;
  target.visualization.edp_setting = '1.2';
  target.visualization.load_percent = 100;
  target.visualization.displayed_power_watts = 1316000;
  fixture.emit('data: ' + JSON.stringify(target) + '\n\n');
  await fixture.drain();
  fixture.model.edpSetting = '1.5';
  await fixture.monitor();
  assert.equal(fixture.receiver.report().connected, true);
  assert.equal(fixture.model.percent, 86);
  assert.equal(fixture.model.edpSetting, '1.5');
  assert.equal(fixture.receiver.report().uiEdpSetting, '1.5');
  assert.ok(fixture.receiver.report().displayedPowerWatts <= 1316000);
  await disconnectFixture(fixture);
});

test('temporary simulation unavailability pauses rather than aborts Flex', async () => {
  const fixture = await connectedFixture();
  fixture.model.videoReady = 0;
  fixture.emit('data: ' + JSON.stringify(receipt()) + '\n\n');
  await fixture.drain();
  assert.equal(fixture.receiver.report().connected, true);
  assert.equal(fixture.receiver.report().state, 'paused_ui');
  fixture.model.videoReady = 4;
  await fixture.monitor();
  assert.equal(fixture.receiver.report().state, 'target_applied');
  assert.equal(fixture.receiver.report().loadPercent, 60);
  await disconnectFixture(fixture);
});

test('all failed RPPs pause application and recover without losing the stream', async () => {
  const fixture = await connectedFixture();
  fixture.model.failedRpps = 4;
  fixture.emit('data: ' + JSON.stringify(receipt()) + '\n\n');
  await fixture.drain();
  assert.equal(fixture.receiver.report().connected, true);
  assert.equal(fixture.receiver.report().state, 'paused_faults');
  assert.equal(fixture.receiver.report().applied, 0);
  fixture.model.failedRpps = 2;
  await fixture.monitor();
  assert.equal(fixture.receiver.report().state, 'target_applied');
  assert.equal(fixture.receiver.report().applied, 1);
  assert.ok(fixture.receiver.report().displayedPowerWatts <= 920000);
  await disconnectFixture(fixture);
});

test('newer target retains ownership during an in-flight expiry restore', async () => {
  const fixture = await connectedFixture();
  const first = receipt();
  first.end_time = first.visualization.end_time = new Date(now + 1000).toISOString();
  first.visualization.apply_before = first.end_time;
  fixture.emit('data: ' + JSON.stringify(first) + '\n\n');
  await fixture.drain();
  fixture.model.defer80 = true;
  await fixture.expire(1000);
  const newer = receipt();
  newer.id = newer.visualization.id = '147c9f7f-a3db-4a1c-b3d1-9656ed58bab1';
  newer.constraint_watts = newer.visualization.requested_watts = 800000;
  newer.visualization.load_percent = 52;
  newer.visualization.displayed_power_watts = 796000;
  fixture.emit('data: ' + JSON.stringify(newer) + '\n\n');
  await fixture.drain();
  await fixture.polls();
  assert.equal(fixture.receiver.report().state, 'target_applied');
  assert.equal(fixture.receiver.report().connected, true);
  assert.equal(fixture.receiver.report().lastId, newer.id);
  assert.equal(fixture.model.percent, 52);
  await disconnectFixture(fixture);
});

test('expired interval baseline is restored after temporary UI unavailability', async () => {
  const fixture = await connectedFixture();
  const target = receipt();
  target.end_time = target.visualization.end_time = new Date(now + 1000).toISOString();
  target.visualization.apply_before = target.end_time;
  fixture.emit('data: ' + JSON.stringify(target) + '\n\n');
  await fixture.drain();
  fixture.model.videoReady = 0;
  await fixture.expire(1000);
  assert.equal(fixture.receiver.report().connected, true);
  fixture.model.videoReady = 4;
  await fixture.monitor();
  assert.equal(fixture.receiver.report().state, 'interval_expired');
  assert.equal(fixture.model.percent, 80);
  await disconnectFixture(fixture);
});

test('unauthorized stream never starts or changes the scenario', async () => {
  const fixture = lifecycle(401);
  await fixture.start();
  assert.equal(fixture.receiver.report().state, 'failed');
  assert.equal(fixture.receiver.report().failureCode, 'authentication_rejected');
  assert.equal(fixture.receiver.report().httpStatus, 401);
  assert.equal(fixture.receiver.report().authenticated, false);
  assert.equal(fixture.model.playing, false);
  assert.equal(fixture.model.percent, 0);
  assert.equal(fixture.token.value, '');
});

test('authenticated stream reports unready video without changing the scenario', async () => {
  const fixture = lifecycle();
  fixture.model.videoReady = 0;
  fixture.model.videoPaused = true;
  await fixture.start();
  const report = fixture.receiver.report();
  assert.equal(report.state, 'failed');
  assert.equal(report.failureCode, 'simulation_not_ready');
  assert.equal(report.httpStatus, 200);
  assert.equal(report.authenticated, true);
  assert.equal(fixture.model.playing, false);
  assert.equal(fixture.model.percent, 0);
  assert.equal(fixture.token.value, '');
});

test('connection errors are classified without exposing raw errors', async () => {
  const fixture = lifecycle();
  fixture.model.fetchFailure = true;
  await fixture.start();
  assert.equal(fixture.receiver.report().failureCode, 'adapter_unreachable');
  assert.equal(JSON.stringify(fixture.receiver.report()).includes('synthetic-private-network-error'), false);
  assert.equal(fixture.model.playing, false);
});

test('receiver defaults to eight minutes and restores at its deadline', async () => {
  const fixture = await connectedFixture();
  assert.equal(fixture.receiver.report().durationMinutes, 8);
  assert.ok(fixture.timerDelays().includes(480000));
  assert.equal(fixture.timerDelays().includes(120000), false);
  await fixture.expire(480000);
  await fixture.polls();
  await fixture.running;
  assert.equal(fixture.receiver.report().state, 'time_limit');
  assert.equal(fixture.receiver.report().restored, true);
  assert.equal(fixture.model.playing, false);
  assert.equal(fixture.model.percent, 0);
});

test('mount duration and form override control the connection lifetime', async () => {
  const fixture = lifecycle(200, { durationMinutes: 6 });
  assert.equal(fixture.receiver.report().durationMinutes, 6);
  fixture.duration.value = 4;
  fixture.running = fixture.start();
  await fixture.drain();
  fixture.emit(': connected\n\n');
  await fixture.drain();
  assert.equal(fixture.receiver.report().durationMinutes, 4);
  assert.ok(fixture.timerDelays().includes(240000));
  await disconnectFixture(fixture);
});

test('invalid durations are rejected before mounting or authentication', async () => {
  for (const durationMinutes of [0, -1, 11, 1.5, NaN, Infinity, '8']) {
    assert.throws(() => lifecycle(200, { durationMinutes }));
  }
  const fixture = lifecycle();
  fixture.duration.value = 0;
  await fixture.start();
  assert.equal(fixture.model.removed, false);
  assert.equal(fixture.receiver.report().authenticated, false);
  assert.equal(fixture.receiver.report().state, 'idle');
});

test('command expiry beyond two minutes is not shortened', async () => {
  const fixture = await connectedFixture();
  const event = receipt();
  event.end_time = event.visualization.end_time = new Date(now + 240000).toISOString();
  event.visualization.apply_before = event.end_time;
  fixture.emit('data: ' + JSON.stringify(event) + '\n\n');
  await fixture.drain();
  assert.ok(fixture.timerDelays().includes(240000));
  assert.equal(fixture.timerDelays().includes(120000), false);
  await disconnectFixture(fixture);
});
