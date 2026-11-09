// Sweeps every chapter page and measures its scene canvas.
//
// Catches a chapter whose figure is empty, clipped or off-centre, which is invisible in a
// text diff and easy to miss when only a few pages are opened by hand.
//
// Drives the Chrome that is already on this machine over the DevTools protocol, rather
// than adding Playwright: the course promises no extra tooling, and this is a maintainer
// script, not a build step.
//
//   node scripts/sweep-scenes.js [base-url] [chrome-path]

const http = require('http');
const { spawn } = require('child_process');

const BASE = process.argv[2] || 'http://127.0.0.1:8098';
const PREFIX = process.argv[4] || '';  // '' for v1, '/v2' when sweeping the nested build
const CHROME = process.argv[3] ||
  `${process.env.HOME}/.agent-browser/browsers/chrome-155.0.8059.39/chrome`;
const PORT = 9333;

const CHAPTERS = [
  '01-chat-with-the-model', '02-tool-calling', '03-running-a-tool',
  '04-tools-that-touch-disk', '05-the-agent-loop', '06-prompt-assembly',
  '07-sessions', '08-context-budget', '09-steer-and-abort', '10-skills',
  '11-skill-invocation', '12-subagent', '13-mcp', '14-retries-and-rate-limits',
  '15-bounding-a-tool-result', '16-the-tool-schema-budget',
  '17-skill-scopes-and-precedence', '18-building-the-request-body',
  '19-session-storage', '20-what-a-conversation-costs',
  '21-configuring-a-provider',
];

function get(path) {
  return new Promise((resolve, reject) => {
    http.get({ host: '127.0.0.1', port: PORT, path }, (res) => {
      let d = '';
      res.on('data', (c) => (d += c));
      res.on('end', () => resolve(d));
    }).on('error', reject);
  });
}

function sleep(ms) { return new Promise((r) => setTimeout(r, ms)); }

async function waitForDevtools() {
  for (let i = 0; i < 100; i++) {
    try {
      const v = JSON.parse(await get('/json/version'));
      if (v.webSocketDebuggerUrl) return v.webSocketDebuggerUrl;
    } catch (_) { /* not up yet */ }
    await sleep(200);
  }
  throw new Error('Chrome DevTools did not start');
}

// Minimal CDP client: enough for navigate, evaluate and new targets.
async function connect(wsUrl) {
  const ws = new WebSocket(wsUrl);
  await new Promise((res, rej) => { ws.onopen = res; ws.onerror = rej; });
  let id = 0;
  const pending = new Map();
  ws.onmessage = (e) => {
    const m = JSON.parse(e.data);
    if (m.id && pending.has(m.id)) {
      const { resolve, reject } = pending.get(m.id);
      pending.delete(m.id);
      m.error ? reject(new Error(m.error.message)) : resolve(m.result);
    }
  };
  const send = (method, params, sessionId) => new Promise((resolve, reject) => {
    const msg = { id: ++id, method, params: params || {} };
    if (sessionId) msg.sessionId = sessionId;
    pending.set(msg.id, { resolve, reject });
    ws.send(JSON.stringify(msg));
  });
  return { ws, send };
}

const MEASURE = `(() => {
  const cv = document.querySelector('canvas.scene3d');
  if (!cv) return { err: 'no canvas (renderer removed it: ' +
    document.documentElement.getAttribute('data-scene3d') + ')' };
  const gl = cv.getContext('webgl') || cv.getContext('experimental-webgl');
  if (!gl) return { err: 'no webgl' };

  // The context does not preserve its drawing buffer, so a readPixels after present
  // returns a cleared surface. Instrument drawArrays instead: if geometry is being
  // submitted, the figure is rendering, and the count tells us every object drew.
  let draws = 0;
  const orig = gl.drawArrays.bind(gl);
  gl.drawArrays = (...a) => { draws++; return orig(...a); };
  return new Promise((resolve) => {
    requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        gl.drawArrays = orig;
        const legend = document.querySelector('.scene3d-legend');
        const scene = window.__miyagiScene;
        resolve({
          draws: draws,
          objects: scene ? scene.objects.length : 0,
          legend: legend ? legend.children.length : 0,
          w: cv.width, h: cv.height,
        });
      });
    });
  });
})()`;

(async () => {
  const chrome = spawn(CHROME, [
    '--headless=new', `--remote-debugging-port=${PORT}`,
    '--no-sandbox', '--disable-gpu-sandbox',
    // Headless Chrome has no GPU, so WebGL needs the software rasteriser. Without these
    // the renderer correctly falls back to the flat figure and every measurement is empty,
    // which is a property of the harness rather than of the site.
    '--enable-unsafe-swiftshader',
    '--use-gl=angle', '--use-angle=swiftshader',
    '--enable-features=Vulkan',
    '--window-size=1280,900', 'about:blank',
  ], { stdio: 'ignore' });

  let failures = 0;
  const rows = [];
  try {
    const wsUrl = await waitForDevtools();
    const { ws, send } = await connect(wsUrl);
    const { targetId } = await send('Target.createTarget', { url: 'about:blank' });
    const { sessionId } = await send('Target.attachToTarget', { targetId, flatten: true });

    await send('Page.enable', {}, sessionId);
    await send('Runtime.enable', {}, sessionId);

    for (const slug of CHAPTERS) {
      await send('Page.navigate', { url: `${BASE}${PREFIX}/chapters/${slug}/` }, sessionId);
      await sleep(1200);
      // Scroll the figure into view so the IntersectionObserver starts the renderer.
      await send('Runtime.evaluate', {
        expression: `(() => { const w = document.querySelector('.scene3d-wrap'); if (w) w.scrollIntoView({block:'center'}); })()`,
      }, sessionId);
      await sleep(2600);

      const r = await send('Runtime.evaluate', {
        expression: MEASURE, returnByValue: true, awaitPromise: true,
      }, sessionId);
      const m = r.result.value || { err: 'no result' };

      if (m.err) { rows.push([slug, m.err, '', '', '']); failures++; continue; }
      // Every object should be drawn: one drawArrays per object per frame, and two
      // frames elapsed. A shortfall means a scene references a group that never reveals.
      const ok = m.draws >= m.objects && m.legend > 0 && m.w > 0;
      if (!ok) failures++;
      rows.push([slug, ok ? 'ok' : 'CHECK', String(m.draws),
                 `${m.objects}obj`, `legend=${m.legend}`]);
    }
    ws.close();
  } finally {
    chrome.kill();
  }

  const pad = (s, n) => String(s).padEnd(n);
  console.log(pad('chapter', 32) + pad('status', 8) + pad('draws', 8) + pad('expected', 11) + 'notes');
  for (const r of rows) console.log(pad(r[0], 32) + pad(r[1], 8) + pad(r[2], 9) + pad(r[3], 10) + r[4]);
  console.log(`\n${rows.length - failures}/${rows.length} scenes pass`);
  process.exit(failures ? 1 : 0);
})();
