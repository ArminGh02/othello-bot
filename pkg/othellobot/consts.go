package othellobot

const HELP_MSG = "Othello is a strategy board game for two players, " +
	"Players take turns placing disks on the board with their assigned " +
	"color facing up. During a play, any disks of the opponent's color " +
	"that are in a straight line and bounded by the disk just placed and " +
	"another disk of the current player's color are turned over to the current " +
	"player's color. played on an 8×8 uncheckered board. The objective of the game " +
	"is to have the majority of disks turned to display one's color " +
	"when the last playable empty square is filled."

const BOT_PIC = "https://cf.ltkcdn.net/boardgames/images/orig/224020-2123x1412-Othello.jpg"

const (
	NEW_GAME_BUTTON_TEXT   = "🎮 New Game"
	SCOREBOARD_BUTTON_TEXT = "🏆 Scoreboard"
	PROFILE_BUTTON_TEXT    = "👤 Profile"
	HELP_BUTTON_TEXT       = "❓ Help"
)

var RESEND_QUERY = "#Resend"
