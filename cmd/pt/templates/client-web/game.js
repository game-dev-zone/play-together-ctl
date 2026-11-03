// 「一起按」前端：用玩聚瀏覽器 SDK 握手 → 把啟動碼交給自己的遊戲伺服器 → 只送操作、畫面照伺服器的狀態顯示。
// 直接用瀏覽器打開（不在玩聚裡）時，SDK 自動進入模擬模式。
//
// 斷線重連：網路中斷時顯示「重新連線中」並持續重試，恢復後自動接上；重新整理或換裝置時，
// 平台會給新的啟動碼，伺服器把你接回原座位。token 失效（例如在別的分頁開了遊戲）時請玩家重新整理。
const cfg = window.GAME_CONFIG;
const $ = (id) => document.getElementById(id);
let token = '';
let greeting = '';
let failures = 0;
let poll = 0;

// 回傳 { status, data }；網路錯誤時 status 是 0
async function post(path, body) {
  try {
    const res = await fetch(cfg.serverUrl + path, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
    return { status: res.status, data: await res.json().catch(() => ({})) };
  } catch {
    return { status: 0, data: {} };
  }
}

function render(state) {
  const list = $('players');
  list.replaceChildren(...state.players.map((p) => {
    const li = document.createElement('li');
    li.textContent = `${p.name}：${p.taps} 下${p.online ? '' : '（離線）'}`; // 暱稱一律當純文字
    return li;
  }));
  $('start').hidden = state.playing;
  $('tap').hidden = !state.playing;
  $('timer').textContent = state.playing ? `剩 ${Math.ceil(state.secondsLeft)} 秒` : state.winner ? `贏家：${state.winner}` : '';
}

// 連線失效：停止輪詢，請玩家重新整理（平台會給新的啟動碼，回到原座位）
function expired(message) {
  clearInterval(poll);
  token = '';
  $('start').hidden = true;
  $('tap').hidden = true;
  $('status').textContent = message;
}

async function act(action) {
  if (!token) return;
  const { status, data } = await post('/action', { token, action });
  if (status === 200) {
    failures = 0;
    $('status').textContent = greeting;
    render(data);
  } else if (status === 401) {
    expired('連線已失效（可能在別的分頁或裝置開了遊戲）：請重新整理頁面');
  } else if (++failures >= 2) {
    $('status').textContent = '連線中斷，重新連線中…'; // 網路恢復後下一次輪詢就會接上
  }
}

const JOIN_ERRORS = {
  409: '這一局已經重新開始，請重新整理頁面',
  503: '暫時無法加入，請稍後重新整理頁面',
};

async function main() {
  const pt = await PlayTogether.init({ platformOrigins: cfg.platformOrigins });
  // 啟動碼只用一次、不要存起來；網路錯誤時重試幾次（伺服器沒收到就還能用）
  let res;
  for (let i = 0; i < 3; i++) {
    res = await post('/join', { launchCode: pt.launch.code });
    if (res.status !== 0 && res.status < 500) break;
    await new Promise((r) => setTimeout(r, 1000 * (i + 1)));
  }
  if (res.status !== 200) {
    $('status').textContent = JOIN_ERRORS[res.status] || (res.status === 403 ? '無法加入這個房間，請重新整理頁面' : '連不上遊戲伺服器，請稍後重新整理頁面');
    return;
  }
  token = res.data.token;
  greeting = `你好，${res.data.name}`;
  $('status').textContent = greeting;
  render(res.data.state); // 重新連線後馬上看到目前的狀態
  $('start').onclick = () => act('start');
  $('tap').onclick = () => act('tap');
  pt.on('resume', () => act('state')); // 從背景回來時立刻更新
  pt.ready(); // 平台會收起載入畫面
  poll = setInterval(() => act('state'), 500);
}

main().catch((e) => {
  $('status').textContent = '無法開始：' + e.message;
});
