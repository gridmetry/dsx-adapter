(function (global, configuration) {
  'use strict';

  const origin = 'https://build.nvidia.com';
  const endpoint = 'http://127.0.0.1:28090/events';
  const profile = Object.freeze(configuration || { feedTag: 'main-a', edpSetting: '1.5', baselineLoadPercent: 80, maximumKW: 1531 });
  if (typeof profile.feedTag !== 'string' || !profile.feedTag || profile.feedTag.length > 64 ||
      !Number.isInteger(profile.baselineLoadPercent) || profile.baselineLoadPercent < 0 || profile.baselineLoadPercent > 100 ||
      !['1.2', '1.5'].includes(profile.edpSetting) ||
      profile.maximumKW !== ({ '1.2': 1316, '1.5': 1531 })[profile.edpSetting]) {
    throw new Error('Invalid adapter visualization profile');
  }
  const displayedWatts = percent => Math.round(profile.maximumKW * percent / 400) * 4000;

  function validate(event, now = Date.now()) {
    const target = event?.visualization;
    const received = Date.parse(event?.received_at);
    const deadline = Date.parse(target?.apply_before);
    if (event?.type !== 'command_received' || !/^[0-9a-f-]{36}$/i.test(event.id || '') ||
        event.feed_tag !== profile.feedTag || !Number.isFinite(received) || received > now + 30000 || received + 300000 <= now ||
        target?.type !== 'visualization_target' || target.id !== event.id || target.feed_tag !== event.feed_tag ||
        target.edp_setting !== profile.edpSetting || target.failed_rpps !== 0 || target.evidence !== 'visualization_input' ||
        !Number.isInteger(target.load_percent) || target.load_percent < 0 || target.load_percent > 100 ||
        target.displayed_power_watts !== displayedWatts(target.load_percent) ||
        !Number.isFinite(deadline) || deadline <= now || deadline > received + 300000 ||
        typeof event.release !== 'boolean' || target.release !== event.release) {
      throw new Error('Invalid or expired visualization receipt');
    }
    if (event.release) {
      if (event.constraint_watts !== undefined || target.requested_watts !== undefined || target.load_percent !== profile.baselineLoadPercent) {
        throw new Error('Invalid baseline release');
      }
    } else if (!Number.isFinite(event.constraint_watts) || event.constraint_watts <= 0 ||
        target.requested_watts !== event.constraint_watts || target.displayed_power_watts > event.constraint_watts) {
      throw new Error('Invalid power constraint');
    }
    if (event.end_time !== undefined || target.end_time !== undefined) {
      const end = Date.parse(event.end_time);
      if (!Number.isFinite(end) || end <= now || target.end_time !== event.end_time || deadline > end) {
        throw new Error('Invalid command interval');
      }
    }
    return target;
  }

  function validDuration(minutes) {
    return Number.isInteger(minutes) && minutes >= 1 && minutes <= 10;
  }

  function mount({ durationMinutes = 8 } = {}) {
    if (!validDuration(durationMinutes)) throw new Error('Receiver duration must be 1-10 whole minutes');
    if (global.location.origin !== origin || !global.location.pathname.endsWith('/omniverse-dsx-blueprint-for-ai-factories')) {
      throw new Error('Receiver requires the shared DSX page');
    }
    if (global.document.getElementById('gridmetry-dsx-receiver')) {
      throw new Error('Receiver already mounted');
    }
    const box = global.document.createElement('section');
    box.id = 'gridmetry-dsx-receiver';
    box.style.cssText = 'position:fixed;right:16px;top:76px;width:270px;padding:12px;background:#171b18;color:#fff;border:1px solid #6f8b72;border-radius:4px;z-index:2147483646;font:14px sans-serif;';
    box.innerHTML = '<strong>DSX Exchange</strong><form style="margin-top:10px"><label>Lab event token<input type="password" autocomplete="off" style="display:block;width:100%;box-sizing:border-box;margin:6px 0" required></label><label>Duration (minutes)<input id="gridmetry-dsx-duration" type="number" min="1" max="10" step="1" value="' + durationMinutes + '" style="display:block;width:100%;box-sizing:border-box;margin:6px 0" required></label><button type="submit">Connect</button></form><button type="button" id="gridmetry-dsx-disconnect" style="margin-top:8px" disabled>Disconnect</button><p id="gridmetry-dsx-status">Idle</p><output id="gridmetry-dsx-result"></output>';
    global.document.body.appendChild(box);
    const status = box.querySelector('#gridmetry-dsx-status');
    const output = box.querySelector('output');
    const disconnect = box.querySelector('#gridmetry-dsx-disconnect');
    const report = { state: 'idle', connected: false, applied: 0, lastId: null, loadPercent: null, displayedPowerWatts: null, restored: false, authenticated: false, httpStatus: null, failureCode: null, durationMinutes, baselineLoadPercent: profile.baselineLoadPercent, edpSetting: profile.edpSetting, adaptations: 0, pendingId: null, uiEdpSetting: null, failedRpps: 0 };
    const failureMessages = {
      authentication_rejected: 'Adapter authentication rejected',
      adapter_unreachable: 'Adapter connection unavailable',
      stream_unavailable: 'Adapter event stream unavailable',
      simulation_not_ready: 'Simulation not ready',
      ui_observation_failed: 'Simulation update not observed',
      event_stream_failed: 'Event stream failed',
      invalid_receipt: 'Invalid or expired target',
      target_expired: 'Target expired',
    };
    const seen = new Set();
    const applied = new Set();
    let controller;
    let baseline;
    let expiry;
    let lifetime;
    let generation = 0;
    let restoring;
    let monitor;
    let monitorBusy = false;
    let retainedTarget;
    let pendingBaselineRestore = false;
    let lastSettings;

    function failure(code) {
      const error = new Error('Receiver operation failed');
      error.receiverCode = code;
      return error;
    }

    function fail(error, fallback = 'event_stream_failed') {
      report.failureCode = Object.hasOwn(failureMessages, error?.receiverCode) ? error.receiverCode : fallback;
      return stop('failed');
    }

    function electrical() {
      const input = global.document.querySelector('input[type="range"]');
      const panel = input?.closest('[role="tabpanel"]');
      const video = global.document.getElementById('remote-video');
      if (!panel || !video || video.readyState < 2 || video.paused) {
        throw failure('simulation_not_ready');
      }
      const selected = Array.from(panel.querySelectorAll('[role="radio"][aria-checked="true"]'));
      const rpp = selected.find(element => /^rpp-[0-4]$/.test(element.id));
      const edp = selected.find(element => /^edp-1\.[25]$/.test(element.id));
      if (!rpp || !edp) throw failure('simulation_not_ready');
      const failedRpps = Number(rpp.id.slice(4));
      const edpSetting = edp.id.slice(4);
      const buttons = Array.from(panel.querySelectorAll('button'));
      return { input, panel, failedRpps, edpSetting, maximumKW: edpSetting === '1.5' ? 1531 : 1316, begin: buttons.find(button => button.textContent.trim() === 'Begin Test'), stop: buttons.find(button => button.textContent.trim() === 'Stop Test') };
    }

    const settingsKey = view => `${view.edpSetting}/${view.failedRpps}`;
    const liveWatts = (percent, view) => view.failedRpps === 4 ? 0 : Math.round(view.maximumKW * percent / (100 * (4 - view.failedRpps))) * (4 - view.failedRpps) * 1000;

    function pause(state, target) {
      report.state = state;
      report.pendingId = target?.id || null;
      report.failureCode = null;
      status.textContent = state === 'paused_faults' ? 'All RPPs failed - waiting' : 'Waiting for Electrical simulation';
    }

    function setLoad(input, percent) {
      Object.getOwnPropertyDescriptor(global.HTMLInputElement.prototype, 'value').set.call(input, String(percent));
      input.dispatchEvent(new global.Event('input', { bubbles: true }));
      input.dispatchEvent(new global.Event('change', { bubbles: true }));
    }

    async function apply(percent, target, operation = generation) {
      if (controller.signal.aborted || operation !== generation) return false;
      if (target && ((!applied.has(target.id) && Date.parse(target.apply_before) <= Date.now()) || (target.end_time && Date.parse(target.end_time) <= Date.now()))) throw failure('target_expired');
      const view = electrical();
      report.uiEdpSetting = view.edpSetting;
      report.failedRpps = view.failedRpps;
      if (view.failedRpps === 4) { pause('paused_faults', target); return false; }
      if (target && !target.release) {
        percent = 100;
        while (percent > 0 && liveWatts(percent, view) > target.requested_watts) percent--;
      }
      const expected = liveWatts(percent, view);
      setLoad(view.input, percent);
      view.begin?.click();
      const until = Date.now() + 4000;
      while (Date.now() < until) {
        if (controller.signal.aborted || operation !== generation) return false;
        if (target && ((!applied.has(target.id) && Date.parse(target.apply_before) <= Date.now()) || (target.end_time && Date.parse(target.end_time) <= Date.now()))) throw failure('target_expired');
        const current = electrical();
        if (settingsKey(current) !== settingsKey(view)) { pause('paused_ui', target); return false; }
        const row = Array.from(current.panel.querySelectorAll('tr')).find(element => element.textContent.includes('Power (kW)'));
        const cells = Array.from(row?.querySelectorAll('td') || []).slice(1);
        const values = cells.slice(current.failedRpps).map(cell => /^\d+$/.test(cell.textContent.trim()) ? Number(cell.textContent) : NaN);
        if (current.stop && Number(current.input.value) === percent && cells.length === 4 &&
            cells.slice(0, current.failedRpps).every(cell => cell.textContent.trim() === '-') &&
            values.every(Number.isFinite) && values.reduce((sum, value) => sum + value, 0) * 1000 === expected) {
          report.loadPercent = percent;
          report.displayedPowerWatts = expected;
          lastSettings = settingsKey(current);
          return true;
        }
        await new Promise(resolve => global.setTimeout(resolve, 40));
      }
      throw failure('ui_observation_failed');
    }

    async function restore() {
      if (!baseline) return;
      let restored = false;
      const until = Date.now() + 2000;
      while (Date.now() < until) {
        try {
          const view = electrical();
          if (!baseline.playing) view.stop?.click();
          setLoad(view.input, baseline.load);
          if (baseline.playing) view.begin?.click();
          await new Promise(resolve => global.setTimeout(resolve, 40));
          const current = electrical();
          if (Number(current.input.value) === baseline.load && Boolean(current.stop) === baseline.playing) {
            restored = true;
            break;
          }
        } catch { break; }
      }
      report.restored = restored;
    }

    function stop(state = 'disconnected') {
      if (restoring) return restoring;
      controller?.abort();
      global.clearTimeout(expiry);
      global.clearTimeout(lifetime);
      global.clearInterval(monitor);
      pendingBaselineRestore = false;
      retainedTarget = null;
      generation++;
      report.connected = false;
      report.state = state;
      status.textContent = state === 'failed' ? failureMessages[report.failureCode] || 'Receiver failed' : state;
      disconnect.disabled = true;
      restoring = restore();
      return restoring;
    }

    function recordApplied(target) {
      if (applied.has(target.id)) report.adaptations++;
      else {
        applied.add(target.id);
        if (applied.size > 128) applied.delete(applied.values().next().value);
        report.applied++;
      }
      report.lastId = target.id;
      report.pendingId = null;
      report.failureCode = null;
      report.state = target.release ? 'release_applied' : 'target_applied';
      status.textContent = report.state;
      output.textContent = `${report.loadPercent}% / ${report.displayedPowerWatts / 1000} kW (${report.uiEdpSetting}, ${report.failedRpps} failed RPPs)`;
      if (target.release) retainedTarget = null;
    }

    async function restoreInterval(operation) {
      try {
        const observed = await apply(profile.baselineLoadPercent, undefined, operation);
        if (!observed || controller.signal.aborted || operation !== generation) return;
        pendingBaselineRestore = false;
        report.state = 'interval_expired';
        status.textContent = report.state;
        output.textContent = `${report.loadPercent}% / ${report.displayedPowerWatts / 1000} kW`;
      } catch {
        if (!controller.signal.aborted && operation === generation) pause('paused_ui');
      }
    }

    async function expireTarget(target) {
      if (controller.signal.aborted || retainedTarget !== target) return;
      global.clearTimeout(expiry);
      retainedTarget = null;
      pendingBaselineRestore = true;
      report.pendingId = null;
      await restoreInterval(++generation);
    }

    async function reconcile(target, operation) {
      try {
        if (!await apply(target.load_percent, target, operation) || controller.signal.aborted || operation !== generation) return;
        recordApplied(target);
      } catch (error) {
        if (controller.signal.aborted || operation !== generation) return;
        if (error?.receiverCode === 'target_expired') {
          if (target.end_time && Date.parse(target.end_time) <= Date.now()) await expireTarget(target);
          else {
            retainedTarget = null;
            report.pendingId = null;
            report.state = 'target_expired';
            status.textContent = 'Target expired';
          }
        } else if (error?.receiverCode === 'simulation_not_ready' || error?.receiverCode === 'ui_observation_failed') {
          pause('paused_ui', target);
        } else throw error;
      }
    }

    async function accept(event) {
      let target;
      try { target = validate(event); } catch { throw failure('invalid_receipt'); }
      if (seen.has(target.id)) return;
      seen.add(target.id);
      if (seen.size > 128) seen.delete(seen.values().next().value);
      global.clearTimeout(expiry);
      pendingBaselineRestore = false;
      retainedTarget = target;
      report.pendingId = target.id;
      const currentGeneration = ++generation;
      if (target.end_time && !target.release) {
        expiry = global.setTimeout(() => {
          expireTarget(target).catch(error => fail(error));
        }, Math.max(0, Math.min(Date.parse(target.end_time) - Date.now(), 2147483647)));
      }
      await reconcile(target, currentGeneration);
    }

    box.querySelector('form').addEventListener('submit', async event => {
      event.preventDefault();
      if (controller) return;
      const input = box.querySelector('input');
      let token = input.value.trim();
      input.value = '';
      const selectedDuration = Number(box.querySelector('#gridmetry-dsx-duration').value);
      if (!validDuration(selectedDuration)) { token = ''; status.textContent = 'Invalid duration (1-10 minutes)'; return; }
      if (!/^[A-Za-z0-9._-]{16,4096}$/.test(token)) { status.textContent = 'Invalid token format'; return; }
      report.durationMinutes = selectedDuration;
      input.closest('form').remove();
      controller = new global.AbortController();
      disconnect.disabled = false;
      report.state = 'connecting';
      status.textContent = report.state;
      lifetime = global.setTimeout(() => stop('time_limit'), selectedDuration * 60000);
      let failureStage = 'adapter_unreachable';
      try {
        const response = await global.fetch(endpoint, { headers: { Authorization: `Bearer ${token}` }, credentials: 'omit', signal: controller.signal });
        token = '';
        report.httpStatus = response.status;
        if (response.status === 401 || response.status === 403) throw failure('authentication_rejected');
        if (response.status !== 200 || !response.headers.get('content-type')?.startsWith('text/event-stream')) throw failure('stream_unavailable');
        report.authenticated = true;
        failureStage = 'simulation_not_ready';
        const view = electrical();
        baseline = { load: Number(view.input.value), playing: Boolean(view.stop) };
        failureStage = 'ui_observation_failed';
        await apply(profile.baselineLoadPercent);
        monitor = global.setInterval(() => {
          if (!report.connected || controller.signal.aborted || (!retainedTarget && !pendingBaselineRestore) || monitorBusy) return;
          try {
            const view = electrical();
            if (!pendingBaselineRestore && !report.pendingId && settingsKey(view) === lastSettings) return;
          } catch { pause('paused_ui', retainedTarget); return; }
          monitorBusy = true;
          const operation = ++generation;
          const work = pendingBaselineRestore ? restoreInterval(operation) : reconcile(retainedTarget, operation);
          work.catch(error => fail(error)).finally(() => { monitorBusy = false; });
        }, 250);
        failureStage = 'event_stream_failed';
        const reader = response.body.getReader();
        const decoder = new global.TextDecoder();
        let buffer = '';
        while (!controller.signal.aborted) {
          const chunk = await reader.read();
          if (chunk.done) break;
          buffer += decoder.decode(chunk.value, { stream: true }).replace(/\r/g, '');
          if (buffer.length > 65536) throw new Error('Event buffer limit exceeded');
          let separator;
          while ((separator = buffer.indexOf('\n\n')) >= 0) {
            const frame = buffer.slice(0, separator);
            buffer = buffer.slice(separator + 2);
            if (frame === ': connected') {
              report.connected = true;
              report.state = 'receiving';
              status.textContent = report.state;
            }
            const data = frame.split('\n').filter(line => line.startsWith('data: ')).map(line => line.slice(6)).join('\n');
            if (data) await accept(JSON.parse(data));
          }
        }
        await stop('disconnected');
      } catch (error) {
        token = '';
        if (controller.signal.aborted) await stop(report.state);
        else await fail(error, failureStage);
      }
    });
    disconnect.addEventListener('click', () => stop());
    return { report: () => ({ ...report }), disconnect: () => stop() };
  }

  global.GridmetryDSXReceiver = { validate, mount };
})(globalThis);
