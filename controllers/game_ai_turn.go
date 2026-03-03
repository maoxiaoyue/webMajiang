package controllers

import (
	"context"
	"fmt"

	"webmajiang/models"
	"webmajiang/utils"
)

// ProcessAITurn 處理 AI 玩家的回合（從摸牌到出牌的完整流程）
// 此函式由 game_loop 中的 runAIDrawAndDiscard 取代日常調用
// 保留作為獨立的 AI 回合入口（例如首次莊家出牌等情境）
func ProcessAITurn(ctx context.Context, gameID string, p models.Player) error {
	utils.Info("[AI Turn] 輪到 AI 玩家 %d (%s) 動作...", p.ID, p.Name)

	state, err := LoadGameState(ctx, gameID)
	if err != nil {
		return fmt.Errorf("載入遊戲狀態失敗: %w", err)
	}

	switch state.Stage {
	case models.StagePlayerDraw:
		// AI 需要摸牌 + 出牌
		_, err := runAIDrawAndDiscard(ctx, gameID, p)
		if err != nil {
			return err
		}

	case models.StagePlayerDiscard:
		// AI 只需出牌（例如莊家發牌後直接出牌、或碰/吃後出牌）
		_, err := runAIDiscard(ctx, gameID, p)
		if err != nil {
			return err
		}

	default:
		return fmt.Errorf("AI 回合不應在 %s 階段觸發", state.Stage)
	}

	return nil
}
