// 預設音效：用 Web Audio API 即時合成，不放任何音檔（沒有授權問題、不佔上傳大小）。
// 換成自己的音效時，來源規則見 PLAYTOGETHER.md「音效與音樂」，並記錄在 art/SOURCES.md。
//
// - 瀏覽器要等玩家「在遊戲裡」點一下才能發聲（平台外殼的點擊不算）：在按鈕的點擊事件裡先呼叫 sfx.unlock()。
// - 平台的「遊戲聲音」開關透過 SDK 的 mute 事件通知：sfx.setMuted(true) 後所有音效都不發聲。
// - iPhone 從背景回來時聲音可能又被暫停：下一次點擊時 unlock() 會再恢復。
const sfx = (() => {
  let ctx = null;
  let master = null;
  let muted = false;
  const VOLUME = 0.25; // 整體音量：不要蓋過玩聚語音

  function unlock() {
    const AC = window.AudioContext || window.webkitAudioContext;
    if (!AC) return;
    if (!ctx) {
      ctx = new AC();
      master = ctx.createGain();
      master.gain.value = muted ? 0 : VOLUME;
      master.connect(ctx.destination);
    }
    if (ctx.state !== 'running') ctx.resume().catch(() => {});
  }

  function setMuted(m) {
    muted = m;
    if (master) master.gain.setValueAtTime(m ? 0 : VOLUME, ctx.currentTime);
  }

  // 一個短音：頻率從 from 滑到 to，帶淡入淡出避免爆音
  function tone(from, to, duration, type = 'sine', delay = 0) {
    if (!ctx || muted || ctx.state !== 'running') return;
    const t = ctx.currentTime + delay;
    const osc = ctx.createOscillator();
    const env = ctx.createGain();
    osc.type = type;
    osc.frequency.setValueAtTime(from, t);
    osc.frequency.exponentialRampToValueAtTime(to, t + duration);
    env.gain.setValueAtTime(0.0001, t);
    env.gain.exponentialRampToValueAtTime(1, t + 0.01);
    env.gain.exponentialRampToValueAtTime(0.0001, t + duration);
    osc.connect(env).connect(master);
    osc.start(t);
    osc.stop(t + duration + 0.02);
  }

  return {
    unlock,
    setMuted,
    tap: () => tone(660, 990, 0.08, 'triangle'),
    tick: () => tone(880, 880, 0.06, 'square'),
    start: () => [523, 659, 784].forEach((f, i) => tone(f, f, 0.12, 'triangle', i * 0.1)),
    end: () => [784, 659, 523, 1047].forEach((f, i) => tone(f, f, i === 3 ? 0.35 : 0.12, 'triangle', i * 0.12)),
  };
})();
