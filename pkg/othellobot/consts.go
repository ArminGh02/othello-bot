package othellobot

const helpMsg = "Othello is a strategy board game for two players, " +
	"Players take turns placing disks on the board with their assigned " +
	"color facing up. During a play, any disks of the opponent's color " +
	"that are in a straight line and bounded by the disk just placed and " +
	"another disk of the current player's color are turned over to the current " +
	"player's color. played on an 8×8 uncheckered board. The objective of the game " +
	"is to have the majority of disks turned to display one's color " +
	"when the last playable empty square is filled."

const botPic = "https://cf.ltkcdn.net/boardgames/images/orig/224020-2123x1412-Othello.jpg"

const (
	newGameButtonText    = "🎮 New Game"
	scoreboardButtonText = "🏆 Scoreboard"
	profileButtonText    = "👤 Profile"
	helpButtonText       = "❓ Help"
)

var resendQuery = "#Resend"
