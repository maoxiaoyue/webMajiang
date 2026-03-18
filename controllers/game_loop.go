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

// RunPostDiscard 出牌後觸發的遊戲推進邏輯
// 1. 收集 AI 玩家的宣告 (pass/pong/chow/hu)
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

	// 收集所有玩家的自動宣告（AI 全自動，真人無可用動作時也自動 pass）
	for pID := 1; pID <= 4; pID++ {
		if pID == state.LastDiscardPlayerID {
			continue // 出牌者不參與宣告
		}

		player, ok := state.Players[pID]
		if !ok {
			continue
		}

		// 判斷該玩家可以做什麼
		action, err := ProcessAIResponse(ctx, gameID, pID, state.LastDiscardTile, state.LastDiscardPlayerID)
		if err != nil {
			utils.Error("[GameLoop] 玩家 %d 宣告判斷失敗: %v", pID, err)
			action = "pass"
		}

		if !player.IsBot && action != "pass" {
			// 真人玩家有可用動作（碰/胡），需等待手動選擇
			utils.Info("[GameLoop] 真人玩家 %d 可 %s，等待 WebSocket player_action...", pID, action)
			// 啟動催促定時器：8秒後 Bot 催促
			go nudgeHumanPlayer(gameID, pID)
			continue
		}

		// AI 或真人無動作 → 自動宣告
		if player.IsBot {
			utils.Info("[GameLoop] AI 玩家 %d 宣告: %s", pID, action)
		} else {
			utils.Info("[GameLoop] 真人玩家 %d 無可用動作，自動 pass", pID)
			action = "pass"
		}

		// 透過 PlayerDeclareAction 記錄宣告
		state, err = PlayerDeclareAction(ctx, gameID, pID, action)
		if err != nil {
			return nil, fmt.Errorf("player %d declare failed: %w", pID, err)
		}

		// 若已結算（胡/碰/槓優先導致立即結算），不再繼續宣告
		if state.Stage != models.StageWaitAction {
			break
		}
	}

	// 如果還在 WAIT_ACTION 階段，檢查是否還需要等待真人玩家
	if state.Stage == models.StageWaitAction {
		declared := len(state.ActionDeclarations)
		if declared < 3 {
			// 還有真人玩家尚未宣告（有可用動作），需要等待
			return state, nil
		}
	}

	// 已結算完畢（全pass/碰/槓/胡），推進到下一階段
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

		// 廣播當前狀態（讓前端看到上家出牌或 pass 結果）
		syncData := buildSyncStateData(gameID, state)
		if globalHub != nil {
			sendProtoBroadcast(globalHub, "sync_state", syncData)
		}

		// 每家之間至少間隔 1 秒
		time.Sleep(1 * time.Second)

		if nextPlayer.IsBot {
			utils.Info("[GameLoop] 下家是 AI 玩家 %d，自動摸牌+出牌...", nextPlayer.ID)
			return runAIDrawAndDiscard(ctx, gameID, nextPlayer)
		}

		// 真人玩家也自動摸牌，只需等待出牌
		utils.Info("[GameLoop] 輪到真人玩家 %d，自動摸牌...", nextPlayer.ID)
		newState, drawnTile, err := DrawTileAction(ctx, gameID, nextPlayer.ID)
		if err != nil {
			return nil, fmt.Errorf("真人玩家 %d 自動摸牌失敗: %w", nextPlayer.ID, err)
		}
		if drawnTile == nil {
			utils.Info("[GameLoop] 牌堆已空，荒莊流局")
			return newState, nil
		}
		utils.Info("[GameLoop] 真人玩家 %d 摸到了 %s，等待出牌...", nextPlayer.ID, drawnTile.String())
		// 檢查真人玩家是否有暗槓機會（前端會顯示暗槓按鈕）
		humanHand, _ := GetPlayerHand(ctx, gameID, nextPlayer.ID)
		if kongTiles := FindConcealedKongs(humanHand); len(kongTiles) > 0 {
			for _, kt := range kongTiles {
				utils.Info("[GameLoop] 真人玩家 %d 可暗槓: %s %d", nextPlayer.ID, kt.Type.String(), kt.Value)
			}
		}
		return newState, nil

	case models.StagePlayerDiscard:
		// 碰/吃得標者需要出牌
		winner, ok := state.Players[state.CurrentPlayerID]
		if !ok {
			return nil, fmt.Errorf("player %d not found", state.CurrentPlayerID)
		}

		if winner.IsBot {
			utils.Info("[GameLoop] 碰/吃得標者是 AI 玩家 %d，自動出牌...", winner.ID)
			// 碰/槓後的 Bot 發話
			speech := service.GetRandomConversation("pong", "bot")
			if speech != "" {
				BroadcastBotSpeech(winner.ID, speech)
			}
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
// 邏輯：
//   - 如果 AI 能胡，宣告 "hu"
//   - 如果 AI 有對子（能碰），宣告 "pong"
//   - 如果是下家且能吃（順子），宣告 "chow"
//   - 否則 "pass"
func ProcessAIResponse(ctx context.Context, gameID string, playerID int, discardedTile *models.Tile, discarderID int) (string, error) {
	if discardedTile == nil {
		return "pass", nil
	}

	// 從 Redis 取得手牌
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

	// 3. 檢查是否能吃 (只有下家可以吃，且僅限萬筒條)
	isNextPlayer := (discarderID%4)+1 == playerID
	if isNextPlayer && CanChow(hand, discardedTile) {
		return "chow", nil
	}

	// 4. 其他情況 pass
	return "pass", nil
}

// CanChow 檢查手牌是否能吃指定的牌
// 規則：只有萬(Wan)、筒(Tong)、條(Tiao) 才能吃，風牌和三元牌不能吃
// 需要手牌中有兩張能和被吃的牌組成順子的牌
func CanChow(hand []models.Tile, discardedTile *models.Tile) bool {
	if discardedTile == nil {
		return false
	}
	// 風牌、三元牌、花牌不能吃
	if discardedTile.Type != models.Wan && discardedTile.Type != models.Tong && discardedTile.Type != models.Tiao {
		return false
	}

	v := discardedTile.Value
	t := discardedTile.Type

	// 收集手牌中同花色的數值集合
	hasValue := make(map[int]bool)
	for _, tile := range hand {
		if tile.Type == t {
			hasValue[tile.Value] = true
		}
	}

	// 三種順子組合：
	// (v-2, v-1, v) — 例如 v=5: 手中有 3,4
	if v >= 3 && hasValue[v-2] && hasValue[v-1] {
		return true
	}
	// (v-1, v, v+1) — 例如 v=5: 手中有 4,6
	if v >= 2 && v <= 8 && hasValue[v-1] && hasValue[v+1] {
		return true
	}
	// (v, v+1, v+2) — 例如 v=5: 手中有 6,7
	if v <= 7 && hasValue[v+1] && hasValue[v+2] {
		return true
	}

	return false
}

// FindChowTiles 找到手牌中可以和被吃的牌組成順子的兩張牌
// 回傳第一組可用的吃牌組合
func FindChowTiles(hand []models.Tile, discardedTile *models.Tile) ([]models.Tile, bool) {
	if discardedTile == nil {
		return nil, false
	}
	if discardedTile.Type != models.Wan && discardedTile.Type != models.Tong && discardedTile.Type != models.Tiao {
		return nil, false
	}

	v := discardedTile.Value
	t := discardedTile.Type

	// 建立同花色牌的索引 (value -> []Tile)
	byValue := make(map[int][]models.Tile)
	for _, tile := range hand {
		if tile.Type == t {
			byValue[tile.Value] = append(byValue[tile.Value], tile)
		}
	}

	// 嘗試三種順子組合，回傳第一組找到的
	combos := [][2]int{
		{v - 2, v - 1}, // (v-2, v-1, v)
		{v - 1, v + 1}, // (v-1, v, v+1)
		{v + 1, v + 2}, // (v, v+1, v+2)
	}

	for _, combo := range combos {
		a, b := combo[0], combo[1]
		if a < 1 || a > 9 || b < 1 || b > 9 {
			continue
		}
		tilesA := byValue[a]
		tilesB := byValue[b]
		if len(tilesA) > 0 && len(tilesB) > 0 {
			return []models.Tile{tilesA[0], tilesB[0]}, true
		}
	}

	return nil, false
}

// FindAddKong 檢查手牌中是否有牌可以和已碰的副露組成加槓
// 回傳第一張可加槓的牌，或 nil
func FindAddKong(hand []models.Tile, ctx context.Context, gameID string, playerID int) *models.Tile {
	// 取得已碰的副露
	rdb := service.RedisClient
	meldsKey := PlayerMeldsKey(gameID, playerID)
	meldJSONs, _ := rdb.LRange(ctx, meldsKey, 0, -1).Result()

	var pongTypes []struct {
		Type  models.TileType
		Value int
	}
	for _, mj := range meldJSONs {
		var m models.Meld
		if json.Unmarshal([]byte(mj), &m) != nil {
			continue
		}
		if m.Type == models.MeldTypePong && len(m.Tiles) > 0 {
			pongTypes = append(pongTypes, struct {
				Type  models.TileType
				Value int
			}{m.Tiles[0].Type, m.Tiles[0].Value})
		}
	}

	// 檢查手牌中是否有和碰副露相同的牌
	for _, t := range hand {
		for _, pt := range pongTypes {
			if t.Type == pt.Type && t.Value == pt.Value {
				tile := t // copy
				return &tile
			}
		}
	}
	return nil
}

// runAIDrawAndDiscard AI 玩家的完整摸牌+出牌流程
func runAIDrawAndDiscard(ctx context.Context, gameID string, player models.Player) (*models.GameState, error) {
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

	// 2. 廣播 Bot 說話 + 等待 1.5 秒
	speech := service.GetRandomConversation("draw", "bot")
	if speech != "" {
		BroadcastBotSpeech(player.ID, speech)
		utils.Info("[AI Turn] 玩家 %d 說: %s", player.ID, speech)
	}

	// 廣播摸牌後的狀態（讓前端看到輪到此 Bot）
	syncData := buildSyncStateData(gameID, state)
	if globalHub != nil {
		sendProtoBroadcast(globalHub, "sync_state", syncData)
	}

	time.Sleep(1500 * time.Millisecond)

	// 3. 取得最新手牌
	hand, err := GetPlayerHand(ctx, gameID, player.ID)
	if err != nil {
		return nil, fmt.Errorf("取得 AI 手牌失敗: %w", err)
	}

	// 4. 檢查暗槓
	kongTiles := FindConcealedKongs(hand)
	if len(kongTiles) > 0 {
		for _, kt := range kongTiles {
			utils.Info("[AI Turn] 🀄 玩家 %d 暗槓: %s %d", player.ID, kt.Type.String(), kt.Value)
			speechK := service.GetRandomConversation("pong", "bot")
			if speechK != "" {
				BroadcastBotSpeech(player.ID, speechK)
			}
			var err error
			state, err = ConcealedKongAction(ctx, gameID, player.ID, kt.Type, kt.Value)
			if err != nil {
				utils.Error("[AI Turn] 暗槓失敗: %v", err)
				break
			}
			// 廣播暗槓後狀態
			syncData2 := buildSyncStateData(gameID, state)
			if globalHub != nil {
				sendProtoBroadcast(globalHub, "sync_state", syncData2)
			}
			time.Sleep(1000 * time.Millisecond)

			// 暗槓後重新取得手牌
			hand, err = GetPlayerHand(ctx, gameID, player.ID)
			if err != nil {
				return nil, fmt.Errorf("暗槓後取得手牌失敗: %w", err)
			}
		}
	}

	// 5. 檢查加槓 (手牌中有一張牌和已碰的副露相同)
	addKongTile := FindAddKong(hand, ctx, gameID, player.ID)
	if addKongTile != nil {
		utils.Info("[AI Turn] 🀄 玩家 %d 加槓: %s %d", player.ID, addKongTile.Type.String(), addKongTile.Value)
		speechK := service.GetRandomConversation("pong", "bot")
		if speechK != "" {
			BroadcastBotSpeech(player.ID, speechK)
		}
		state, err = AddKongAction(ctx, gameID, player.ID, addKongTile.Type, addKongTile.Value)
		if err != nil {
			utils.Error("[AI Turn] 加槓失敗: %v", err)
		} else {
			syncData3 := buildSyncStateData(gameID, state)
			if globalHub != nil {
				sendProtoBroadcast(globalHub, "sync_state", syncData3)
			}
			time.Sleep(1000 * time.Millisecond)
			// 加槓後重新取得手牌
			hand, err = GetPlayerHand(ctx, gameID, player.ID)
			if err != nil {
				return nil, fmt.Errorf("加槓後取得手牌失敗: %w", err)
			}
		}
	}

	// 6. 檢查是否自摸
	if models.CanHu(hand) {
		utils.Info("[AI Turn] 🌟 玩家 %d 自摸了！", player.ID)
		// 先喊「自摸」，廣播狀態讓大家看到，再結算
		BroadcastBotSpeech(player.ID, "自摸！")
		// 廣播當前狀態（讓前端先看到誰在喊自摸）
		if curState, loadErr := LoadGameState(ctx, gameID); loadErr == nil {
			syncData := buildSyncStateData(gameID, curState)
			if globalHub != nil {
				sendProtoBroadcast(globalHub, "sync_state", syncData)
			}
		}
		time.Sleep(2 * time.Second)

		state, err = LoadGameState(ctx, gameID)
		if err != nil {
			return nil, err
		}
		state.Stage = models.StageRoundOver
		state.CurrentPlayerID = player.ID
		state.WinnerIDs = []int{player.ID}
		state.IsSelfDrawnWin = true
		if err := SaveGameState(ctx, state); err != nil {
			return nil, err
		}
		return state, nil
	}

	// 6. 選擇最佳出牌
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

	// 廣播當前狀態（讓前端看到碰/槓後的手牌變化）
	if curState, err := LoadGameState(ctx, gameID); err == nil {
		syncData := buildSyncStateData(gameID, curState)
		if globalHub != nil {
			sendProtoBroadcast(globalHub, "sync_state", syncData)
		}
	}

	// 出牌前等待 1 秒，讓玩家看到碰/摸牌等動作
	time.Sleep(1 * time.Second)

	discardTile := models.GetBestDiscard(player.Hand)
	utils.Info("[AI Turn] 玩家 %d 決定丟出 %s", player.ID, discardTile.String())

	// 廣播 Bot 出牌說話
	speech := service.GetRandomConversation("discard", "bot")
	if speech != "" {
		BroadcastBotSpeech(player.ID, speech)
	}

	if _, err := DiscardTileAction(ctx, gameID, player.ID, discardTile); err != nil {
		return nil, fmt.Errorf("AI 丟牌失敗: %w", err)
	}

	// 出牌後自動推進（收集其他 AI 宣告等）
	resultState, err := RunPostDiscard(ctx, gameID)
	if err != nil {
		return nil, err
	}

	// 如果進入 WAIT_ACTION（等待真人玩家碰/吃/胡），廣播狀態讓前端顯示按鈕
	if resultState != nil && resultState.Stage == models.StageWaitAction {
		syncData := buildSyncStateData(gameID, resultState)
		if globalHub != nil {
			sendProtoBroadcast(globalHub, "sync_state", syncData)
		}
	}

	return resultState, nil
}

// nudgeHumanPlayer 催促真人玩家做決定
// 在 20 秒後檢查玩家是否已宣告，若未宣告則由 Bot 催促（使用 conversation_dict type='push'）
func nudgeHumanPlayer(gameID string, humanPlayerID int) {
	time.Sleep(20 * time.Second)

	ctx := context.Background()
	state, err := LoadGameState(ctx, gameID)
	if err != nil {
		return
	}

	// 遊戲已不在 WAIT_ACTION 階段，不需催促
	if state.Stage != models.StageWaitAction {
		return
	}

	// 玩家已經宣告了，不需催促
	if _, declared := state.ActionDeclarations[humanPlayerID]; declared {
		return
	}

	// 從 conversation_dict 取得催促語 (type='push', role='bot')
	speech := service.GetRandomConversation("push", "bot")
	if speech == "" {
		speech = "快一點啦~" // fallback
	}

	// 找一個 Bot 來說話催促
	for _, p := range state.Players {
		if p.IsBot {
			BroadcastBotSpeech(p.ID, speech)
			utils.Info("[Nudge] Bot %d 催促真人玩家 %d: %s", p.ID, humanPlayerID, speech)
			break
		}
	}
}
