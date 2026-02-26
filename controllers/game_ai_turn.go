package controllers

import (
	"context"
	"fmt"
	"time"

	"webmajiang/models"
)

// ProcessAITurn 處理 AI 玩家的回合
// 1. 摸牌 (DrawTile)
// 2. 判斷是否自摸 (CanHu)
// 3. 找出最沒用的牌 (models.GetBestDiscard)
// 4. 丟牌 (TODO: 未實作丟牌寫入 Redis 流程，先印出 Log)
func ProcessAITurn(ctx context.Context, gameID string, p models.Player) error {
	fmt.Printf("[AI Turn] 輪到 AI 玩家 %d (%s) 動作...\n", p.ID, p.Name)

	// 模擬 AI 思考時間 (增加真實感與前端動畫時間)
	time.Sleep(1 * time.Second)

	// 1. 摸牌
	drawnTile, err := DrawTile(ctx, gameID)
	if err != nil {
		return fmt.Errorf("AI 摸牌失敗: %w", err)
	}

	fmt.Printf("[AI Turn] 玩家 %d 摸到了 %s\n", p.ID, drawnTile.String())

	// 更新該玩家的手牌資料 (從 Redis 重新獲取最新手牌，包含剛摸的那張)
	hands, err := GetPlayerHand(ctx, gameID, p.ID)
	if err != nil {
		return fmt.Errorf("取得 AI 手牌失敗: %w", err)
	}
	p.Hand = hands

	// 2. 檢查是否自摸
	if models.CanHu(p.Hand) {
		fmt.Printf("[AI Turn] 🌟 玩家 %d 自摸了！\n", p.ID)
		// TODO: 觸發胡牌結束遊戲流程
		return nil
	}

	time.Sleep(500 * time.Millisecond)

	// 3. 判斷要丟掉哪一張牌
	discardTile := models.GetBestDiscard(p.Hand)
	fmt.Printf("[AI Turn] 玩家 %d 決定丟出 %s\n", p.ID, discardTile.String())

	// 4. TODO: 實作丟牌至海底，並切換到下一位玩家
	// DiscardTile(ctx, gameID, p.ID, discardTile)

	return nil
}
