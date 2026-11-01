extends Node
## 玩聚 SDK（Godot 版），以 autoload「PlayTogether」提供。規格見 docs/sdk-godot.md。
##
## - 網頁匯出：透過 JavaScriptBridge 呼叫 playtogether-sdk-godot.iife.js 提供的 PlayTogetherGodot。
## - 在編輯器或非網頁平台執行：自動進入模擬模式（假玩家、mock- 開頭的啟動碼），
##   搭配 Go SDK 的模擬模式（PT_MOCK=1）即可在本機測試。
##
## 注意：本 autoload 不宣告 class_name，避免和 autoload 名稱衝突。

## 握手完成。info 內含 mode、player（name、locale）、launchCode、roomId
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

## "platform"：在玩聚平台中；"mock"：模擬模式
var mode: String = ""
var player_name: String = ""
var locale: String = ""
## 一次性啟動碼：交給自己的遊戲伺服器兌換，不要存起來或寫進日誌
var launch_code: String = ""
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
