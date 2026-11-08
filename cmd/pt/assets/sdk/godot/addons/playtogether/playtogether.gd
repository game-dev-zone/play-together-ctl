extends Node
## 玩聚 SDK（Godot 版），以 autoload「PlayTogether」提供。規格見 docs/sdk-godot.md。
##
## - 網頁匯出：透過 JavaScriptBridge 呼叫 playtogether-sdk-godot.iife.js 提供的 PlayTogetherGodot。
## - 在編輯器或非網頁平台執行：自動進入模擬模式（假玩家、mock- 開頭的啟動碼），
##   搭配 Go SDK 的模擬模式（PT_MOCK=1）即可在本機測試。
##
## 注意：本 autoload 不宣告 class_name，避免和 autoload 名稱衝突。

## 握手完成。info 內含 mode、player（name、locale）、launchCode、serverUrl（代管時才有值）、roomId
signal initialized(info: Dictionary)
## 握手失敗（例如逾時、未載入 JS 檔）
signal init_failed(message: String)
signal paused
signal resumed
## 平台即將移除遊戲
signal closing
signal mute_changed(muted: bool)
## members：[{ "id": 玩家假名 ID, "name": 暱稱 }, ...]
signal members_changed(members: Array)
## 平台浮動按鈕佔用的區域（像素）：top、right、bottom、left
signal insets_changed(insets: Dictionary)
## state：connected、canSpeak、mode
signal voice_state_changed(state: Dictionary)
## 某位玩家正在說話，level 為 0–1 的音量，可用來做角色說話動畫
signal speaking(player_id: String, level: float)
## 平台點數的兌換結果（purchase 的回覆）：status 是 completed、cancelled 或 failed。
## 只是提示：遊戲伺服器要向平台查兌換狀態（GetPurchase），是 completed 才給玩家外觀
signal purchase_result(purchase_id: String, status: String)
## 音高玩法（pitch_start 之後）：平台開始分析
signal pitch_started
## 音高分析停止：reason 是 game、user、denied、not_authorized、mic_denied、unsupported、background、timeout、busy、closed
signal pitch_stopped(reason: String)
## 音高（每秒 15–20 次，只在玩家正在說話時）：voiced、hz、note（MIDI，69 = A4）、cents、confidence、seq、t。
## 沒有聲音時 voiced 是 false，hz、note、cents 是 null。只能當娛樂分數，不能用於任何玩家權益
signal pitch_changed(payload: Dictionary)

## "platform"：在玩聚平台中；"mock"：模擬模式
var mode: String = ""
var player_name: String = ""
var locale: String = ""
## 一次性啟動碼：交給自己的遊戲伺服器兌換，不要存起來或寫進日誌
var launch_code: String = ""
## 遊戲伺服器由玩聚代管時，這個房間的伺服器網址（例如 https://d-….ikuwra.com）；自己架伺服器時是空字串
var server_url: String = ""
## 平台房號
var room_id: String = ""
var is_initialized: bool = false

var _bridge: JavaScriptObject = null
## 保留 JavaScript 回呼的參照，否則會被回收而收不到事件
var _callbacks: Array = []


## 開始和平台握手。platform_origins 為允許的平台來源，例如 ["https://（平台網域）"]。
func init_sdk(platform_origins: PackedStringArray, timeout_ms: int = 5000) -> void:
	if is_initialized:
		return
	if not OS.has_feature("web"):
		_init_mock.call_deferred()
		return
	_bridge = JavaScriptBridge.get_interface("PlayTogetherGodot")
	if _bridge == null:
		init_failed.emit("找不到 PlayTogetherGodot：請在匯出設定的 Head Include 載入 playtogether-sdk-godot.iife.js")
		return
	var on_ready: JavaScriptObject = JavaScriptBridge.create_callback(_on_ready)
	var on_error: JavaScriptObject = JavaScriptBridge.create_callback(_on_error)
	var on_event: JavaScriptObject = JavaScriptBridge.create_callback(_on_event)
	_callbacks = [on_ready, on_error, on_event]
	_bridge.init(",".join(platform_origins), timeout_ms, on_ready, on_error, on_event)


## 遊戲載入完成、可以開始了
func mark_ready() -> void:
	if _bridge:
		_bridge.markReady()


## 回報載入進度（0–1）
func loading_progress(value: float) -> void:
	if _bridge:
		_bridge.loadingProgress(clampf(value, 0.0, 1.0))


## 一局開始。players 為 -1 時不提供人數
func game_started(players: int = -1) -> void:
	if _bridge:
		_bridge.gameStarted(players)


## 一局結束。duration_sec 為 -1 時不提供時長
func game_ended(duration_sec: float = -1.0) -> void:
	if _bridge:
		_bridge.gameEnded(duration_sec)


## 開啟平台的分享面板
func share() -> void:
	if _bridge:
		_bridge.share()


## 請求滿版。必須在玩家點擊後呼叫
func fullscreen() -> void:
	if _bridge:
		_bridge.fullscreen()


## value："portrait"、"landscape" 或 "any"
func set_orientation(value: String) -> void:
	if _bridge:
		_bridge.orientation(value)


## 隱藏或顯示平台的「按住說話」按鈕（改用遊戲自己的介面時隱藏）
func set_ptt_button_visible(visible: bool) -> void:
	if _bridge:
		_bridge.pttButton(visible)


## 遊戲自己的按住說話按鈕：按下傳 true、放開傳 false
func push_to_talk(start: bool) -> void:
	if _bridge:
		_bridge.pushToTalk(start)


## 請求發言（是否允許由遊戲伺服器的發言規則決定）
func request_speak() -> void:
	if _bridge:
		_bridge.requestSpeak()


## 平台點數的兌換：purchase_id 是遊戲伺服器建立的兌換意圖（pur_…）。平台外殼顯示確認畫面，結果用 purchase_result 送回。
## 模擬模式（不是網頁版）一律 failed
func purchase(purchase_id: String) -> void:
	if _bridge:
		_bridge.purchase(purchase_id)
	else:
		purchase_result.emit.call_deferred(purchase_id, "failed")


## 音高玩法：請平台開始分析這位玩家的音高。遊戲伺服器要先送 pitch.policy 授權，玩家要在平台畫面同意。
## 模擬模式（不是網頁版）產生假資料：滑音 220→440 Hz、斷音、穩定 440 Hz 加抖音、低信心
func pitch_start() -> void:
	if _bridge:
		_bridge.pitchStart()
	elif not _mock_pitch:
		_mock_pitch = true
		_mock_pitch_t = 0.0
		_mock_pitch_next = 0.0
		_mock_pitch_seq = 0
		pitch_started.emit.call_deferred()


func pitch_stop() -> void:
	if _bridge:
		_bridge.pitchStop()
	elif _mock_pitch:
		_mock_pitch = false
		pitch_stopped.emit.call_deferred("game")


var _mock_pitch := false
var _mock_pitch_t := 0.0
var _mock_pitch_next := 0.0
var _mock_pitch_seq := 0


func _process(delta: float) -> void:
	if not _mock_pitch:
		return
	_mock_pitch_t += delta
	if _mock_pitch_t < _mock_pitch_next:
		return
	_mock_pitch_next = _mock_pitch_t + 0.05
	var phase := fmod(_mock_pitch_t, 6.0)
	var frame := {"playerId": "mock-player", "sessionId": "mock", "seq": _mock_pitch_seq, "t": _mock_pitch_t * 1000.0,
		"voiced": false, "hz": null, "note": null, "cents": null, "confidence": 0.1}
	_mock_pitch_seq += 1
	if (phase < 2.0) or (phase >= 2.6 and phase < 5.0):
		var hz: float = 220.0 * pow(2.0, phase / 2.0) if phase < 2.0 else 440.0 * pow(2.0, sin(phase * TAU * 5.0) * 15.0 / 1200.0)
		var exact: float = 69.0 + 12.0 * log(hz / 440.0) / log(2.0)
		var note: int = roundi(exact)
		frame.merge({"voiced": true, "hz": hz, "note": note, "cents": clampf((exact - note) * 100.0, -50.0, 50.0), "confidence": 0.92}, true)
	pitch_changed.emit(frame)


func _init_mock() -> void:
	var bytes: PackedByteArray = Crypto.new().generate_random_bytes(16)
	_apply({
		"mode": "mock",
		"player": {"name": "測試玩家", "locale": "zh-TW"},
		"launchCode": "mock-" + bytes.hex_encode(),
		"roomId": "mock-room",
	})


func _apply(info: Dictionary) -> void:
	var player: Dictionary = info.get("player", {})
	mode = str(info.get("mode", ""))
	player_name = str(player.get("name", ""))
	locale = str(player.get("locale", ""))
	launch_code = str(info.get("launchCode", ""))
	server_url = str(info.get("serverUrl", ""))
	room_id = str(info.get("roomId", ""))
	is_initialized = true
	initialized.emit(info)


func _on_ready(args: Array) -> void:
	var info: Variant = JSON.parse_string(str(args[0]))
	if typeof(info) != TYPE_DICTIONARY:
		init_failed.emit("平台回傳的資料格式錯誤")
		return
	_apply(info)


func _on_error(args: Array) -> void:
	init_failed.emit(str(args[0]) if args.size() > 0 else "未知錯誤")


func _on_event(args: Array) -> void:
	if args.size() < 2:
		return
	var event_name: String = str(args[0])
	var payload: Variant = JSON.parse_string(str(args[1]))
	if typeof(payload) != TYPE_DICTIONARY:
		return
	match event_name:
		"pause":
			paused.emit()
		"resume":
			resumed.emit()
		"close":
			closing.emit()
		"mute":
			mute_changed.emit(bool(payload.get("muted", false)))
		"members":
			members_changed.emit(payload.get("members", []))
		"insets":
			insets_changed.emit(payload)
		"voice_state":
			voice_state_changed.emit(payload)
		"speaking":
			speaking.emit(str(payload.get("playerId", "")), float(payload.get("level", 0.0)))
		"purchase_result":
			purchase_result.emit(str(payload.get("purchaseId", "")), str(payload.get("status", "failed")))
		"pitch_state":
			if str(payload.get("state", "")) == "started":
				pitch_started.emit()
			else:
				pitch_stopped.emit(str(payload.get("reason", "")))
		"pitch":
			pitch_changed.emit(payload)
