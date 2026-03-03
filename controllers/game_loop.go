package controllers

import (
	"context"
	"fmt"
	"time"

	"webmajiang/models"
	"webmajiang/utils"
)

// RunPostDiscard 出牌後觸發的遊戲推進邏輯
// 1. 收集 AI 玩家的宣告 (pass/pong/hu)
// 2. 若三家都表態完畢（或有人胡），自動結算
// 3. 結算後推進到下一階段
func RunPostDiscard(ctx context.Context, gameID string) (*models.GameState, error) {
	state, err := LoadGameState(ctx, gameID)
	if err != nil {
		return nil, err
	}

	if state.Stage != models.StageWaitAction {
		return nil, fmt.Errorf("RunPostDiscard: expected WAIT_ACTION stage, got %s", state.Stage)
	}

	// 收集所有 AI 玩家的自動宣告
	for pID := 1; pID <= 4; pID++ {
		if pID == state.LastDiscardPlayerID {
			continue // 出牌者不參與宣告
		}

		player, ok := state.Players[pID]
		if !ok {
			continue
		}

		if !player.IsBot {
			continue // 真人玩家需透過 WebSocket 手動宣告
		}

		// AI 自動判斷要宣告什麼
		aiAction, err := ProcessAIResponse(ctx, gameID, pID, state.LastDiscardTile)
		if err != nil {
			utils.Error("[GameLoop] AI 玩家 %d 宣告失敗: %v", pID, err)
			aiAction = "pass"
		}

		utils.Info("[GameLoop] AI 玩家 %d 宣告: %s", pID, aiAction)

		// 透過 PlayerDeclareAction 記錄宣告
		state, err = PlayerDeclareAction(ctx, gameID, pID, aiAction)
		if err != nil {
			return nil, fmt.Errorf("AI player %d declare failed: %w", pID, err)
		}

		// 若有人胡了，PlayerDeclareAction 會自動觸發 ResolveActions
		if state.Stage == models.StageRoundOver {
			return state, nil
		}
	}

	// 檢查是否還需要等待真人玩家宣告
	if state.Stage == models.StageWaitAction {
		// 計算已宣告人數（排除出牌者的其他 3 家）
		declared := len(state.ActionDeclarations)
		if declared < 3 {
			// 還有真人玩家尚未宣告，需要等待
			return state, nil
		}
	}

	// 所有人都已表態，結算已在 PlayerDeclareAction 中自動觸發
	// 接著推進到下一階段
	return RunPostResolve(ctx, gameID)
}

// RunPostResolve 結算完畢後推進下一階段
// 根據結算結果：
//   - 全 pass → PLAYER_DRAW: 若下家是 AI 則自動摸牌+出牌
//   - 碰/吃 → PLAYER_DISCARD: 若得標者是 AI 則自動出牌
//   - 胡 → ROUND_OVER: 不做任何事
func RunPostResolve(ctx context.Context, gameID string) (*models.GameState, error) {
	state, err := LoadGameState(ctx, gameID)
	if err != nil {
		return nil, err
	}

	switch state.Stage {
	case models.StageRoundOver:
		utils.Info("[GameLoop] 遊戲結束 (ROUND_OVER)")
		return state, nil

	case models.StagePlayerDraw:
		// 下家需要摸牌
		nextPlayer, ok := state.Players[state.CurrentPlayerID]
		if !ok {
			return nil, fmt.Errorf("player %d not found", state.CurrentPlayerID)
		}

		if nextPlayer.IsBot {
			utils.Info("[GameLoop] 下家是 AI 玩家 %d，自動摸牌+出牌...", nextPlayer.ID)
			return runAIDrawAndDiscard(ctx, gameID, nextPlayer)
		}

		// 真人玩家需透過 WebSocket 手動摸牌
		utils.Info("[GameLoop] 輪到真人玩家 %d 摸牌，等待 WebSocket draw_tile...", nextPlayer.ID)
		return state, nil

	case models.StagePlayerDiscard:
		// 碰/吃得標者需要出牌
		winner, ok := state.Players[state.CurrentPlayerID]
		if !ok {
			return nil, fmt.Errorf("player %d not found", state.CurrentPlayerID)
		}

		if winner.IsBot {
			utils.Info("[GameLoop] 碰/吃得標者是 AI 玩家 %d，自動出牌...", winner.ID)
			return runAIDiscard(ctx, gameID, winner)
		}

		// 真人玩家需透過 WebSocket 手動出牌
		utils.Info("[GameLoop] 碰/吃得標者是真人玩家 %d，等待 WebSocket discard_tile...", winner.ID)
		return state, nil

	default:
		return state, nil
	}
}

// ProcessAIResponse AI 對「他人出牌」的自動回應
// 目前邏輯：
//   - 如果 AI 能胡，宣告 "hu"
//   - 如果 AI 有對子（能碰），宣告 "pong"
//   - 否則 "pass"
func ProcessAIResponse(ctx context.Context, gameID string, playerID int, discardedTile *models.Tile) (string, error) {
	if discardedTile == nil {
		return "pass", nil
	}

	// 從 Redis 取得 AI 手牌
	hand, err := GetPlayerHand(ctx, gameID, playerID)
	if err != nil {
		return "pass", fmt.Errorf("failed to get AI hand: %w", err)
	}

	// 1. 檢查是否能胡 (將被打出的牌加入手牌判斷)
	testHand := append([]models.Tile{}, hand...)
	testHand = append(testHand, *discardedTile)
	if models.CanHu(testHand) {
		return "hu", nil
	}

	// 2. 檢查是否能碰 (手牌中有兩張同樣的牌)
	matchCount := 0
	for _, t := range hand {
		if t.Type == discardedTile.Type && t.Value == discardedTile.Value {
			matchCount++
		}
	}
	if matchCount >= 2 {
		return "pong", nil
	}

	// 3. 其他情況 pass
	return "pass", nil
}

// runAIDrawAndDiscard AI 玩家的完整摸牌+出牌流程
func runAIDrawAndDiscard(ctx context.Context, gameID string, player models.Player) (*models.GameState, error) {
	// 模擬 AI 思考時間
	time.Sleep(1 * time.Second)

	// 1. 摸牌
	state, drawnTile, err := DrawTileAction(ctx, gameID, player.ID)
	if err != nil {
		return nil, fmt.Errorf("AI 摸牌失敗: %w", err)
	}

	// 荒莊流局
	if drawnTile == nil {
		utils.Info("[AI Turn] 牌堆已空，荒莊流局")
		return state, nil
	}

	utils.Info("[AI Turn] 玩家 %d 摸到了 %s", player.ID, drawnTile.String())

	// 2. 取得最新手牌
	hand, err := GetPlayerHand(ctx, gameID, player.ID)
	if err != nil {
		return nil, fmt.Errorf("取得 AI 手牌失敗: %w", err)
	}

	// 3. 檢查是否自摸
	if models.CanHu(hand) {
		utils.Info("[AI Turn] 🌟 玩家 %d 自摸了！", player.ID)
		state, err = LoadGameState(ctx, gameID)
		if err != nil {
			return nil, err
		}
		state.Stage = models.StageRoundOver
		state.CurrentPlayerID = player.ID
		if err := SaveGameState(ctx, state); err != nil {
			return nil, err
		}
		return state, nil
	}

	time.Sleep(500 * time.Millisecond)

	// 4. 選擇最佳出牌
	return runAIDiscard(ctx, gameID, models.Player{ID: player.ID, Name: player.Name, IsBot: true, Hand: hand})
}

// runAIDiscard AI 玩家出牌並觸發後續流程
func runAIDiscard(ctx context.Context, gameID string, player models.Player) (*models.GameState, error) {
	// 取得最新手牌（如果 Hand 為空）
	if len(player.Hand) == 0 {
		hand, err := GetPlayerHand(ctx, gameID, player.ID)
		if err != nil {
			return nil, fmt.Errorf("取得 AI 手牌失敗: %w", err)
		}
		player.Hand = hand
	}

	discardTile := models.GetBestDiscard(player.Hand)
	utils.Info("[AI Turn] 玩家 %d 決定丟出 %s", player.ID, discardTile.String())

	if _, err := DiscardTileAction(ctx, gameID, player.ID, discardTile); err != nil {
		return nil, fmt.Errorf("AI 丟牌失敗: %w", err)
	}

	// 出牌後自動推進（收集其他 AI 宣告等）
	return RunPostDiscard(ctx, gameID)
}
