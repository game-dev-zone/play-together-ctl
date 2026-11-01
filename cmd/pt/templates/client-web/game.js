// 「一起按」前端：用玩聚瀏覽器 SDK 握手 → 把啟動碼交給自己的遊戲伺服器 → 只送操作、畫面照伺服器的狀態顯示。
// 直接用瀏覽器打開（不在玩聚裡）時，SDK 自動進入模擬模式。
const cfg = window.GAME_CONFIG;
const $ = (id) => document.getElementById(id);
let token = '';

async function post(path, body) {
  const res = await fetch(cfg.serverUrl + path, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
  if (!res.ok) throw new Error(path + ' ' + res.status);
  return res.json();
}

function render(state) {
  const list = $('players');
  list.replaceChildren(...state.players.map((p) => {
    const li = document.createElement('li');
    li.textContent = `${p.name}：${p.taps} 下`; // 暱稱一律當純文字
    return li;
  }));
  $('start').hidden = state.playing;
  $('tap').hidden = !state.playing;
  $('timer').textContent = state.playing ? `剩 ${Math.ceil(state.secondsLeft)} 秒` : state.winner ? `贏家：${state.winner}` : '';
}

async function act(action) {
  try {
    render(await post('/action', { token, action }));
  } catch (e) {
    $('status').textContent = '連線中斷，請重新整理';
  }
}

async function main() {
  const pt = await PlayTogether.init({ platformOrigins: cfg.platformOrigins });
  const me = await post('/join', { launchCode: pt.launch.code }); // 啟動碼只用一次、不要存起來
  token = me.token;
  $('status').textContent = `你好，${me.name}`;
  $('start').onclick = () => act('start');
  $('tap').onclick = () => act('tap');
  pt.ready(); // 平台會收起載入畫面
  await act('state'); // 一進房就顯示房間裡的玩家
  setInterval(() => act('state'), 500);
}

main().catch((e) => {
  $('status').textContent = '無法開始：' + e.message;
});
