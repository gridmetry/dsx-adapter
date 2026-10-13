(async () => {
  if (location.origin !== 'https://build.nvidia.com' ||
      !location.pathname.endsWith('/omniverse-dsx-blueprint-for-ai-factories')) {
    throw new Error('Run this loader in the NVIDIA DSX blueprint page');
  }

  const response = await fetch('http://127.0.0.1:28090/visualization-client.js', {
    credentials: 'omit',
    signal: AbortSignal.timeout(5000),
  });
  if (!response.ok || !response.headers.get('content-type')?.startsWith('text/javascript')) {
    throw new Error('Adapter receiver script unavailable');
  }
  const source = await response.text();
  const digest = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(source));
  const fingerprint = Array.from(new Uint8Array(digest))
    .map(value => value.toString(16).padStart(2, '0')).join('');
  if (fingerprint !== '0d51b5bd1edb8330954c5ade951653976ddaf1e8c2465554a31a3631743809c4') {
    throw new Error('Receiver fingerprint mismatch; update the loader for the reviewed deployment');
  }

  await window.__gridmetryReceiver?.disconnect();
  document.getElementById('gridmetry-dsx-receiver')?.remove();
  (0, eval)(source);
  window.__gridmetryReceiver = window.GridmetryDSXReceiver.mount();
  return window.__gridmetryReceiver.report();
})();
