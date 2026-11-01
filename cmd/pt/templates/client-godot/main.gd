extends Control
## 「一起按」Godot 版：用玩聚 SDK 握手 → 把啟動碼交給自己的遊戲伺服器 → 只送操作、畫面照伺服器的狀態顯示。
## 和網頁版範本用同一個遊戲伺服器（server/）。直接在瀏覽器開（不在玩聚裡）或在編輯器執行時，SDK 自動進入模擬模式。

const Config = preload("res://config.gd")

var _token := ""
var _base := ""
var _status: Label
var _timer: Label
var _players: Label
var _start: Button
var _tap: Button


func _ready() -> void:
	_build_ui()
	PlayTogether.initialized.connect(_on_initialized)
	PlayTogether.init_failed.connect(func(msg): _status.text = "無法開始：" + msg)
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
	# 啟動碼只用一次、不要存起來或寫進日誌
	_post("/join", {"launchCode": PlayTogether.launch_code}, func(res: Dictionary):
		_token = str(res.get("token", ""))
		_status.text = "你好，%s" % res.get("name", "")
		PlayTogether.mark_ready()  # 平台會收起載入畫面
		_act("state")
		var poll := Timer.new()
		poll.wait_time = 0.5
		poll.autostart = true
		poll.timeout.connect(func(): _act("state"))
		add_child(poll))


func _act(action: String) -> void:
	if _token == "":
		return
	_post("/action", {"token": _token, "action": action}, _render)


func _render(state: Dictionary) -> void:
	var playing := bool(state.get("playing", false))
	_start.visible = not playing
	_tap.visible = playing
	var winner := str(state.get("winner", ""))
	_timer.text = "剩 %d 秒" % ceili(float(state.get("secondsLeft", 0))) if playing else ("贏家：" + winner if winner != "" else "")
	var lines: PackedStringArray = []
	var i := 1
	for p in state.get("players", []):
		lines.append("%d. %s：%d 下" % [i, p.get("name", ""), int(p.get("taps", 0))])
		i += 1
	_players.text = "\n".join(lines)
	_report({"joined": true, "players": lines.size(), "playing": playing})


## 每個請求用自己的 HTTPRequest（同一個 HTTPRequest 不能同時送兩個請求）
func _post(path: String, body: Dictionary, on_ok: Callable) -> void:
	var req := HTTPRequest.new()
	add_child(req)
	req.request_completed.connect(func(_result: int, code: int, _headers: PackedStringArray, raw: PackedByteArray):
		req.queue_free()
		if code != 200:
			_status.text = "連線中斷，請重新整理（%d）" % code
			return
		var data: Variant = JSON.parse_string(raw.get_string_from_utf8())
		if typeof(data) == TYPE_DICTIONARY:
			on_ok.call(data))
	req.request(_base + path, ["Content-Type: application/json"], HTTPClient.METHOD_POST, JSON.stringify(body))


## 把狀態寫到頁面上（只給自動測試讀取，不影響遊戲）
func _report(state: Dictionary) -> void:
	if OS.has_feature("web"):
		JavaScriptBridge.get_interface("window").__ptGame = JSON.stringify(state)
