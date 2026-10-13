import assert from 'node:assert/strict';
import { createServer } from 'node:http';
import { readFile } from 'node:fs/promises';
import { pathToFileURL } from 'node:url';

const signalingPath = '/v2/predict/websocket/blueprints/omniverse-dsx-blueprint-for-ai-factories';
const maximumDurationMs = 60000;

function selectedStreamType(types) {
  return types.NVCF;
}

function allowedSocket(address, protocols, attempts, live) {
  if (!live || attempts !== 0 || (protocols !== undefined && (!Array.isArray(protocols) || protocols.length !== 0))) return false;
  try {
    const url = new URL(address);
    return url.protocol === 'wss:' && url.hostname === 'api.ngc.nvidia.com' &&
      (!url.port || url.port === '443') && !url.username && !url.password && !url.hash &&
      url.pathname === `${signalingPath}/sign_in` && url.searchParams.size === 2 &&
      url.searchParams.getAll('version').length === 1 && url.searchParams.get('version') === '2' &&
      url.searchParams.getAll('peer_id').length === 1 && /^peer-\d{10}$/.test(url.searchParams.get('peer_id') || '');
  } catch {
    return false;
  }
}

function connectionOutcome(report, live) {
  if (!report.socketOpened && (report.socketError || report.socketClosed)) return 'connection_failed_before_upgrade';
  if (report.socketPolicyDenied) return live ? 'socket_policy_rejected' : 'offline_gate_verified';
  if (report.videoFrames >= 2 && report.dataChannelsOpen > 0 && report.pixelRange > 8 && report.nonDarkPixels > 100) return 'rendered_frames_and_open_datachannel';
  if (report.socketError || report.socketClosed || report.sdkFailure) return 'connection_failed';
  return 'connection_deadline_or_no_frames';
}

function safeSocketFailure(message) {
  const text = typeof message === 'string' ? message : '';
  const status = text.match(/Unexpected response code:\s*([1-5]\d\d)/i);
  if (status) return { failureClass: 'http_handshake_rejected', failureHTTPStatus: Number(status[1]) };
  const codes = ['ERR_CERT_AUTHORITY_INVALID', 'ERR_CERT_DATE_INVALID', 'ERR_NAME_NOT_RESOLVED', 'ERR_CONNECTION_RESET', 'ERR_CONNECTION_CLOSED', 'ERR_CONNECTION_REFUSED', 'ERR_CONNECTION_TIMED_OUT', 'ERR_TUNNEL_CONNECTION_FAILED', 'ERR_SSL_PROTOCOL_ERROR'];
  return { failureClass: codes.find(code => text.includes(code)) || 'unclassified_socket_error', failureHTTPStatus: 0 };
}

function selfTest() {
  assert.equal(selectedStreamType({ DIRECT: 'direct', NVCF: 'nvcf' }), 'nvcf');
  const target = `wss://api.ngc.nvidia.com:443${signalingPath}/sign_in?peer_id=peer-1234567890&version=2`;
  assert.equal(allowedSocket(target, [], 0, true), true);
  assert.equal(allowedSocket(target, [], 0, false), false);
  assert.equal(allowedSocket(target, [], 1, true), false);
  assert.equal(allowedSocket(target, ['Authorization-Bearer.synthetic'], 0, true), false);
  assert.equal(allowedSocket(target.replace('api.ngc.nvidia.com', 'example.com'), [], 0, true), false);
  assert.equal(allowedSocket(target.replace('wss:', 'ws:'), [], 0, true), false);
  assert.equal(allowedSocket(`${target}&token=synthetic`, [], 0, true), false);
  assert.equal(allowedSocket(`${target}&version=2`, [], 0, true), false);
  assert.equal(allowedSocket(`${target}#fragment`, [], 0, true), false);
  assert.equal(allowedSocket(target.replace(':443', ':8443'), [], 0, true), false);
  assert.equal(allowedSocket(target.replace('/sign_in?', '/invoke?'), [], 0, true), false);
  assert.equal(allowedSocket('not-a-url', [], 0, true), false);
  assert.equal(maximumDurationMs, 60000);
  assert.equal(connectionOutcome({ socketError: true, socketClosed: true, socketPolicyDenied: true }, true), 'connection_failed_before_upgrade');
  assert.equal(connectionOutcome({ socketPolicyDenied: true }, false), 'offline_gate_verified');
  assert.equal(connectionOutcome({ socketOpened: true, videoFrames: 3, dataChannelsOpen: 1, pixelRange: 40, nonDarkPixels: 1000 }, true), 'rendered_frames_and_open_datachannel');
  assert.deepEqual(safeSocketFailure('Unexpected response code: 403'), { failureClass: 'http_handshake_rejected', failureHTTPStatus: 403 });
  assert.deepEqual(safeSocketFailure('net::ERR_CERT_AUTHORITY_INVALID'), { failureClass: 'ERR_CERT_AUTHORITY_INVALID', failureHTTPStatus: 0 });
  assert.deepEqual(safeSocketFailure('synthetic-secret-value'), { failureClass: 'unclassified_socket_error', failureHTTPStatus: 0 });
  console.log('SELF_TEST: route, credential, one-attempt and offline gates passed.');
}

function browserProbe(AppStreamer, StreamType, configuration, liveMode, gate, selectType) {
  const report = {
    socketAttempts: 0, socketForwarded: 0, socketOpened: false, socketClosed: false,
    socketError: false, socketPolicyDenied: false, requestedRouteMatches: false,
    iceCandidateMessages: 0, descriptionMessages: 0, signalingAcks: 0,
    peerErrors: 0, sdkEvents: 0, sdkStarted: false, sdkFailure: false,
    dataChannelsCreated: 0, dataChannelsOpen: 0, videoFrames: 0,
    videoWidth: 0, videoHeight: 0, pixelRange: 0, nonDarkPixels: 0,
    dataChannelSendsBlocked: 0, dataChannelSendsForwarded: 0,
  };
  const channels = [];
  window.RTCDataChannel.prototype.send = function () {
    report.dataChannelSendsBlocked++;
  };
  const NativeSocket = window.WebSocket;
  window.WebSocket = class extends NativeSocket {
    constructor(address, protocols) {
      report.socketAttempts++;
      report.requestedRouteMatches = gate(String(address), protocols, 0, true);
      if (!gate(String(address), protocols, report.socketForwarded, liveMode)) {
        report.socketPolicyDenied = true;
        throw new Error('Socket blocked by probe policy');
      }
      report.socketForwarded++;
      super(address, protocols);
      this.addEventListener('open', () => { report.socketOpened = true; });
      this.addEventListener('error', () => { report.socketError = true; });
      this.addEventListener('close', () => { report.socketClosed = true; });
      this.addEventListener('message', event => {
        try {
          const message = JSON.parse(event.data);
          if (Object.hasOwn(message, 'ack')) report.signalingAcks++;
          if (message.error) report.peerErrors++;
          if (typeof message.peer_msg?.msg === 'string') {
            const peer = JSON.parse(message.peer_msg.msg);
            if (Object.hasOwn(peer, 'candidate')) report.iceCandidateMessages++;
            if (Object.hasOwn(peer, 'sdp')) report.descriptionMessages++;
          }
        } catch {}
      });
    }
  };
  const NativePeer = window.RTCPeerConnection;
  const trackChannel = channel => {
    channels.push(channel);
    report.dataChannelsCreated++;
  };
  window.RTCPeerConnection = class extends NativePeer {
    constructor(...parameters) {
      super(...parameters);
      this.addEventListener('datachannel', event => trackChannel(event.channel));
    }
    createDataChannel(...parameters) {
      const channel = super.createDataChannel(...parameters);
      trackChannel(channel);
      return channel;
    }
  };
  const video = document.getElementById('remote-video');
  const sample = async () => {
    report.videoFrames = video.getVideoPlaybackQuality().totalVideoFrames;
    report.videoWidth = video.videoWidth;
    report.videoHeight = video.videoHeight;
    report.dataChannelsOpen = channels.filter(channel => channel.readyState === 'open').length;
    if (video.readyState >= 2 && report.videoFrames > 0) {
      try {
        const canvas = document.createElement('canvas');
        canvas.width = 160;
        canvas.height = 90;
        const context = canvas.getContext('2d', { willReadFrequently: true });
        context.drawImage(video, 0, 0, canvas.width, canvas.height);
        const pixels = context.getImageData(0, 0, canvas.width, canvas.height).data;
        let minimum = 255;
        let maximum = 0;
        let nonDark = 0;
        for (let offset = 0; offset < pixels.length; offset += 4) {
          const brightness = (pixels[offset] + pixels[offset + 1] + pixels[offset + 2]) / 3;
          minimum = Math.min(minimum, brightness);
          maximum = Math.max(maximum, brightness);
          if (brightness > 12) nonDark++;
        }
        report.pixelRange = Math.round(maximum - minimum);
        report.nonDarkPixels = nonDark;
      } catch {}
    }
    document.getElementById('state').textContent = report.videoFrames > 0 ? 'Frames received' : report.sdkFailure || report.socketError ? 'Connection failed' : 'Connecting';
    await window.updateProbe({ ...report }).catch(() => {});
  };
  const updated = message => {
    report.sdkEvents++;
    if (message?.status === 'error' || message?.status === 'failure') report.sdkFailure = true;
    sample().catch(() => {});
  };
  window.runProbe = () => {
    setInterval(() => sample().catch(() => {}), 250);
    AppStreamer.connect({
      streamSource: selectType(StreamType),
      streamConfig: {
        ...configuration,
        onStart: message => { report.sdkStarted = true; updated(message); },
        onUpdate: updated,
        onStop: updated,
        onTerminate: updated,
        onCustomEvent: () => {},
      },
    }).catch(() => { report.sdkFailure = true; sample().catch(() => {}); });
  };
  window.stopProbe = () => AppStreamer.stop().catch(() => {});
  window.sampleProbe = sample;
  window.probeReport = report;
}

async function runProbe(live) {
  const dependencyRoot = process.env.DSX_PROBE_DEPENDENCIES || '/root/.cache/dsx-browser-research';
  const sdk = await readFile(`${dependencyRoot}/node_modules/@nvidia/omniverse-webrtc-streaming-library/dist/omniverse-webrtc-streaming-library.js`);
  const { chromium } = await import(pathToFileURL(`${dependencyRoot}/node_modules/playwright/index.mjs`).href);
  const configuration = {
    signalingServer: 'api.ngc.nvidia.com', signalingPort: 443, signalingPath,
    forceWSS: true, authenticate: false, newSession: false, maxReconnects: 0,
    connectivityTimeout: 15000, width: 1280, height: 720, fps: 30,
    videoElementId: 'remote-video', audioElementId: 'remote-audio',
    mic: false, enableAV1Support: false,
  };
  assert.equal(configuration.authenticate, false);
  assert.equal(configuration.maxReconnects, 0);
  assert.equal(Object.hasOwn(configuration, 'accessToken'), false);
  const client = `import { AppStreamer, StreamType } from '/sdk.mjs';
const signalingPath = ${JSON.stringify(signalingPath)};
console.log = console.info = console.debug = console.warn = console.error = () => {};
(${browserProbe.toString()})(AppStreamer, StreamType, ${JSON.stringify(configuration)}, ${JSON.stringify(live)}, ${allowedSocket.toString()}, ${selectedStreamType.toString()});`;
  const document = '<!doctype html><html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>DSX Connection Check</title><style>body{margin:0;background:#f4f6f4;color:#17201d;font:16px monospace}header{padding:16px;border-bottom:1px solid #b9c4bf}video{display:block;width:100%;max-height:80vh;aspect-ratio:16/9;object-fit:contain;background:#111}#state{margin-left:16px;font-size:14px}</style><header>DSX Connection Check<span id="state">Idle</span></header><video id="remote-video" autoplay playsinline muted></video><audio id="remote-audio" autoplay muted></audio><script type="module" src="/client.mjs"></script></html>';
  const server = createServer((request, response) => {
    const assets = {
      '/': ['text/html', document], '/client.mjs': ['text/javascript', client],
      '/sdk.mjs': ['text/javascript', sdk],
    };
    const asset = request.method === 'GET' ? assets[request.url] : undefined;
    if (!asset) { response.writeHead(404).end(); return; }
    response.writeHead(200, { 'Content-Type': asset[0], 'Cache-Control': 'no-store', 'Content-Security-Policy': "default-src 'self'; script-src 'self'; style-src 'unsafe-inline'; connect-src 'self' wss://api.ngc.nvidia.com; media-src 'self' blob:; img-src 'self' data:; worker-src 'none'; frame-src 'none'" });
    response.end(asset[1]);
  });
  let browser;
  let browserServer;
  let browserProcess;
  let context;
  let deadline;
  let deadlineKilledBrowser = false;
  let cleanupFailed = false;
  let startedAt;
  let lastReport = {};
  let blockedHTTP = 0;
  let pageErrors = 0;
  let outcome = 'initialization_failed';
  const handshake = { networkSocketCount: 0, handshakeStatus: 0, localOriginVerified: false, authorizationHeaderPresent: false, cookieHeaderPresent: false, subprotocolHeaderPresent: false, failureClass: 'not_captured', failureHTTPStatus: 0 };
  try {
    await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
    const origin = `http://127.0.0.1:${server.address().port}`;
    browserServer = await chromium.launchServer({ headless: true, host: '127.0.0.1' });
    browserProcess = browserServer.process();
    browser = await chromium.connect(browserServer.wsEndpoint());
    context = await browser.newContext({ serviceWorkers: 'block', viewport: { width: 1280, height: 850 } });
    await context.route('**/*', async route => {
      const request = route.request();
      const url = new URL(request.url());
      if (request.method() === 'GET' && url.origin === origin && ['/', '/client.mjs', '/sdk.mjs'].includes(url.pathname) && !url.search) {
        await route.continue();
      } else {
        blockedHTTP++;
        await route.abort('blockedbyclient');
      }
    });
    const page = await context.newPage();
    page.on('pageerror', () => { pageErrors++; });
    const diagnostics = await context.newCDPSession(page);
    await diagnostics.send('Network.enable');
    diagnostics.on('Network.webSocketCreated', event => {
      if (allowedSocket(event.url, [], 0, true)) handshake.networkSocketCount++;
    });
    diagnostics.on('Network.webSocketWillSendHandshakeRequest', event => {
      const headers = Object.fromEntries(Object.entries(event.request.headers).map(([name, value]) => [name.toLowerCase(), value]));
      handshake.localOriginVerified = headers.origin === origin;
      handshake.authorizationHeaderPresent = Object.hasOwn(headers, 'authorization');
      handshake.cookieHeaderPresent = Object.hasOwn(headers, 'cookie');
      handshake.subprotocolHeaderPresent = Object.hasOwn(headers, 'sec-websocket-protocol');
    });
    diagnostics.on('Network.webSocketHandshakeResponseReceived', event => {
      const status = event.response.status;
      if (Number.isInteger(status) && status >= 100 && status <= 599) handshake.handshakeStatus = status;
    });
    diagnostics.on('Network.webSocketFrameError', event => {
      Object.assign(handshake, safeSocketFailure(event.errorMessage));
    });
    await page.exposeFunction('updateProbe', report => {
      lastReport = Object.fromEntries(Object.entries(report).filter(([key, value]) =>
        ['socketAttempts', 'socketForwarded', 'socketOpened', 'socketClosed', 'socketError', 'socketPolicyDenied', 'requestedRouteMatches', 'iceCandidateMessages', 'descriptionMessages', 'signalingAcks', 'peerErrors', 'sdkEvents', 'sdkStarted', 'sdkFailure', 'dataChannelsCreated', 'dataChannelsOpen', 'videoFrames', 'videoWidth', 'videoHeight', 'pixelRange', 'nonDarkPixels', 'dataChannelSendsBlocked', 'dataChannelSendsForwarded'].includes(key) &&
        (typeof value === 'boolean' || (typeof value === 'number' && Number.isFinite(value)))));
    });
    await page.goto(origin, { waitUntil: 'load', timeout: 15000 });
    await page.waitForFunction(() => typeof window.runProbe === 'function', undefined, { timeout: 10000 });
    startedAt = Date.now();
    deadline = setTimeout(() => {
      deadlineKilledBrowser = true;
      browserProcess.kill('SIGKILL');
    }, live ? maximumDurationMs : 5000);
    console.log(JSON.stringify({ mode: live ? 'live' : 'offline', origin, maximumDurationMs: live ? maximumDurationMs : 5000 }));
    await page.evaluate(() => window.runProbe());
    await page.waitForFunction(() => {
      const report = window.probeReport;
      return report.socketPolicyDenied || report.socketError || report.socketClosed || report.sdkFailure ||
        (report.videoFrames >= 2 && report.dataChannelsOpen > 0 && report.pixelRange > 8 && report.nonDarkPixels > 100);
    }, undefined, { timeout: live ? 57000 : 2500 }).catch(() => {});
    await page.evaluate(() => window.sampleProbe()).catch(() => {});
    outcome = connectionOutcome(lastReport, live);
    if (outcome === 'rendered_frames_and_open_datachannel') {
      await page.screenshot({ path: `${dependencyRoot}/dsx-connection.png` }).catch(() => {});
    }
    await Promise.race([page.evaluate(() => window.stopProbe()).catch(() => {}), new Promise(resolve => setTimeout(resolve, 1000))]);
    if (!live) {
      assert.equal(lastReport.socketForwarded, 0);
      assert.equal(lastReport.requestedRouteMatches, true);
      assert.equal(lastReport.socketPolicyDenied, true);
    }
    assert.equal(lastReport.dataChannelSendsForwarded, 0);
  } finally {
    await context?.close().catch(() => { cleanupFailed = true; });
    await browser?.close().catch(() => { cleanupFailed = true; });
    await browserServer?.close().catch(() => { cleanupFailed = true; });
    clearTimeout(deadline);
    const clientClosed = Boolean(browserProcess && (browserProcess.exitCode !== null || browserProcess.signalCode !== null));
    server.closeAllConnections();
    await new Promise(resolve => server.close(resolve));
    console.log(JSON.stringify({ outcome, ...lastReport, ...handshake, blockedHTTP, pageErrors, clientClosed, cleanupFailed, deadlineKilledBrowser, clientLifetimeMs: startedAt ? Date.now() - startedAt : 0, apiKeyUsed: false }));
    assert.equal(clientClosed, true);
    assert.equal(cleanupFailed, false);
  }
}

try {
  assert.ok(process.argv.slice(2).every(argument => ['--self-test', '--offline', '--live'].includes(argument)));
  assert.ok(!process.argv.includes('--live') || (!process.argv.includes('--offline') && !process.argv.includes('--self-test')));
  selfTest();
  if (!process.argv.includes('--self-test')) await runProbe(process.argv.includes('--live'));
} catch {
  console.log('PROBE_FAILED: dependency, initialization or safety check failed; raw error suppressed.');
  process.exitCode = 1;
}
