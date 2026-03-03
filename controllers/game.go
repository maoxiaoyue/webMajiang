package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"webmajiang/models"
	"webmajiang/service"
	"webmajiang/utils"
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

// RollDice3 使用 ChaCha20 擲三顆骰子 (16張用)
func RollDice3() (models.DiceResult, error) {
	rng, err := newChaCha20Rand()
	if err != nil {
		return models.DiceResult{}, fmt.Errorf("failed to create RNG for dice: %w", err)
	}

	die1 := rng.Intn(6) + 1 // 1-6
	die2 := rng.Intn(6) + 1 // 1-6
	die3 := rng.Intn(6) + 1 // 1-6

	return models.DiceResult{
		Die1:  die1,
		Die2:  die2,
		Die3:  die3,
		Total: die1 + die2 + die3,
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

	var dice models.DiceResult
	if state.GameType == models.GameType16 {
		dice, err = RollDice3()
	} else {
		dice, err = RollDice()
	}
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

	var dice models.DiceResult
	if state.GameType == models.GameType16 {
		dice, err = RollDice3()
	} else {
		dice, err = RollDice()
	}
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

	if state.GameType == models.GameType16 {
		state.Stage = models.StageReplaceFlower
		if err := SaveGameState(ctx, state); err != nil {
			return nil, err
		}
		if err := ExecuteAutoFlowerReplacement(ctx, gameID); err != nil {
			return nil, fmt.Errorf("auto flower replacement failed: %w", err)
		}
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

// ExecuteAutoFlowerReplacement 執行開局自動補花
func ExecuteAutoFlowerReplacement(ctx context.Context, gameID string) error {
	state, err := LoadGameState(ctx, gameID)
	if err != nil {
		return err
	}
	rdb := service.RedisClient
	deckKey := DeckRedisKey(gameID)

	order := [4]int{
		state.DealerPlayerID,
		(state.DealerPlayerID % 4) + 1,
		((state.DealerPlayerID + 1) % 4) + 1,
		((state.DealerPlayerID + 2) % 4) + 1,
	}

	for _, p := range order {
		for {
			playerKey := PlayerHandKey(gameID, p)
			tileJSONs, err := rdb.LRange(ctx, playerKey, 0, -1).Result()
			if err != nil {
				return err
			}

			var nonFlowers []string
			var flowers []string
			flowerCount := 0

			for _, tj := range tileJSONs {
				var tile models.Tile
				if err := json.Unmarshal([]byte(tj), &tile); err != nil {
					return err
				}
				if tile.Type == models.Flower {
					flowerCount++
					flowers = append(flowers, tj)
					utils.Info("[Flower] player%d draws a flower: %v", p, tile)
				} else {
					nonFlowers = append(nonFlowers, tj)
				}
			}

			if flowerCount == 0 {
				break // 本家無花，換下一家
			}

			// 將花牌存入專屬的 Redis 列表
			flowersKey := PlayerFlowersKey(gameID, p)
			argsF := make([]interface{}, len(flowers))
			for i, v := range flowers {
				argsF[i] = v
			}
			if err := rdb.RPush(ctx, flowersKey, argsF...).Err(); err != nil {
				return fmt.Errorf("failed to RPush flowers: %w", err)
			}

			// 有花牌，需要補牌
			for i := 0; i < flowerCount; i++ {
				// 從嶺上 (LPop) 補牌
				data, err := rdb.LPop(ctx, deckKey).Result()
				if err != nil {
					return fmt.Errorf("failed to LPop for flower replacement: %w", err)
				}
				nonFlowers = append(nonFlowers, data)
			}

			// 更新手牌
			rdb.Del(ctx, playerKey)
			if len(nonFlowers) > 0 {
				args := make([]interface{}, len(nonFlowers))
				for i, v := range nonFlowers {
					args[i] = v
				}
				// 保持順序寫回
				if err := rdb.RPush(ctx, playerKey, args...).Err(); err != nil {
					return fmt.Errorf("failed to RPush new hand: %w", err)
				}
			}
		}
		// 理牌
		SortPlayerHand(ctx, gameID, p)
	}

	return nil
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
	state.IsAfterKong = false // 一旦出牌，取消「剛槓牌」狀態

	if err := SaveGameState(ctx, state); err != nil {
		return nil, err
	}
	return state, nil
}

// DrawTileAction 玩家摸牌 (Stage: PLAYER_DRAW → PLAYER_DISCARD)
func DrawTileAction(ctx context.Context, gameID string, playerID int) (*models.GameState, *models.Tile, error) {
	state, err := LoadGameState(ctx, gameID)
	if err != nil {
		return nil, nil, err
	}

	if state.Stage != models.StagePlayerDraw {
		return nil, nil, fmt.Errorf("action not allowed in current stage: %s", state.Stage)
	}

	if state.CurrentPlayerID != playerID {
		return nil, nil, fmt.Errorf("not your turn to draw, current player is %d", state.CurrentPlayerID)
	}

	// 檢查牌堆是否還有牌（荒莊流局檢查）
	deckCount, err := GetDeckCount(ctx, gameID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to check deck count: %w", err)
	}
	if deckCount == 0 {
		// 荒莊流局：牌堆已空
		state.Stage = models.StageRoundOver
		if err := SaveGameState(ctx, state); err != nil {
			return nil, nil, err
		}
		return state, nil, nil
	}

	// 從牌堆摸一張牌 (循環直到摸到非花牌)
	var drawnTile *models.Tile
	rdb := service.RedisClient
	for {
		dt, err := DrawTile(ctx, gameID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to draw tile: %w", err)
		}

		if dt.Type == models.Flower {
			utils.Info("[Flower] player%d draws a flower: %v during normal play, auto-replacing", playerID, dt)

			// 將花牌放入專屬 Redis List
			flowersKey := PlayerFlowersKey(gameID, playerID)
			dtJSON, _ := json.Marshal(dt)
			if err := rdb.RPush(ctx, flowersKey, string(dtJSON)).Err(); err != nil {
				return nil, nil, fmt.Errorf("failed to save drawn flower: %w", err)
			}

			// 從嶺上 (LPop) 補牌，並繼續檢查是否為花牌
			deckKey := DeckRedisKey(gameID)
			replacementJSON, err := rdb.LPop(ctx, deckKey).Result()
			if err != nil {
				return nil, nil, fmt.Errorf("failed to LPop for flower replacement: %w", err)
			}

			// 我們需要將拿到的字串反序列化，用作下一輪的 dt
			var rt models.Tile
			if err := json.Unmarshal([]byte(replacementJSON), &rt); err != nil {
				return nil, nil, fmt.Errorf("failed to unmarshal replacement tile: %w", err)
			}

			// 為了使用 DrawTile 的統一格式（DrawTile 本身呼叫 RPop），
			// 這裡直接將 rt 當作新摸到的牌在迴圈中繼續判定
			drawnTile = &rt
			continue
		}

		drawnTile = dt
		break
	}

	// 將最終摸到的(非花)牌加入玩家手牌 (透過 Redis LPUSH)
	tileJSON, _ := json.Marshal(drawnTile)
	playerKey := PlayerHandKey(gameID, playerID)
	if err := service.RedisClient.LPush(ctx, playerKey, string(tileJSON)).Err(); err != nil {
		return nil, nil, fmt.Errorf("failed to add drawn tile to hand: %w", err)
	}

	// 更新狀態機：進入出牌階段
	state.Stage = models.StagePlayerDiscard
	// CurrentPlayerID 維持不變（摸牌者接著出牌）

	if err := SaveGameState(ctx, state); err != nil {
		return nil, nil, err
	}
	return state, drawnTile, nil
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
		state, err = ResolveActions(ctx, gameID, state)
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
func ResolveActions(ctx context.Context, gameID string, state *models.GameState) (*models.GameState, error) {
	// 需求:
	// 1. 胡牌順位優先：下家 > 對家 > 上家
	// 2. 一炮三響：三人同時胡牌，各自獨立結算
	// 3. 胡大於碰/槓：有人宣告胡，即使其他人宣告碰或槓，一律以胡牌優先。

	var huPlayers []int
	var highestPriority int = -1
	var winnerID int = -1
	var winningAction string = "pass"

	priorityMap := map[string]int{
		"pass": 0,
		"chow": 1,
		"pong": 2,
		"kong": 3,
	}

	for pID, action := range state.ActionDeclarations {
		if action == "hu" {
			huPlayers = append(huPlayers, pID)
		} else {
			// 如果沒有人胡，則紀錄最高優先順序的碰/槓/吃
			p := priorityMap[action]
			if p > highestPriority {
				highestPriority = p
				winnerID = pID
				winningAction = action
			}
		}
	}

	if len(huPlayers) > 0 {
		// 有人胡牌，進入結算階段，忽略所有的碰/槓/吃
		state.Stage = models.StageRoundOver

		// 根據與出牌者 (LastDiscardPlayerID) 的距離進行排序：下家(1) > 對家(2) > 上家(3)
		// 距離算法: (pID - discarderID + 4) % 4
		discarderID := state.LastDiscardPlayerID
		if discarderID == 0 {
			discarderID = state.CurrentPlayerID // 自摸情況
		}

		// 簡單的排序邏輯 (如果只有 1 人就不用排)
		if len(huPlayers) > 1 && discarderID != 0 {
			for i := 0; i < len(huPlayers)-1; i++ {
				for j := i + 1; j < len(huPlayers); j++ {
					distI := (huPlayers[i] - discarderID + 4) % 4
					if distI == 0 {
						distI = 4
					} // 理論上不該發生，出牌者不能自己胡別人
					distJ := (huPlayers[j] - discarderID + 4) % 4
					if distJ == 0 {
						distJ = 4
					}

					if distI > distJ {
						huPlayers[i], huPlayers[j] = huPlayers[j], huPlayers[i]
					}
				}
			}
		}

		// 記錄所有的贏家與計算台數
		state.WinnerIDs = huPlayers
		state.CurrentPlayerID = huPlayers[0] // 向下相容，把第一順位放在 CurrentPlayerID

		rdb := service.RedisClient
		var winningTile models.Tile
		if discarderID == state.CurrentPlayerID {
			// 自摸，拿手牌最後一張作為WinningTile (實務上需更精確判定，此處簡化為最後摸的一張)
			// TODO: 自摸WinningTile的精確取得，我們這裡暫時給個預設，或需要從PlayerLastDraw取得
		} else {
			if state.LastDiscardTile != nil {
				winningTile = *(state.LastDiscardTile)
			}
		}

		for _, wid := range huPlayers {
			// 取得手牌
			hk := PlayerHandKey(gameID, wid)
			handJSONs, _ := rdb.LRange(ctx, hk, 0, -1).Result()
			var closedHand []models.Tile
			for _, hj := range handJSONs {
				var t models.Tile
				if json.Unmarshal([]byte(hj), &t) == nil {
					closedHand = append(closedHand, t)
				}
			}

			// 取得副露
			mk := PlayerMeldsKey(gameID, wid)
			meldJSONs, _ := rdb.LRange(ctx, mk, 0, -1).Result()
			var melds []models.Meld
			for _, mj := range meldJSONs {
				var m models.Meld
				if json.Unmarshal([]byte(mj), &m) == nil {
					melds = append(melds, m)
				}
			}

			// 取得花牌
			fk := PlayerFlowersKey(gameID, wid)
			flowerJSONs, _ := rdb.LRange(ctx, fk, 0, -1).Result()
			var flowers []models.Tile
			for _, fj := range flowerJSONs {
				var t models.Tile
				if json.Unmarshal([]byte(fj), &t) == nil {
					flowers = append(flowers, t)
				}
			}

			// 建構計分上下文
			scoreCtx := models.ScoringContext{
				ClosedHand:  closedHand,
				Melds:       melds,
				WinningTile: winningTile,
				IsSelfDrawn: discarderID == wid, // 如果出牌者是自己，代表是自摸
				IsDealer:    state.DealerPlayerID == wid,
				Flowers:     flowers,
			}

			scoreResult := models.CalculateScore(scoreCtx)

			// 寫入結算結果
			if state.ScoreResults == nil {
				state.ScoreResults = make(map[int]models.ScoreResult)
			}
			state.ScoreResults[wid] = scoreResult

			// 將結果存入玩家狀態 (需要擴充 GameState 結構，此處暫存至 log 或新欄位)
			utils.Info("[Scoring] Player %d Hu! TotalTai: %d, Patterns: %v", wid, scoreResult.TotalTai, scoreResult.Patterns)
		}

		return state, nil
	}

	if winningAction == "kong" || winningAction == "pong" || winningAction == "chow" {
		rdb := service.RedisClient
		meldsKey := PlayerMeldsKey(gameID, winnerID)
		targetTile := *(state.LastDiscardTile)
		var meld models.Meld

		switch winningAction {
		case "kong":
			// 明槓: 拿桌上一張，自己手上扣掉三張一樣的
			removed, err := RemoveTilesFromPlayerHand(ctx, gameID, winnerID, 3, targetTile.Type, targetTile.Value)
			if err == nil {
				meld = models.Meld{
					Type:  models.MeldTypeKong,
					Tiles: append(removed, targetTile),
				}
			}
		case "pong":
			// 碰: 拿桌上一張，自己手上扣掉兩張一樣的
			removed, err := RemoveTilesFromPlayerHand(ctx, gameID, winnerID, 2, targetTile.Type, targetTile.Value)
			if err == nil {
				meld = models.Meld{
					Type:  models.MeldTypePong,
					Tiles: append(removed, targetTile),
				}
			}
		case "chow":
			// TODO 吃牌邏輯：目前假設需要特定 API 來知道玩家要用哪兩張牌吃，暫時留空或簡單處理
			utils.Info("Chow is not fully implemented yet for outdesk")
		}

		if len(meld.Tiles) > 0 {
			meldJSON, _ := json.Marshal(meld)
			rdb.RPush(ctx, meldsKey, string(meldJSON))
		}

		// 若為槓牌，標記剛槓牌狀態 (供槓上開花判斷)，且需要從嶺上補一張牌
		if winningAction == "kong" {
			state.IsAfterKong = true

			deckKey := DeckRedisKey(gameID)
			replacementJSON, err := rdb.LPop(ctx, deckKey).Result()
			if err != nil {
				return nil, fmt.Errorf("failed to LPop for kong replacement: %w", err)
			}
			var rt models.Tile
			if err := json.Unmarshal([]byte(replacementJSON), &rt); err == nil {
				// 新牌加入手牌
				playerKey := PlayerHandKey(gameID, winnerID)
				rtJSON, _ := json.Marshal(rt)
				rdb.LPush(ctx, playerKey, string(rtJSON))
				utils.Info("Player%d Kong auto draw from tail: %v", winnerID, rt)
			}

		} else {
			state.IsAfterKong = false
		}

		state.Stage = models.StagePlayerDiscard // 碰/吃/槓完要打一張牌
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
		// 總共發出 65 張牌 (16張 * 4人 + 1張開門牌)
		drawn := make([]string, 65)
		for i := 0; i < 65; i++ {
			data, err := rdb.RPop(ctx, deckKey).Result()
			if err != nil {
				return fmt.Errorf("failed to pop tile %d: %w", i, err)
			}
			drawn[i] = data
		}

		playerHands := make(map[int][]string)
		for _, p := range order {
			playerHands[p] = make([]string, 0, 17)
		}

		// ----- 第一階段：輪流摸 4 張 × 4 輪 = 每人 16 張 -----
		idx := 0
		for round := 0; round < 4; round++ {
			for _, p := range order {
				playerHands[p] = append(playerHands[p], drawn[idx:idx+4]...)
				idx += 4
			}
		}

		// ----- 第二階段：莊家開門牌 -----
		// 莊家多拿最後第 65 張 (idx=64)
		playerHands[order[0]] = append(playerHands[order[0]], drawn[64])

		// 將分配好的牌一次推入玩家的 Redis Hand List
		for _, p := range order {
			playerKey := PlayerHandKey(gameID, p)
			args := make([]interface{}, len(playerHands[p]))
			for i, v := range playerHands[p] {
				args[i] = v
			}
			if err := rdb.LPush(ctx, playerKey, args...).Err(); err != nil {
				return fmt.Errorf("failed to push hand to player%d: %w", p, err)
			}
		}
	} else {
		// ===== 13張玩法 (真實跳牌與抓牌還原) =====
		// 總共發出 53 張牌 (12張 * 4人 + 1張 * 3閒家 + 2張莊家)
		drawn := make([]string, 53)
		for i := 0; i < 53; i++ {
			data, err := rdb.RPop(ctx, deckKey).Result()
			if err != nil {
				return fmt.Errorf("failed to pop tile %d: %w", i, err)
			}
			drawn[i] = data
		}

		playerHands := make(map[int][]string)
		for _, p := range order {
			playerHands[p] = make([]string, 0, 14)
		}

		// ----- 第一階段：輪流摸 4 張 × 3 輪 = 每人 12 張 -----
		// 順序：莊(order[0]) -> 南(order[1]) -> 西(order[2]) -> 北(order[3])
		idx := 0
		for round := 0; round < 3; round++ {
			for _, p := range order {
				playerHands[p] = append(playerHands[p], drawn[idx:idx+4]...)
				idx += 4
			}
		}

		// ----- 第二階段：莊家跳牌，閒家依序補牌 -----
		// 此時 idx = 48
		// 第 25 墩: drawn[48](上), drawn[49](下)
		// 第 26 墩: drawn[50](上), drawn[51](下)
		// 第 27 墩: drawn[52](上), ...

		// 莊家 (order[0]): 拿上層第一張(48) 和 上層第三張(52)
		playerHands[order[0]] = append(playerHands[order[0]], drawn[48], drawn[52])
		// 南家 (下家, order[1]): 拿上層第二張(50)
		playerHands[order[1]] = append(playerHands[order[1]], drawn[50])
		// 西家 (對家, order[2]): 拿下層第一張(49)
		playerHands[order[2]] = append(playerHands[order[2]], drawn[49])
		// 北家 (上家, order[3]): 拿下層第二張(51)
		playerHands[order[3]] = append(playerHands[order[3]], drawn[51])

		// 將分配好的牌一次推入玩家的 Redis Hand List
		for _, p := range order {
			playerKey := PlayerHandKey(gameID, p)
			args := make([]interface{}, len(playerHands[p]))
			// 由於 Redis LPUSH 會把最後一個參數放在 List 最前面，
			// 我們維持順序依次放入（這不影響最終的排序，因為會呼叫 SortPlayerHand）
			for i, v := range playerHands[p] {
				args[i] = v
			}
			if err := rdb.LPush(ctx, playerKey, args...).Err(); err != nil {
				return fmt.Errorf("failed to push hand to player%d: %w", p, err)
			}
		}
	}

	// ----- 理牌 -----
	for p := 1; p <= 4; p++ {
		if err := SortPlayerHand(ctx, gameID, p); err != nil {
			return fmt.Errorf("sort player%d hand failed: %w", p, err)
		}
	}

	return nil
}
