extends Control
## 「一起按」Godot 版：用玩聚 SDK 握手 → 把啟動碼交給自己的遊戲伺服器 → 只送操作、畫面照伺服器的狀態顯示。
## 和網頁版範本用同一個遊戲伺服器（server/）。直接在瀏覽器開（不在玩聚裡）或在編輯器執行時，SDK 自動進入模擬模式。
##
## 斷線重連：網路中斷時顯示「重新連線中」並持續重試，恢復後自動接上；重新整理或換裝置時，
## 平台會給新的啟動碼，伺服器把你接回原座位。token 失效（例如在別的分頁開了遊戲）時請玩家重新整理。

const Config = preload("res://config.gd")

var _token := ""
var _base := ""
var _greeting := ""
var _failures := 0
var _join_tries := 0
var _poll: Timer
var _status: Label
var _timer: Label
var _players: Label
var _start: Button
var _tap: Button


func _ready() -> void:
	_build_ui()
	PlayTogether.initialized.connect(_on_initialized)
	PlayTogether.init_failed.connect(func(msg): _status.text = "無法開始：" + msg)
	PlayTogether.resumed.connect(func(): _act("state"))  # 從背景回來時立刻更新
	PlayTogether.init_sdk(Config.PLATFORM_ORIGINS)


func _build_ui() -> void:
	var box := VBoxContainer.new()
	box.set_anchors_and_offsets_preset(Control.PRESET_FULL_RECT)
	# 上方留空間給平台的浮動按鈕
	box.offset_top = 72
	box.offset_left = 16
	box.offset_right = -16
	box.add_theme_constant_override("separation", 12)
	add_child(box)
	var title := Label.new()
	title.text = ProjectSettings.get_setting("application/config/name")
	title.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	title.add_theme_font_size_override("font_size", 32)
	box.add_child(title)
	_status = _label(box, "連線中…")
	_start = _button(box, "開始一局（10 秒）", func(): _act("start"))
	_tap = _button(box, "按我！", func(): _act("tap"))
	_tap.visible = false
	_timer = _label(box, "")
	_players = _label(box, "")
	_players.horizontal_alignment = HORIZONTAL_ALIGNMENT_LEFT


func _label(parent: Node, text: String) -> Label:
	var l := Label.new()
	l.text = text
	l.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	l.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	l.add_theme_font_size_override("font_size", 20)
	parent.add_child(l)
	return l


func _button(parent: Node, text: String, on_press: Callable) -> Button:
	var b := Button.new()
	b.text = text
	b.custom_minimum_size = Vector2(0, 64)
	b.add_theme_font_size_override("font_size", 24)
	b.pressed.connect(on_press)
	parent.add_child(b)
	return b


func _on_initialized(_info: Dictionary) -> void:
	# HTTPRequest 不接受相對網址：沒有設定伺服器網址時，網頁版用遊戲畫面自己的來源（pt dev）
	_base = Config.SERVER_URL
	if _base == "":
		_base = str(JavaScriptBridge.get_interface("location").origin) if OS.has_feature("web") else "http://127.0.0.1:8090"
	_join()


## 啟動碼只用一次、不要存起來或寫進日誌；網路錯誤時重試幾次（伺服器沒收到就還能用）
func _join() -> void:
	_join_tries += 1
	_post("/join", {"launchCode": PlayTogether.launch_code}, func(code: int, res: Dictionary):
		if code == 200:
			_token = str(res.get("token", ""))
			_greeting = "你好，%s" % res.get("name", "")
			_status.text = _greeting
			_render(res.get("state", {}))  # 重新連線後馬上看到目前的狀態
			PlayTogether.mark_ready()  # 平台會收起載入畫面
			_poll = Timer.new()
			_poll.wait_time = 0.5
			_poll.autostart = true
			_poll.timeout.connect(func(): _act("state"))
			add_child(_poll)
		elif (code == 0 or code >= 500) and _join_tries < 3:
			get_tree().create_timer(_join_tries).timeout.connect(_join)
		elif code == 409:
			_status.text = "這一局已經重新開始，請重新整理頁面"
		elif code == 503:
			_status.text = "暫時無法加入，請稍後重新整理頁面"
		elif code == 403:
			_status.text = "無法加入這個房間，請重新整理頁面"
		else:
			_status.text = "連不上遊戲伺服器，請稍後重新整理頁面")


func _act(action: String) -> void:
	if _token == "":
		return
	_post("/action", {"token": _token, "action": action}, func(code: int, res: Dictionary):
		if code == 200:
			_failures = 0
			_status.text = _greeting
			_render(res)
		elif code == 401:
			_expired("連線已失效（可能在別的分頁或裝置開了遊戲）：請重新整理頁面")
		else:
			_failures += 1
			if _failures >= 2:
				_status.text = "連線中斷，重新連線中…"  # 網路恢復後下一次輪詢就會接上
		_report({"joined": _token != "", "players": _players.text.count("\n") + 1 if _players.text != "" else 0, "playing": _tap.visible, "status": _status.text}))


## 連線失效：停止輪詢，請玩家重新整理（平台會給新的啟動碼，回到原座位）
func _expired(message: String) -> void:
	_token = ""
	if _poll:
		_poll.stop()
	_start.visible = false
	_tap.visible = false
	_status.text = message


func _render(state: Dictionary) -> void:
	var playing := bool(state.get("playing", false))
	_start.visible = not playing
	_tap.visible = playing
	var winner := str(state.get("winner", ""))
	_timer.text = "剩 %d 秒" % ceili(float(state.get("secondsLeft", 0))) if playing else ("贏家：" + winner if winner != "" else "")
	var lines: PackedStringArray = []
	var i := 1
	for p in state.get("players", []):
		lines.append("%d. %s：%d 下%s" % [i, p.get("name", ""), int(p.get("taps", 0)), "" if p.get("online", true) else "（離線）"])
		i += 1
	_players.text = "\n".join(lines)


## 每個請求用自己的 HTTPRequest（同一個 HTTPRequest 不能同時送兩個請求）。
## on_done 收到 (HTTP 狀態碼, 回應)；網路錯誤時狀態碼是 0
func _post(path: String, body: Dictionary, on_done: Callable) -> void:
	var req := HTTPRequest.new()
	req.timeout = 8.0
	add_child(req)
	req.request_completed.connect(func(result: int, code: int, _headers: PackedStringArray, raw: PackedByteArray):
		req.queue_free()
		if result != HTTPRequest.RESULT_SUCCESS:
			code = 0
		var data: Variant = JSON.parse_string(raw.get_string_from_utf8())
		on_done.call(code, data if typeof(data) == TYPE_DICTIONARY else {}))
	if req.request(_base + path, ["Content-Type: application/json"], HTTPClient.METHOD_POST, JSON.stringify(body)) != OK:
		req.queue_free()
		on_done.call(0, {})


## 把狀態寫到頁面上（只給自動測試讀取，不影響遊戲）
func _report(state: Dictionary) -> void:
	if OS.has_feature("web"):
		JavaScriptBridge.get_interface("window").__ptGame = JSON.stringify(state)
