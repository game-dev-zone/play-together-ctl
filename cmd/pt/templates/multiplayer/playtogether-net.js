// 玩聚遊戲伺服器連線小工具（pt add server 產生；這個檔案是你的，可以自由修改）。
//
// 做的事和 pt new 範本的 web/game.js 一樣：握手後把啟動碼交給遊戲伺服器換 token、只送「操作」、
// 定時輪詢狀態、網路中斷時自動重試、token 失效時請玩家重新整理。分數、勝負、流程一律由伺服器決定。
//
// 用法（先載入玩聚瀏覽器 SDK 與這個檔案）：
//
//   const pt = await PlayTogether.init({ platformOrigins: ['https://（玩聚平台網址）'] });
//   const net = await PlayTogetherNet.connect(pt, {
//     serverUrl: '',                 // 自己部署的遊戲伺服器網址；玩聚代管或本機 pt dev 留空
//     onState: (state) => render(state),  // 伺服器 View 回傳的狀態（只有這位玩家可以看的資料）
//     onStatus: (kind, message) => {},     // kind：ok、reconnecting、expired、failed
//   });
//   button.onclick = () => net.act('move', { x: 1, y: 2 });  // 送操作（data 選填，要是 JSON）
//   pt.ready();
//
// 暱稱等玩家輸入的文字一律用 textContent 顯示，不要用 innerHTML。
(function () {
  const JOIN_ERRORS = {
    403: '無法加入這個房間，請重新整理頁面',
    409: '這一局已經重新開始，請重新整理頁面',
    503: '暫時無法加入，請稍後重新整理頁面',
  };

  async function connect(pt, opts) {
    const onState = opts.onState || (() => {});
    const onStatus = opts.onStatus || (() => {});
    const pollMs = opts.pollMs || 500;
    // 玩聚代管時，每個房間的伺服器網址由平台在握手時給（不能用 location.host）
    let server = (pt.launch.serverUrl || opts.serverUrl || '').replace(/\/+$/, '');
    let token = '';
    let failures = 0;
    let timer = 0;

    async function post(path, body) {
      try {
        const res = await fetch(server + path, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(body),
        });
        return { status: res.status, data: await res.json().catch(() => ({})) };
      } catch {
        return { status: 0, data: {} };
      }
    }

    function stop() {
      clearInterval(timer);
      token = '';
    }

    // 啟動碼只用一次、不要存起來；網路錯誤時重試幾次（伺服器沒收到就還能用）
    let res;
    for (let i = 0; i < 3; i++) {
      res = await post('/join', { launchCode: pt.launch.code });
      if (res.status !== 0 && res.status < 500) break;
      await new Promise((r) => setTimeout(r, 1000 * (i + 1)));
    }
    if (res.status !== 200) {
      onStatus('failed', JOIN_ERRORS[res.status] || '連不上遊戲伺服器，請稍後重新整理頁面');
      throw new Error('join failed: ' + res.status);
    }
    token = res.data.token;
    onStatus('ok', '');
    onState(res.data.state); // 重新連線後馬上看到目前的狀態

    // 送一個操作；伺服器回傳最新狀態。400 是遊戲規則拒絕（回傳 { error }），其他錯誤由 onStatus 通知
    async function act(action, data) {
      if (!token) return null;
      const body = { token, action };
      if (data !== undefined) body.data = data;
      const r = await post('/action', body);
      if (r.status === 200) {
        if (failures > 0) onStatus('ok', '');
        failures = 0;
        onState(r.data);
        return r.data;
      }
      if (r.status === 401) {
        stop();
        onStatus('expired', '連線已失效（可能在別的分頁或裝置開了遊戲）：請重新整理頁面');
      } else if (r.status === 400) {
        return { error: r.data.error || 'rejected' };
      } else if (++failures >= 2) {
        onStatus('reconnecting', '連線中斷，重新連線中…'); // 網路恢復後下一次輪詢就會接上
      }
      return null;
    }

    timer = setInterval(() => act('state'), pollMs);
    pt.on('resume', () => act('state')); // 從背景回來時立刻更新
    return { act, stop, playerId: res.data.playerId, name: res.data.name };
  }

  window.PlayTogetherNet = { connect };
})();
