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

// DetermineDealerByDice 根據擲骰結果決定莊家座位
// 骰子點數總和對 4 取餘：
//
//	1 → 座位 0 (東/player1)
//	2 → 座位 1 (南/player2)
//	3 → 座位 2 (西/player3)
//	0 → 座位 3 (北/player4)
func DetermineDealerByDice(diceTotal int) int {
	seat := diceTotal % 4
	if seat == 0 {
		seat = 4
	}
	return seat - 1 // 轉為 0-indexed
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
// 1. 建立遊戲狀態 (東風東 1-1)
// 2. 擲骰子決定第一局莊家
// 3. 初始化牌堆 → 洗牌 → LPUSH 到 Redis
// 4. 發牌 + 理牌
func StartNewGame(ctx context.Context, gameID string) (*models.GameState, error) {
	// 1. 擲骰子
	dice, err := RollDice()
	if err != nil {
		return nil, fmt.Errorf("dice roll failed: %w", err)
	}

	// 2. 決定莊家座位
	dealerSeat := DetermineDealerByDice(dice.Total)

	// 3. 建立遊戲狀態
	state := &models.GameState{
		GameID:     gameID,
		Round:      models.NewFirstRound(), // 東風東 (1-1)
		DealerSeat: dealerSeat,
		Dice:       dice,
		IsStarted:  true,
		IsFinished: false,
	}

	// 4. 儲存狀態
	if err := SaveGameState(ctx, state); err != nil {
		return nil, err
	}

	// 5. 初始化牌堆
	if err := InitDeckToRedis(ctx, gameID); err != nil {
		return nil, fmt.Errorf("init deck failed: %w", err)
	}

	// 6. 發牌（以莊家座位為起始）
	if err := DealTilesFromSeat(ctx, gameID, dealerSeat); err != nil {
		return nil, fmt.Errorf("deal tiles failed: %w", err)
	}

	return state, nil
}

// NextRound 進入下一局
// 1. 讀取目前狀態
// 2. 計算下一局局號
// 3. 莊家輪轉（門風變更時莊家不變，連莊另外處理）
// 4. 重新建牌堆 → 洗牌 → 發牌
func NextRound(ctx context.Context, gameID string) (*models.GameState, bool, error) {
	// 1. 讀取目前狀態
	state, err := LoadGameState(ctx, gameID)
	if err != nil {
		return nil, false, err
	}

	// 2. 計算下一局
	nextRound, isComplete := state.Round.NextRound()
	if isComplete {
		state.IsFinished = true
		if err := SaveGameState(ctx, state); err != nil {
			return nil, true, err
		}
		return state, true, nil // 一將結束
	}

	// 3. 更新局號，莊家座位順轉（下家做莊）
	state.Round = nextRound
	state.DealerSeat = (state.DealerSeat + 1) % 4

	// 4. 儲存更新後的狀態
	if err := SaveGameState(ctx, state); err != nil {
		return nil, false, err
	}

	// 5. 重新建牌堆
	if err := InitDeckToRedis(ctx, gameID); err != nil {
		return nil, false, fmt.Errorf("init deck failed: %w", err)
	}

	// 6. 發牌
	if err := DealTilesFromSeat(ctx, gameID, state.DealerSeat); err != nil {
		return nil, false, fmt.Errorf("deal tiles failed: %w", err)
	}

	return state, false, nil
}

// DealTilesFromSeat 從指定莊家座位開始發牌
// 發牌順序以莊家為起始，逆時針輪流 (莊→下家→對家→上家)
// 座位順序: dealerSeat, (dealer+1)%4, (dealer+2)%4, (dealer+3)%4
func DealTilesFromSeat(ctx context.Context, gameID string, dealerSeat int) error {
	rdb := service.RedisClient
	deckKey := DeckRedisKey(gameID)

	// 計算發牌順序
	order := [4]int{
		dealerSeat,
		(dealerSeat + 1) % 4,
		(dealerSeat + 2) % 4,
		(dealerSeat + 3) % 4,
	}

	// 先清除舊的玩家手牌
	for _, p := range order {
		playerKey := PlayerHandKey(gameID, p)
		if err := rdb.Del(ctx, playerKey).Err(); err != nil {
			return fmt.Errorf("failed to clear player%d hand: %w", p+1, err)
		}
	}

	// ----- 第一階段：輪流摸 4 張 × 3 輪 -----
	for round := 0; round < 3; round++ {
		for _, p := range order {
			playerKey := PlayerHandKey(gameID, p)
			for t := 0; t < 4; t++ {
				data, err := rdb.RPop(ctx, deckKey).Result()
				if err != nil {
					return fmt.Errorf("round %d, player%d, tile %d: RPOP failed: %w", round+1, p+1, t+1, err)
				}
				if err := rdb.LPush(ctx, playerKey, data).Err(); err != nil {
					return fmt.Errorf("round %d, player%d, tile %d: LPUSH failed: %w", round+1, p+1, t+1, err)
				}
			}
		}
	}

	// ----- 第二階段：每人再摸 1 張 -----
	for _, p := range order {
		playerKey := PlayerHandKey(gameID, p)
		data, err := rdb.RPop(ctx, deckKey).Result()
		if err != nil {
			return fmt.Errorf("final draw, player%d: RPOP failed: %w", p+1, err)
		}
		if err := rdb.LPush(ctx, playerKey, data).Err(); err != nil {
			return fmt.Errorf("final draw, player%d: LPUSH failed: %w", p+1, err)
		}
	}

	// ----- 第三階段：莊家再摸 1 張開門 -----
	dealerKey := PlayerHandKey(gameID, dealerSeat)
	data, err := rdb.RPop(ctx, deckKey).Result()
	if err != nil {
		return fmt.Errorf("dealer open tile: RPOP failed: %w", err)
	}
	if err := rdb.LPush(ctx, dealerKey, data).Err(); err != nil {
		return fmt.Errorf("dealer open tile: LPUSH failed: %w", err)
	}

	// ----- 第四階段：理牌 -----
	for p := 0; p < 4; p++ {
		if err := SortPlayerHand(ctx, gameID, p); err != nil {
			return fmt.Errorf("sort player%d hand failed: %w", p+1, err)
		}
	}

	return nil
}
