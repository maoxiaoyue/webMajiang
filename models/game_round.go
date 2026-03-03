package models

// WindPosition 風位 (圈風/門風)
type WindPosition int

const (
	East  WindPosition = 1 // 東
	South WindPosition = 2 // 南
	West  WindPosition = 3 // 西
	North WindPosition = 4 // 北
)

// WindPositionName 風位名稱
var WindPositionName = map[WindPosition]string{
	East:  "東",
	South: "南",
	West:  "西",
	North: "北",
}

// String 回傳風位名稱
func (w WindPosition) String() string {
	if name, ok := WindPositionName[w]; ok {
		return name
	}
	return "未知"
}

// GameRound 局號狀態
// 圈風 (PrevailingWind): 東→南→西→北 (1-4)
// 門風 / 局 (HandWind):  東→南→西→北 (1-4)
// 合計 4×4 = 16 局 = 一將 (一雀)
//
// 局號對照:
//
//	東風東(1-1) 東風南(1-2) 東風西(1-3) 東風北(1-4)
//	南風東(2-1) 南風南(2-2) 南風西(2-3) 南風北(2-4)
//	西風東(3-1) 西風南(3-2) 西風西(3-3) 西風北(3-4)
//	北風東(4-1) 北風南(4-2) 北風西(4-3) 北風北(4-4)
type GameRound struct {
	PrevailingWind WindPosition `json:"prevailing_wind"` // 圈風 (1-4)
	HandWind       WindPosition `json:"hand_wind"`       // 門風/局 (1-4)
}

// RoundLabel 回傳可讀局號，例如 "東風東"
func (r GameRound) RoundLabel() string {
	return WindPositionName[r.PrevailingWind] + "風" + WindPositionName[r.HandWind]
}

// RoundCode 回傳局號代碼，例如 "1-1"
func (r GameRound) RoundCode() string {
	return string(rune('0'+r.PrevailingWind)) + "-" + string(rune('0'+r.HandWind))
}

// RoundNumber 回傳線性局號 (1-16)
func (r GameRound) RoundNumber() int {
	return (int(r.PrevailingWind)-1)*4 + int(r.HandWind)
}

// IsLastRound 是否為最後一局 (北風北 4-4)
func (r GameRound) IsLastRound() bool {
	return r.PrevailingWind == North && r.HandWind == North
}

// NextRound 計算下一局的局號
// 回傳下一局的 GameRound，如果已經是最後一局則回傳 nil 和 true
func (r GameRound) NextRound() (GameRound, bool) {
	if r.IsLastRound() {
		return GameRound{}, true // 一將結束
	}

	next := GameRound{
		PrevailingWind: r.PrevailingWind,
		HandWind:       r.HandWind + 1,
	}

	// 門風到北風後，圈風進位
	if next.HandWind > North {
		next.HandWind = East
		next.PrevailingWind++
	}

	return next, false
}

// NewFirstRound 建立第一局 (東風東 1-1)
func NewFirstRound() GameRound {
	return GameRound{
		PrevailingWind: East,
		HandWind:       East,
	}
}

// RoundFromNumber 由線性局號 (1-16) 建立 GameRound
func RoundFromNumber(n int) GameRound {
	if n < 1 {
		n = 1
	}
	if n > 16 {
		n = 16
	}
	n-- // 轉為 0-indexed
	return GameRound{
		PrevailingWind: WindPosition(n/4 + 1),
		HandWind:       WindPosition(n%4 + 1),
	}
}

// DiceResult 擲骰子結果
type DiceResult struct {
	Die1  int `json:"die1"`  // 第一顆骰子 (1-6)
	Die2  int `json:"die2"`  // 第二顆骰子 (1-6)
	Total int `json:"total"` // 點數總和
}

// GameStage 遊戲階段
type GameStage string

const (
	StageWaitingPlayers     GameStage = "WAITING_PLAYERS"     // 等待玩家加入/準備
	StageDeterminePositions GameStage = "DETERMINE_POSITIONS" // 開始 -> 擲骰子決定位置
	StageDetermineDealer    GameStage = "DETERMINE_DEALER"    // 坐下後擲骰子決定東風位置
	StageDealing            GameStage = "DEALING"             // 第一局開始取牌/發牌
	StagePlayerDraw         GameStage = "PLAYER_DRAW"         // 玩家摸牌階段 (包含開門)
	StagePlayerDiscard      GameStage = "PLAYER_DISCARD"      // 玩家出牌階段
	StageWaitAction         GameStage = "WAIT_ACTION"         // 等待其他家宣告 (碰/槓/胡)
	StageRoundOver          GameStage = "ROUND_OVER"          // 單局結算
	StageGameOver           GameStage = "GAME_OVER"           // 遊戲終局
)

// GameType 遊戲類型 (13張 或 16張)
type GameType int

const (
	GameType13 GameType = 13 // 13張玩法 (不含花牌，136張)
	GameType16 GameType = 16 // 16張玩法 (含花牌，144張)
)

// GameState 完整遊戲狀態（存放在 Redis 中）
type GameState struct {
	GameID              string         `json:"game_id"`
	GameType            GameType       `json:"game_type"`              // 遊戲類型 (13 或 16)
	Stage               GameStage      `json:"stage"`                  // 目前遊戲階段
	CurrentPlayerID     int            `json:"current_player_id"`      // 目前輪到的玩家代號 (1-4)
	Round               GameRound      `json:"round"`                  // 目前局號
	DealerPlayerID      int            `json:"dealer_player_id"`       // 莊家玩家代號 (1-4)
	Dice                DiceResult     `json:"dice"`                   // 擲骰子結果
	IsStarted           bool           `json:"is_started"`             // 是否已開始
	IsFinished          bool           `json:"is_finished"`            // 一將是否結束
	Players             map[int]Player `json:"players"`                // 玩家列表 (SeatID 1-4 對應 -> Player)
	LastDiscardTile     *Tile          `json:"last_discard_tile"`      // 最新打出的一張牌 (可為 null)
	LastDiscardPlayerID int            `json:"last_discard_player_id"` // 是誰打出最新的這張牌
	ActionDeclarations  map[int]string `json:"action_declarations"`    // 紀錄各家在 WAIT_ACTION 階段宣吿的動作 ("pass", "pong", "kong", "hu")
}
