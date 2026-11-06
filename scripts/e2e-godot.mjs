// Godot 範本的瀏覽器測試（CI 用）：打開 pt dev 提供的網頁版（網址帶 pt_dev 時經過本機假平台，否則是 -mock 模擬模式），確認 Godot 遊戲握手、
// 向遊戲伺服器兌換啟動碼並收到房間狀態（遊戲把狀態寫到 window.__ptGame）。
// 用法：node scripts/e2e-godot.mjs <網址> [截圖.png]；需要 Chrome（CHROME 環境變數可指定路徑）
import { spawn } from 'node:child_process';
import { existsSync, mkdtempSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

const [url = 'http://127.0.0.1:8090/', out] = process.argv.slice(2);
const chromePath = process.env.CHROME || ['/Applications/Google Chrome.app/Contents/MacOS/Google Chrome', '/usr/bin/google-chrome', '/usr/bin/chromium']
  .find((p) => existsSync(p));
if (!chromePath) throw new Error('找不到 Chrome（設定 CHROME 環境變數）');
const port = 9400 + Math.floor(Math.random() * 400);
// 無頭 Chrome 沒有 GPU：用 SwiftShader 提供 WebGL2（Godot 的 Compatibility 算繪器需要）
const chrome = spawn(chromePath, ['--headless=new', '--no-sandbox', '--use-angle=swiftshader', '--enable-unsafe-swiftshader',
  `--remote-debugging-port=${port}`, '--user-data-dir=' + mkdtempSync(join(tmpdir(), 'pt-e2e-')), 'about:blank'], { stdio: 'ignore' });
const wait = (ms) => new Promise((r) => setTimeout(r, ms));
const fail = (msg) => { console.error('失敗：' + msg); chrome.kill(); process.exit(1); };
let target;
for (let i = 0; i < 100 && !target; i++) {
  await wait(200);
  try { target = (await (await fetch(`http://127.0.0.1:${port}/json`)).json()).find((t) => t.type === 'page'); } catch {}
}
if (!target) fail('Chrome 沒有啟動');
const ws = new WebSocket(target.webSocketDebuggerUrl);
await new Promise((r) => (ws.onopen = r));
let id = 0; const pending = new Map(); const logs = [];
ws.onmessage = (e) => {
  const m = JSON.parse(e.data);
  if (m.method === 'Runtime.consoleAPICalled') logs.push(m.params.args.map((a) => a.value ?? a.description).join(' '));
  if (m.method === 'Runtime.exceptionThrown') logs.push(m.params.exceptionDetails.text);
  pending.get(m.id)?.(m.result ?? m.error);
};
const send = (method, params = {}) => new Promise((r) => { pending.set(++id, r); ws.send(JSON.stringify({ id, method, params })); });
await send('Emulation.setDeviceMetricsOverride', { width: 390, height: 844, deviceScaleFactor: 1, mobile: true });
await send('Runtime.enable');
await send('Page.navigate', { url });
let state = null;
for (let i = 0; i < 120 && !state?.joined; i++) {
  await wait(500);
  const r = await send('Runtime.evaluate', { expression: 'window.__ptGame ?? null', returnByValue: true });
  if (r.result?.value) state = JSON.parse(r.result.value);
}
await wait(1000);
if (out) {
  const shot = await send('Page.captureScreenshot', { format: 'png' });
  writeFileSync(out, Buffer.from(shot.data, 'base64'));
  console.log('截圖：' + out);
}
if (!state?.joined) fail('Godot 遊戲沒有完成握手與兌換；主控台：\n  ' + logs.slice(-15).join('\n  '));
if (state.players < 1) fail('房間狀態沒有玩家：' + JSON.stringify(state));
console.log('通過：Godot 網頁版在模擬模式握手、兌換啟動碼、收到房間狀態 ' + JSON.stringify(state));
chrome.kill();
