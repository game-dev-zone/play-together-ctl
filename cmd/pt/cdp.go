package main

// 透過管道（--remote-debugging-pipe）控制本機的無頭 Chrome：Chrome DevTools Protocol 的 JSON 訊息，
// Chrome 從 fd 3 讀指令、寫回應到 fd 4，每則訊息以 \0 結尾。只用標準函式庫（不需要 WebSocket）。
// pt store screenshot 用它模擬手機畫面、在真實時間裡擷取同一個頁面的多張截圖。

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

type cdpMessage struct {
	ID        int             `json:"id,omitempty"`
	Method    string          `json:"method,omitempty"`
	Params    json.RawMessage `json:"params,omitempty"`
	Result    json.RawMessage `json:"result,omitempty"`
	SessionID string          `json:"sessionId,omitempty"`
	Error     *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// cdpConn 是一條 CDP 連線（一個 Chrome 行程）。
type cdpConn struct {
	w       io.Writer
	mu      sync.Mutex
	nextID  int
	pending map[int]chan cdpMessage
	done    chan struct{} // Chrome 關掉管道（結束或當掉）時關閉
	crashed string        // 頁面當掉的事件（Inspector.targetCrashed 等）
}

func newCDPConn(w io.Writer, r io.Reader) *cdpConn {
	c := &cdpConn{w: w, pending: map[int]chan cdpMessage{}, done: make(chan struct{})}
	go c.readLoop(r)
	return c
}

func (c *cdpConn) readLoop(r io.Reader) {
	defer close(c.done)
	br := bufio.NewReaderSize(r, 1<<20)
	for {
		raw, err := br.ReadBytes(0)
		if len(raw) > 1 {
			var m cdpMessage
			if json.Unmarshal(bytes.TrimSuffix(raw, []byte{0}), &m) == nil {
				c.mu.Lock()
				if m.ID != 0 {
					if ch, ok := c.pending[m.ID]; ok {
						ch <- m
						delete(c.pending, m.ID)
					}
				} else if m.Method == "Inspector.targetCrashed" || m.Method == "Target.targetCrashed" {
					c.crashed = m.Method
				}
				c.mu.Unlock()
			}
		}
		if err != nil {
			return
		}
	}
}

// call 送出一個指令並等回應；session 是空字串時送給瀏覽器本身。
func (c *cdpConn) call(ctx context.Context, session, method string, params any, out any) error {
	c.mu.Lock()
	c.nextID++
	id := c.nextID
	ch := make(chan cdpMessage, 1)
	c.pending[id] = ch
	c.mu.Unlock()
	msg := map[string]any{"id": id, "method": method}
	if params != nil {
		msg["params"] = params
	}
	if session != "" {
		msg["sessionId"] = session
	}
	raw, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	if _, err := c.w.Write(append(raw, 0)); err != nil {
		return fmt.Errorf("%s：Chrome 已經結束（%v）", method, err)
	}
	select {
	case m := <-ch:
		if m.Error != nil {
			return fmt.Errorf("%s：%s", method, m.Error.Message)
		}
		if out != nil {
			return json.Unmarshal(m.Result, out)
		}
		return nil
	case <-c.done:
		return fmt.Errorf("%s：Chrome 意外結束", method)
	case <-ctx.Done():
		return fmt.Errorf("%s：逾時", method)
	}
}

// shotPlan 是一次擷取的設定：同一個頁面先等 wait，之後每隔 interval 再拍，共 n 張。
type shotPlan struct {
	URL          string
	CSSW, CSSH   int     // 遊戲看到的手機視窗（CSS px）
	Scale        float64 // devicePixelRatio：輸出 = CSS × Scale
	Wait         time.Duration
	Interval     time.Duration
	N            int
	Clip         *[2]float64
	ExtraTimeout time.Duration // 除了等待時間以外，啟動與擷取最多再花多久（預設 60 秒）
}

func (sp shotPlan) timeout() time.Duration {
	extra := sp.ExtraTimeout
	if extra == 0 {
		extra = 60 * time.Second
	}
	return sp.Wait + time.Duration(max(sp.N-1, 0))*sp.Interval + extra
}

// chromeArgs 是無頭 Chrome 的參數（CDP 走 fd 3／4 的管道）。
func chromeArgs(userDir string) []string {
	return []string{
		"--headless=new",
		"--remote-debugging-pipe",
		"--user-data-dir=" + userDir,
		"--no-first-run", "--no-default-browser-check", "--disable-extensions", "--disable-sync",
		"--disable-background-networking", "--disable-component-update", "--mute-audio",
		"--hide-scrollbars",
		// 計時器與動畫照真實時間跑（無頭分頁不要被當成背景分頁降速）
		"--disable-background-timer-throttling", "--disable-renderer-backgrounding", "--disable-backgrounding-occluded-windows",
		// 沒有 GPU 的環境（CI、遠端主機）也能畫 WebGL
		"--use-angle=swiftshader", "--enable-unsafe-swiftshader",
		"about:blank",
	}
}

// captureWithChrome 啟動無頭 Chrome、模擬手機打開網址，在真實時間裡擷取 n 張截圖，每張交給 onShot（第幾張、PNG）。
// onShot 回傳錯誤時停止。整體逾時（等待時間＋60 秒）後一定結束 Chrome。
func captureWithChrome(chrome string, sp shotPlan, onShot func(i int, png []byte) error) (err error) {
	if runtime.GOOS == "windows" {
		return errors.New("Windows 還不支援 pt store screenshot（需要以管道控制 Chrome）：請用瀏覽器的手機模擬模式手動截圖")
	}
	userDir, err := os.MkdirTemp("", "pt-chrome-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(userDir)
	ctx, cancel := context.WithTimeout(context.Background(), sp.timeout())
	defer cancel()

	// toChrome：我們寫、Chrome 從 fd 3 讀；fromChrome：Chrome 寫到 fd 4、我們讀
	toChromeR, toChromeW, err := os.Pipe()
	if err != nil {
		return err
	}
	fromChromeR, fromChromeW, err := os.Pipe()
	if err != nil {
		toChromeR.Close()
		toChromeW.Close()
		return err
	}
	defer toChromeW.Close()
	defer fromChromeR.Close()
	cmd := exec.CommandContext(ctx, chrome, chromeArgs(userDir)...)
	cmd.ExtraFiles = []*os.File{toChromeR, fromChromeW}
	var stderr limitedBuffer
	cmd.Stdout, cmd.Stderr = &stderr, &stderr
	cmd.WaitDelay = 5 * time.Second
	startErr := cmd.Start()
	toChromeR.Close()
	fromChromeW.Close()
	if startErr != nil {
		return fmt.Errorf("無法啟動 Chrome（%s）：%w", chrome, startErr)
	}
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()
	c := newCDPConn(toChromeW, fromChromeR)
	defer func() {
		// 正常關閉；關不掉就強制結束（CommandContext 在 cancel 時會砍掉行程）
		closeCtx, closeCancel := context.WithTimeout(context.Background(), 3*time.Second)
		_ = c.call(closeCtx, "", "Browser.close", nil, nil)
		closeCancel()
		select {
		case <-exited:
		case <-time.After(5 * time.Second):
			cancel()
			<-exited
		}
	}()
	fail := func(e error) error {
		select {
		case <-c.done:
			// Chrome 自己結束了：附上它最後的輸出
			return fmt.Errorf("Chrome 異常結束：%v\n%s", e, stderr.tail(5))
		default:
		}
		if ctx.Err() != nil {
			return fmt.Errorf("超過 %s 還沒完成（遊戲一直在載入？）：%v", sp.timeout(), e)
		}
		c.mu.Lock()
		crashed := c.crashed
		c.mu.Unlock()
		if crashed != "" {
			return fmt.Errorf("遊戲頁面當掉了（%s）：%v", crashed, e)
		}
		return e
	}

	var target struct {
		TargetID string `json:"targetId"`
	}
	if err := c.call(ctx, "", "Target.createTarget", map[string]any{"url": "about:blank"}, &target); err != nil {
		return fail(err)
	}
	var attached struct {
		SessionID string `json:"sessionId"`
	}
	if err := c.call(ctx, "", "Target.attachToTarget", map[string]any{"targetId": target.TargetID, "flatten": true}, &attached); err != nil {
		return fail(err)
	}
	s := attached.SessionID
	landscape := sp.CSSW > sp.CSSH
	orientation := map[bool]map[string]any{false: {"type": "portraitPrimary", "angle": 0}, true: {"type": "landscapePrimary", "angle": 90}}[landscape]
	for _, step := range []struct {
		method string
		params any
	}{
		{"Page.enable", nil},
		{"Emulation.setDeviceMetricsOverride", map[string]any{"width": sp.CSSW, "height": sp.CSSH, "deviceScaleFactor": sp.Scale, "mobile": true,
			"screenWidth": sp.CSSW, "screenHeight": sp.CSSH, "screenOrientation": orientation}},
		{"Emulation.setTouchEmulationEnabled", map[string]any{"enabled": true, "maxTouchPoints": 5}},
		{"Emulation.setFocusEmulationEnabled", map[string]any{"enabled": true}},
	} {
		if err := c.call(ctx, s, step.method, step.params, nil); err != nil {
			return fail(err)
		}
	}
	var nav struct {
		ErrorText string `json:"errorText"`
	}
	if err := c.call(ctx, s, "Page.navigate", map[string]any{"url": sp.URL}, &nav); err != nil {
		return fail(err)
	}
	if nav.ErrorText != "" {
		return fmt.Errorf("打不開 %s：%s", sp.URL, nav.ErrorText)
	}
	next := time.Now().Add(sp.Wait)
	for i := 0; i < sp.N; i++ {
		select {
		case <-time.After(time.Until(next)):
		case <-c.done:
			return fail(errors.New("擷取前 Chrome 就結束了"))
		case <-ctx.Done():
			return fail(ctx.Err())
		}
		// 下一張從這張開始擷取時算（擷取本身要花一點時間，不累積誤差）
		next = time.Now().Add(sp.Interval)
		var shot struct {
			Data []byte `json:"data"` // base64，encoding/json 會自動解碼
		}
		params := map[string]any{"format": "png", "fromSurface": true}
		if sp.Clip != nil {
			params["clip"] = map[string]any{"x": 0, "y": 0, "width": sp.Clip[0], "height": sp.Clip[1], "scale": 1}
		}
		if err := c.call(ctx, s, "Page.captureScreenshot", params, &shot); err != nil {
			return fail(err)
		}
		if err := onShot(i, shot.Data); err != nil {
			return err
		}
	}
	return nil
}

// limitedBuffer 只保留最後 64 KB 的輸出（Chrome 的錯誤訊息）。
type limitedBuffer struct {
	mu sync.Mutex
	b  []byte
}

func (l *limitedBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.b = append(l.b, p...)
	if len(l.b) > 64<<10 {
		l.b = l.b[len(l.b)-64<<10:]
	}
	return len(p), nil
}

func (l *limitedBuffer) tail(n int) string {
	l.mu.Lock()
	defer l.mu.Unlock()
	lines := strings.Split(strings.TrimSpace(string(l.b)), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}
