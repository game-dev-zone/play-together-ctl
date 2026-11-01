@tool
extends EditorPlugin
## 啟用外掛時自動加入 autoload「PlayTogether」。

const AUTOLOAD_NAME := "PlayTogether"


func _enable_plugin() -> void:
	add_autoload_singleton(AUTOLOAD_NAME, "res://addons/playtogether/playtogether.gd")


func _disable_plugin() -> void:
	remove_autoload_singleton(AUTOLOAD_NAME)
