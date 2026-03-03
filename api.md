# 麻將遊戲 API 與 WebSocket 接口文檔 (Mahjong API & WebSocket Documentation)

為了配合「前一個步驟未完成前，不得進行下一個步驟」的安全狀態機（State Machine）流程卡控機制，這份文件整理了目前所有可用的後端接口與其內部運作邏輯。

---

## 遊戲狀態機流程圖

```
WAITING_PLAYERS → DETERMINE_POSITIONS → DETERMINE_DEALER → DEALING
       ↓                                                       ↓
  (join_room)                                           (deal_tiles)
                                                              ↓
    ┌──────────────── PLAYER_DRAW ←── (全 pass) ── WAIT_ACTION
    │                     ↓                              ↑
    │    (draw_tile)   摸牌+檢查自摸              (discard_tile)
    │                     ↓                              │
    │              PLAYER_DISCARD ────── 出牌 ────────────┘
    │                     ↑
    │           (碰/吃得標者出牌)
    │
    └→ (荒莊/自摸/胡牌) → ROUND_OVER → (next_round) → DEALING / GAME_OVER
```

### AI 自動化流程

當輪到 AI 玩家時，系統會自動執行以下動作：

- **WAIT_ACTION 階段**: AI 自動判斷是否要碰/胡/pass（由 `ProcessAIResponse` 處理）
- **PLAYER_DRAW 階段**: AI 自動摸牌、檢查自摸、選擇最佳出牌（由 `runAIDrawAndDiscard` 處理）
- **PLAYER_DISCARD 階段**: AI 自動選擇出牌（由 `runAIDiscard` 處理）

真人玩家則需透過 WebSocket 手動送出指令。

---

## 1. REST API (遊戲初始化)

### **開始新房間 (Start Game)**
- **接口位置**: `POST /api/game/start`
- **用途**: 新建一組麻將對戰回合並產生唯一的 `game_id`。
- **邏輯設計**:
  1. 建立具有唯一 `game_id` 的 Redis [GameState](file:///d:/GoProjects/webMajiangGame/models/game_round.go#L140-L154)。
  2. 初始化局號為第一局 (東風東 1-1)。
  3. 將遊戲階段設為 **`WAITING_PLAYERS`**。
  4. 回傳 `game_id` 給客戶端，供他連線 WebSocket 時使用。

---

## 2. WebSocket 事件 (主要遊戲流程)

WebSocket 使用 Protobuf 格式傳輸。外層統一為 `WSMessage { action, data }`，`data` 為對應 Protobuf 訊息的序列化資料。

### **狀態機流程指令**
以下指令嚴格跟隨狀態機推演，**若上一階段未完成，伺服器將拒絕請求。**

#### (1) 決定座位 — `roll_positions`
- **Data**: `JoinRoomReq { room_id }`
- **可用階段**: `WAITING_PLAYERS`
- **邏輯**: 擲骰子決定方位，Stage → `DETERMINE_DEALER`

#### (2) 決定莊家 — `roll_dealer`
- **Data**: `JoinRoomReq { room_id }`
- **可用階段**: `DETERMINE_DEALER`
- **邏輯**: 擲骰子決定莊家（`DealerPlayerID`），Stage → `DEALING`

#### (3) 洗牌與發牌 — `deal_tiles`
- **Data**: `JoinRoomReq { room_id, player_id }`
- **可用階段**: `DEALING`
- **邏輯**:
  1. ChaCha20 洗牌 → 發牌 → 理牌
  2. 莊家拿多一張開門牌
  3. Stage → `PLAYER_DISCARD`，`CurrentPlayerID` = 莊家
  4. **若莊家是 AI，自動觸發出牌流程**
- **回傳**: 請求者的手牌 tile ID list

#### (4) 玩家摸牌 — `draw_tile`
- **Data**: `JoinRoomReq { room_id, player_id }`
- **可用階段**: `PLAYER_DRAW`
- **邏輯**:
  1. 驗證輪到該玩家
  2. 從牌堆 RPOP 一張牌加入手牌
  3. 檢查牌堆是否為空（荒莊流局 → `ROUND_OVER`）
  4. Stage → `PLAYER_DISCARD`
- **回傳**: 摸到的牌 ID

#### (5) 玩家出牌 — `discard_tile`
- **Data**: `PlayerActionData { tile_id }`
- **可用階段**: `PLAYER_DISCARD`
- **邏輯**:
  1. 驗證輪到該玩家
  2. 從手牌中移除指定牌
  3. Stage → `WAIT_ACTION`
  4. **自動觸發 `RunPostDiscard`**: 收集 AI 宣告 → 結算 → 推進
- **回傳**: 出牌結果

#### (6) 玩家宣告 — `player_action`
- **Data**: `PlayerActionData { action_type, tile_id }`
  - `action_type`: 2=吃, 3=碰, 4=槓, 5=胡, 6=過
- **可用階段**: `WAIT_ACTION`
- **邏輯**:
  1. 記錄宣告
  2. 若有人胡或三家都表態 → 自動結算 (`ResolveActions`)
  3. 結算後自動觸發 `RunPostResolve`（推進 AI 動作）
- **結算優先權**: hu > kong/pong > chow > pass

#### (7) 進入下一局 — `next_round`
- **Data**: `JoinRoomReq { room_id }`
- **可用階段**: `ROUND_OVER`
- **邏輯**: 推進局號，莊家順轉，Stage → `DEALING`。若已到 4-4 (北風北) → `GAME_OVER`

---

### **資訊與輔助指令 (不受階段限制)**

#### (1) 手牌排序 — `sort_hand`
- **Data**: `JoinRoomReq { room_id, player_id }`
- **邏輯**: 從 Redis 讀取手牌 → 排序 → 重新寫回
- **回傳**: 排序後的 tile ID list

#### (2) 取得當前遊戲狀態 — `get_state`
- **Data**: `JoinRoomReq { room_id }`
- **回傳**: `SyncStateData` (Stage, CurrentPlayerID, DealerPlayerID, 圈風等)

#### (3) 取得當前牌桌存牌 — `get_hands`
- **Data**: `JoinRoomReq { room_id }`
- **回傳**: 各家 1-4 號玩家的手牌與長度

#### (4) 取得牌堆剩餘數量 — `get_deck_count`
- **Data**: `JoinRoomReq { room_id }`
- **回傳**: `{"deck_count": N}`
