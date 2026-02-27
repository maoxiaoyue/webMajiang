package controllers

import (
	"context"
	"encoding/json"
	"fmt"

	"webmajiang/models"
	"webmajiang/service"
)

// GameStateKey 遊戲狀態在 Redis 中的 key
func GameStateKey(gameID string) string {
	return fmt.Sprintf("game:%s:state", gameID)
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
func StartNewGame(ctx context.Context, gameID string) (*models.GameState, error) {
	state := &models.GameState{
		GameID:          gameID,
		Stage:           models.StageWaitingPlayers,
		CurrentPlayerID: 0,
		Round:           models.NewFirstRound(), // 東風東 (1-1)
		DealerPlayerID:  0,
		IsStarted:       true,
		IsFinished:      false,
		Players:         make(map[int]models.Player),
	}

	// 暫時設定為「Seat 1 為真人玩家，Seat 2,3,4 為 AI 玩家」
	state.Players[1] = models.Player{ID: 1, Name: "真人玩家1", IsBot: false, Hand: []models.Tile{}}
	state.Players[2] = models.Player{ID: 2, Name: "AI 電腦1", IsBot: true, Hand: []models.Tile{}}
	state.Players[3] = models.Player{ID: 3, Name: "AI 電腦2", IsBot: true, Hand: []models.Tile{}}
	state.Players[4] = models.Player{ID: 4, Name: "AI 電腦3", IsBot: true, Hand: []models.Tile{}}

	if err := SaveGameState(ctx, state); err != nil {
		return nil, err
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
	return state, nil
}

// DealTilesAction 執行發牌流程 (含洗牌、發牌、理牌)
func DealTilesAction(ctx context.Context, gameID string) (*models.GameState, error) {
	state, err := LoadGameState(ctx, gameID)
	if err != nil {
		return nil, err
	}
	if state.Stage != models.StageDealing {
		return nil, fmt.Errorf("action not allowed in current stage: %s", state.Stage)
	}

	if err := InitDeckToRedis(ctx, gameID); err != nil {
		return nil, fmt.Errorf("init deck failed: %w", err)
	}

	if err := DealTilesFromSeat(ctx, gameID, state.DealerPlayerID); err != nil {
		return nil, fmt.Errorf("deal tiles failed: %w", err)
	}

	// 莊家已拿 14 張，其他 13 張，故接下來換莊家打牌
	state.Stage = models.StagePlayerDiscard
	state.CurrentPlayerID = state.DealerPlayerID

	if err := SaveGameState(ctx, state); err != nil {
		return nil, err
	}
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

	return state, false, nil
}

// DealTilesFromSeat 從指定莊家玩家 ID 開始發牌
// 發牌順序以莊家為起始，逆時針輪流 (莊→下家→對家→上家)
// 座位順序: dealerPlayerID, dealerPlayerID下家 等 (1-4循環)
func DealTilesFromSeat(ctx context.Context, gameID string, dealerPlayerID int) error {
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

	// ----- 第一階段：輪流摸 4 張 × 3 輪 -----
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

	// ----- 第二階段：每人再摸 1 張 -----
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

	// ----- 第三階段：莊家再摸 1 張開門 -----
	dealerKey := PlayerHandKey(gameID, dealerPlayerID)
	data, err := rdb.RPop(ctx, deckKey).Result()
	if err != nil {
		return fmt.Errorf("dealer open tile: RPOP failed: %w", err)
	}
	if err := rdb.LPush(ctx, dealerKey, data).Err(); err != nil {
		return fmt.Errorf("dealer open tile: LPUSH failed: %w", err)
	}

	// ----- 第四階段：理牌 -----
	for p := 1; p <= 4; p++ {
		if err := SortPlayerHand(ctx, gameID, p); err != nil {
			return fmt.Errorf("sort player%d hand failed: %w", p, err)
		}
	}

	return nil
}
