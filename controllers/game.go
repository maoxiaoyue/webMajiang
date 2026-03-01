package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"webmajiang/models"
	"webmajiang/service"
)

// GameStateKey 遊戲狀態在 Redis 中的 key
func GameStateKey(gameID string) string {
	return fmt.Sprintf("game:%s:state", gameID)
}

// GameStatusKey 遊戲狀況紀錄在 Redis 中的 key
func GameStatusKey(gameID string) string {
	return fmt.Sprintf("mjgame:%s:status", gameID)
}

// GameStatus 遊戲狀況紀錄結構 (存放在 mjgame:<gameid>:status)
type GameStatus struct {
	Type     int    `json:"type"`     // 13 為 13 張，16 為 16 張
	Player1  string `json:"player1"`  // 玩家1識別 (user:<id> 或 bot)
	Player2  string `json:"player2"`  // 玩家2識別
	Player3  string `json:"player3"`  // 玩家3識別
	Player4  string `json:"player4"`  // 玩家4識別
	Dealer   string `json:"dealer"`   // 莊家 (例: "player2")
	Start    int64  `json:"start"`    // 遊戲開始時間戳
	Progress string `json:"progress"` // 目前進度 (例: "2-3" 表示南風西局)
}

// SaveGameStatus 儲存遊戲狀況紀錄到 Redis
func SaveGameStatus(ctx context.Context, gameID string, status *GameStatus) error {
	data, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("failed to marshal game status: %w", err)
	}

	key := GameStatusKey(gameID)
	if err := service.RedisClient.Set(ctx, key, string(data), 0).Err(); err != nil {
		return fmt.Errorf("failed to save game status: %w", err)
	}

	return nil
}

// LoadGameStatus 從 Redis 讀取遊戲狀況紀錄
func LoadGameStatus(ctx context.Context, gameID string) (*GameStatus, error) {
	key := GameStatusKey(gameID)
	data, err := service.RedisClient.Get(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to load game status: %w", err)
	}

	var status GameStatus
	if err := json.Unmarshal([]byte(data), &status); err != nil {
		return nil, fmt.Errorf("failed to unmarshal game status: %w", err)
	}

	return &status, nil
}

// UpdateGameStatusProgress 更新遊戲進度 (局號)
func UpdateGameStatusProgress(ctx context.Context, gameID string, round models.GameRound) error {
	status, err := LoadGameStatus(ctx, gameID)
	if err != nil {
		return err
	}
	status.Progress = round.RoundCode()
	return SaveGameStatus(ctx, gameID, status)
}

// buildPlayerIdentifier 根據玩家資料生成識別字串
// 真人玩家回傳 "user:<id>"，AI 玩家回傳 "bot"
func buildPlayerIdentifier(player models.Player) string {
	if player.IsBot {
		return "bot"
	}
	return fmt.Sprintf("user:%d", player.ID)
}

// RollDice 使用 ChaCha20 擲兩顆骰子
func RollDice() (models.DiceResult, error) {
	rng, err := newChaCha20Rand()
	if err != nil {
		return models.DiceResult{}, fmt.Errorf("failed to create RNG for dice: %w", err)
	}

	die1 := rng.Intn(6) + 1 // 1-6
	die2 := rng.Intn(6) + 1 // 1-6

	return models.DiceResult{
		Die1:  die1,
		Die2:  die2,
		Total: die1 + die2,
	}, nil
}

// DetermineDealerByDice 根據擲骰結果決定莊家玩家代號 (1-4)
// 骰子點數總和對 4 取餘：
//
//	1 → 玩家 1 (東)
//	2 → 玩家 2 (南)
//	3 → 玩家 3 (西)
//	0 → 玩家 4 (北)
func DetermineDealerByDice(diceTotal int) int {
	seat := diceTotal % 4
	if seat == 0 {
		seat = 4
	}
	return seat
}

// SaveGameState 儲存遊戲狀態到 Redis
func SaveGameState(ctx context.Context, state *models.GameState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal game state: %w", err)
	}

	key := GameStateKey(state.GameID)
	if err := service.RedisClient.Set(ctx, key, string(data), 0).Err(); err != nil {
		return fmt.Errorf("failed to save game state: %w", err)
	}

	return nil
}

// LoadGameState 從 Redis 讀取遊戲狀態
func LoadGameState(ctx context.Context, gameID string) (*models.GameState, error) {
	key := GameStateKey(gameID)
	data, err := service.RedisClient.Get(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to load game state: %w", err)
	}

	var state models.GameState
	if err := json.Unmarshal([]byte(data), &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal game state: %w", err)
	}

	return &state, nil
}

// StartNewGame 開始新的一將（第一局）
// 初始化遊戲狀態，並進入 StageWaitingPlayers 階段
// gameType: 13 為 13 張玩法 (不含花牌)，16 為 16 張玩法 (含花牌)
func StartNewGame(ctx context.Context, gameID string, gameType models.GameType) (*models.GameState, error) {
	// 驗證遊戲類型
	if gameType != models.GameType13 && gameType != models.GameType16 {
		gameType = models.GameType16 // 預設 16 張
	}

	state := &models.GameState{
		GameID:          gameID,
		GameType:        gameType,
		Stage:           models.StageWaitingPlayers,
		CurrentPlayerID: 0,
		Round:           models.NewFirstRound(), // 東風東 (1-1)
		DealerPlayerID:  0,
		IsStarted:       true,
		IsFinished:      false,
		Players:         make(map[int]models.Player),
	}

	// 暫時設定為「Seat 1,2 為真人玩家，Seat 3,4 為 AI 玩家」
	state.Players[1] = models.Player{ID: 1, Name: "真人玩家1", IsBot: false, Hand: []models.Tile{}}
	state.Players[2] = models.Player{ID: 2, Name: "真人玩家2", IsBot: false, Hand: []models.Tile{}}
	state.Players[3] = models.Player{ID: 3, Name: "AI 電腦1", IsBot: true, Hand: []models.Tile{}}
	state.Players[4] = models.Player{ID: 4, Name: "AI 電腦2", IsBot: true, Hand: []models.Tile{}}

	if err := SaveGameState(ctx, state); err != nil {
		return nil, err
	}

	// 記錄遊戲狀況到 mjgame:<gameid>:status
	status := &GameStatus{
		Type:     int(gameType),
		Player1:  buildPlayerIdentifier(state.Players[1]),
		Player2:  buildPlayerIdentifier(state.Players[2]),
		Player3:  buildPlayerIdentifier(state.Players[3]),
		Player4:  buildPlayerIdentifier(state.Players[4]),
		Dealer:   "", // 尚未決定莊家
		Start:    time.Now().Unix(),
		Progress: state.Round.RoundCode(), // "1-1" 東風東
	}
	if err := SaveGameStatus(ctx, gameID, status); err != nil {
		return nil, fmt.Errorf("failed to save game status: %w", err)
	}

	return state, nil
}

// RollPositions 決定座位 (擲骰子)
func RollPositions(ctx context.Context, gameID string) (*models.GameState, error) {
	state, err := LoadGameState(ctx, gameID)
	if err != nil {
		return nil, err
	}
	if state.Stage != models.StageWaitingPlayers {
		return nil, fmt.Errorf("action not allowed in current stage: %s", state.Stage)
	}

	dice, err := RollDice()
	if err != nil {
		return nil, err
	}
	state.Dice = dice
	state.Stage = models.StageDetermineDealer // 進到決定莊家

	if err := SaveGameState(ctx, state); err != nil {
		return nil, err
	}
	return state, nil
}

// RollDealer 決定第一局莊家
func RollDealer(ctx context.Context, gameID string) (*models.GameState, error) {
	state, err := LoadGameState(ctx, gameID)
	if err != nil {
		return nil, err
	}
	if state.Stage != models.StageDetermineDealer {
		return nil, fmt.Errorf("action not allowed in current stage: %s", state.Stage)
	}

	dice, err := RollDice()
	if err != nil {
		return nil, err
	}
	state.Dice = dice
	state.DealerPlayerID = DetermineDealerByDice(dice.Total)
	state.Stage = models.StageDealing // 準備發牌

	if err := SaveGameState(ctx, state); err != nil {
		return nil, err
	}

	// 更新遊戲狀況紀錄中的莊家
	if status, err := LoadGameStatus(ctx, gameID); err == nil {
		status.Dealer = fmt.Sprintf("player%d", state.DealerPlayerID)
		_ = SaveGameStatus(ctx, gameID, status)
	}

	return state, nil
}

// DealTilesAction 執行發牌流程 (含洗牌、發牌、理牌)
// 13張玩法: 每人13張，莊家14張 (136張牌，不含花牌)
// 16張玩法: 每人16張，莊家17張 (144張牌，含花牌)
func DealTilesAction(ctx context.Context, gameID string) (*models.GameState, error) {
	state, err := LoadGameState(ctx, gameID)
	if err != nil {
		return nil, err
	}
	if state.Stage != models.StageDealing {
		return nil, fmt.Errorf("action not allowed in current stage: %s", state.Stage)
	}

	if err := InitDeckToRedis(ctx, gameID, state.GameType); err != nil {
		return nil, fmt.Errorf("init deck failed: %w", err)
	}

	if err := DealTilesFromSeat(ctx, gameID, state.DealerPlayerID, state.GameType); err != nil {
		return nil, fmt.Errorf("deal tiles failed: %w", err)
	}

	// 莊家已拿到多一張開門牌，接下來換莊家打牌
	state.Stage = models.StagePlayerDiscard
	state.CurrentPlayerID = state.DealerPlayerID
	state.ActionDeclarations = make(map[int]string)

	if err := SaveGameState(ctx, state); err != nil {
		return nil, err
	}
	return state, nil
}

// DiscardTileAction 玩家出牌
func DiscardTileAction(ctx context.Context, gameID string, playerID int, tile models.Tile) (*models.GameState, error) {
	state, err := LoadGameState(ctx, gameID)
	if err != nil {
		return nil, err
	}

	if state.Stage != models.StagePlayerDiscard {
		return nil, fmt.Errorf("action not allowed in current stage: %s", state.Stage)
	}

	if state.CurrentPlayerID != playerID {
		return nil, fmt.Errorf("not your turn to discard, current player is %d", state.CurrentPlayerID)
	}

	// 1. 從玩家手牌中移除該牌
	if err := RemoveTileFromPlayerHand(ctx, gameID, playerID, tile); err != nil {
		return nil, fmt.Errorf("failed to discard tile: %w", err)
	}

	// 2. 更新狀態機
	state.LastDiscardTile = &tile
	state.LastDiscardPlayerID = playerID
	state.ActionDeclarations = make(map[int]string) // 重置各家宣告
	state.Stage = models.StageWaitAction

	if err := SaveGameState(ctx, state); err != nil {
		return nil, err
	}
	return state, nil
}

// PlayerDeclareAction 玩家宣告 (吃/碰/槓/胡/放棄)
func PlayerDeclareAction(ctx context.Context, gameID string, playerID int, action string) (*models.GameState, error) {
	state, err := LoadGameState(ctx, gameID)
	if err != nil {
		return nil, err
	}

	if state.Stage != models.StageWaitAction {
		return nil, fmt.Errorf("action not allowed in current stage: %s", state.Stage)
	}

	if playerID == state.LastDiscardPlayerID {
		return nil, fmt.Errorf("cannot declare action on your own discard")
	}

	// 紀錄宣告
	if state.ActionDeclarations == nil {
		state.ActionDeclarations = make(map[int]string)
	}
	state.ActionDeclarations[playerID] = action

	// 檢查是否所有其他玩家都宣告了，或者有人直接胡了 (胡最大)
	allDeclared := len(state.ActionDeclarations) == 3
	hasHu := false
	for _, a := range state.ActionDeclarations {
		if a == "hu" {
			hasHu = true
			break
		}
	}

	// 如果有人宣告 "hu" 或者所有人都表態了，進行結算
	if hasHu || allDeclared {
		state, err = ResolveActions(ctx, state)
		if err != nil {
			return nil, err
		}
	}

	if err := SaveGameState(ctx, state); err != nil {
		return nil, err
	}
	return state, nil
}

// ResolveActions 結算吃碰槓胡優先權
func ResolveActions(ctx context.Context, state *models.GameState) (*models.GameState, error) {
	// 優先權：hu > kong/pong > chow > pass
	// 由於需要找得標者，我們定義 Priority
	var highestPriority int = -1
	var winnerID int = -1
	var winningAction string = "pass"

	priorityMap := map[string]int{
		"pass": 0,
		"chow": 1,
		"pong": 2,
		"kong": 3,
		"hu":   4,
	}

	for pID, action := range state.ActionDeclarations {
		p := priorityMap[action]
		if p > highestPriority {
			highestPriority = p
			winnerID = pID
			winningAction = action
		}
	}

	if winningAction == "hu" {
		// 有人胡牌，進入結算階段
		state.Stage = models.StageRoundOver
		state.CurrentPlayerID = winnerID // 讓客戶端知道誰胡了
		return state, nil
	}

	if winningAction == "kong" || winningAction == "pong" || winningAction == "chow" {
		// TODO: 將 LastDiscardTile 放入得標者的副露區 (目前沒寫副露邏輯)
		// ... 執行吃碰邏輯 ...

		state.Stage = models.StagePlayerDiscard // 碰/吃完要打一張牌
		state.CurrentPlayerID = winnerID
		state.LastDiscardTile = nil
		return state, nil
	}

	// Todos Pass: 無人要牌，輪到下家摸牌
	nextPlayerID := (state.LastDiscardPlayerID % 4) + 1
	state.Stage = models.StagePlayerDraw
	state.CurrentPlayerID = nextPlayerID
	state.LastDiscardTile = nil // 已成廢牌

	return state, nil
}

// NextRound 進入下一局
func NextRound(ctx context.Context, gameID string) (*models.GameState, bool, error) {
	state, err := LoadGameState(ctx, gameID)
	if err != nil {
		return nil, false, err
	}

	nextRound, isComplete := state.Round.NextRound()
	if isComplete {
		state.IsFinished = true
		state.Stage = models.StageGameOver
		if err := SaveGameState(ctx, state); err != nil {
			return nil, true, err
		}
		return state, true, nil // 一將結束
	}

	// 更新局號，莊家順轉（下家做莊）
	state.Round = nextRound
	state.DealerPlayerID = (state.DealerPlayerID % 4) + 1
	state.Stage = models.StageDealing // 下一局回到洗牌/發牌階段
	state.CurrentPlayerID = 0

	if err := SaveGameState(ctx, state); err != nil {
		return nil, false, err
	}

	// 更新遊戲狀況紀錄中的進度和莊家
	if status, err := LoadGameStatus(ctx, gameID); err == nil {
		status.Progress = nextRound.RoundCode()
		status.Dealer = fmt.Sprintf("player%d", state.DealerPlayerID)
		_ = SaveGameStatus(ctx, gameID, status)
	}

	return state, false, nil
}

// DealTilesFromSeat 從指定莊家玩家 ID 開始發牌
// 發牌順序以莊家為起始，逆時針輪流 (莊→下家→對家→上家)
// 座位順序: dealerPlayerID, dealerPlayerID下家 等 (1-4循環)
//
// 13張玩法: 每人13張，莊家14張
//
//	第一階段：輪流摸 4 張 × 3 輪 = 每人 12 張
//	第二階段：每人再摸 1 張 = 每人 13 張
//	第三階段：莊家再摸 1 張開門 = 莊家 14 張
//
// 16張玩法: 每人16張，莊家17張
//
//	第一階段：輪流摸 4 張 × 4 輪 = 每人 16 張
//	第二階段：莊家再摸 1 張開門 = 莊家 17 張
func DealTilesFromSeat(ctx context.Context, gameID string, dealerPlayerID int, gameType models.GameType) error {
	rdb := service.RedisClient
	deckKey := DeckRedisKey(gameID)

	// 計算發牌順序
	order := [4]int{
		dealerPlayerID,
		(dealerPlayerID % 4) + 1,
		((dealerPlayerID + 1) % 4) + 1,
		((dealerPlayerID + 2) % 4) + 1,
	}

	// 先清除舊的玩家手牌
	for _, p := range order {
		playerKey := PlayerHandKey(gameID, p)
		if err := rdb.Del(ctx, playerKey).Err(); err != nil {
			return fmt.Errorf("failed to clear player%d hand: %w", p, err)
		}
	}

	if gameType == models.GameType16 {
		// ===== 16張玩法 =====
		// ----- 第一階段：輪流摸 4 張 × 4 輪 = 每人 16 張 -----
		for round := 0; round < 4; round++ {
			for _, p := range order {
				playerKey := PlayerHandKey(gameID, p)
				for t := 0; t < 4; t++ {
					data, err := rdb.RPop(ctx, deckKey).Result()
					if err != nil {
						return fmt.Errorf("round %d, player%d, tile %d: RPOP failed: %w", round+1, p, t+1, err)
					}
					if err := rdb.LPush(ctx, playerKey, data).Err(); err != nil {
						return fmt.Errorf("round %d, player%d, tile %d: LPUSH failed: %w", round+1, p, t+1, err)
					}
				}
			}
		}
	} else {
		// ===== 13張玩法 =====
		// ----- 第一階段：輪流摸 4 張 × 3 輪 = 每人 12 張 -----
		for round := 0; round < 3; round++ {
			for _, p := range order {
				playerKey := PlayerHandKey(gameID, p)
				for t := 0; t < 4; t++ {
					data, err := rdb.RPop(ctx, deckKey).Result()
					if err != nil {
						return fmt.Errorf("round %d, player%d, tile %d: RPOP failed: %w", round+1, p, t+1, err)
					}
					if err := rdb.LPush(ctx, playerKey, data).Err(); err != nil {
						return fmt.Errorf("round %d, player%d, tile %d: LPUSH failed: %w", round+1, p, t+1, err)
					}
				}
			}
		}

		// ----- 第二階段：每人再摸 1 張 = 每人 13 張 -----
		for _, p := range order {
			playerKey := PlayerHandKey(gameID, p)
			data, err := rdb.RPop(ctx, deckKey).Result()
			if err != nil {
				return fmt.Errorf("final draw, player%d: RPOP failed: %w", p, err)
			}
			if err := rdb.LPush(ctx, playerKey, data).Err(); err != nil {
				return fmt.Errorf("final draw, player%d: LPUSH failed: %w", p, err)
			}
		}
	}

	// ----- 莊家再摸 1 張開門 -----
	dealerKey := PlayerHandKey(gameID, dealerPlayerID)
	data, err := rdb.RPop(ctx, deckKey).Result()
	if err != nil {
		return fmt.Errorf("dealer open tile: RPOP failed: %w", err)
	}
	if err := rdb.LPush(ctx, dealerKey, data).Err(); err != nil {
		return fmt.Errorf("dealer open tile: LPUSH failed: %w", err)
	}

	// ----- 理牌 -----
	for p := 1; p <= 4; p++ {
		if err := SortPlayerHand(ctx, gameID, p); err != nil {
			return fmt.Errorf("sort player%d hand failed: %w", p, err)
		}
	}

	return nil
}
