# 麻將遊戲 API 與 WebSocket 接口文檔 (Mahjong API & WebSocket Documentation)

為了配合「前一個步驟未完成前，不得進行下一個步驟」的安全狀態機（State Machine）流程卡控機制，這份文件整理了目前所有可用的後端接口與其內部運作邏輯。

---

## 1. REST API (遊戲初始化)

### **開始新房間 (Start Game)**
- **接口位置**: `POST /api/game/start`
- **用途**: 新建一組麻將對戰回合並產生唯一的 `game_id`。
- **邏輯設計**:
  1. 建立具有唯一 `game_id` 的 Redis [GameState](file:///d:/GoProjects/webMajiangGame/models/game_round.go#132-146)。
  2. 初始化局號為第一局 (東風東 1-1)。
  3. 將遊戲階段 ([Stage](file:///d:/GoProjects/webMajiangGame/models/game_round.go#117-118)) 設為 **`WAITING_PLAYERS`**。
  4. 回傳 `game_id` 給客戶端，供他連線 WebSocket 時使用。

---

## 2. WebSocket 事件 (主要遊戲流程)

WebSocket 的負載格式 (Payload) 通常為 JSON 物件，必須帶有 `"game_id"` 以及 `"type"`（指令）。部分事件會額外需要 `"data"` 物件。

### **狀態機流程指令**
以下指令嚴格跟隨狀態機推演，**若上一階段未完成，伺服器將拒絕請求。**

#### (1) 決定座位
- **發送格式**: `{"game_id": "majiang_xxx", "type": "roll_positions"}`
- **可用階段**: `WAITING_PLAYERS`
- **用途/邏輯**: 模擬遊戲一開始決定方位的投擲動作。擲出後伺服器會將 [Stage](file:///d:/GoProjects/webMajiangGame/models/game_round.go#117-118) 推移至 `DETERMINE_DEALER`。

#### (2) 決定莊家
- **發送格式**: `{"game_id": "majiang_xxx", "type": "roll_dealer"}`
- **可用階段**: `DETERMINE_DEALER`
- **用途/邏輯**: 針對入座玩家擲骰子決定東風（莊家）位置 (`DealerPlayerID`)。回傳擲骰點數，並將 [Stage](file:///d:/GoProjects/webMajiangGame/models/game_round.go#117-118) 推展至 `DEALING`。

#### (3) 洗牌與發牌
- **發送格式**: `{"game_id": "majiang_xxx", "type": "deal_tiles"}`
- **可用階段**: `DEALING`
- **用途/邏輯**:
  1. 建立 144 張牌的牌堆並使用 ChaCha20 加密級別洗牌。
  2. 將牌存入 Redis ([DeckRedisKey](file:///d:/GoProjects/webMajiangGame/controllers/deck.go#137-141))。
  3. 給 4 家分別發牌 (其他家 13 張，莊家 14 張)。
  4. 由於莊家預設開門拿 14 張，理牌建檔完成後，自動將 `CurrentPlayerID` 指派給莊家。
  5. 狀態機進入 **`PLAYER_DISCARD`** 階段，準備讓莊家出牌。

#### (4) 玩家出牌
- **發送格式**: 
  ```json
  {
    "game_id": "majiang_xxx", 
    "type": "discard_tile",
    "data": {
      "player_id": 1,
      "tile": {"id": 12, "type": 1, "value": 3}
    }
  }
  ```
- **可用階段**: `PLAYER_DISCARD`
- **用途/邏輯**:
  1. **驗證身分**: 確定發送請求的人(`player_id`)是否等於目前輪廓的玩家(`CurrentPlayerID`)。
  2. 從該玩家在 Redis 的手牌清單中確實刪除該張實體牌。
  3. 將該撲克牌指派為丟出的廢牌 (`LastDiscardTile`)，並將 [Stage](file:///d:/GoProjects/webMajiangGame/models/game_round.go#117-118) 變更為 **`WAIT_ACTION`**。
  4. 各家客戶端畫面轉為等待/跳出「吃、碰、槓、胡」選單。

#### (5) 等待宣告 / 競標結算
- **發送格式**: 
  ```json
  {
    "game_id": "majiang_xxx", 
    "type": "player_action",
    "data": {
      "player_id": 2,
      "action": "pong" // 可用選項: "pass", "chow", "pong", "kong", "hu"
    }
  }
  ```
- **可用階段**: `WAIT_ACTION`
- **用途/邏輯**:
  1. 伺服器會記錄每家投出的票。
  2. **自動觸發結算**: 若有人喊 `"hu"` 或三家都表態完畢，系統開啟決策比較。由大到小: [hu](file:///d:/GoProjects/webMajiangGame/controllers/deck.go#111-126) > `kong`/`pong` > `chow` > `pass`。
  3. 若無人要牌 (`pass`): `CurrentPlayerID` 換給原打牌者的「下家」，狀態推移至 **`PLAYER_DRAW`**。
  4. 若有人吃/碰/槓 (`chow`/`pong`/`kong`): `CurrentPlayerID` 變更為得標者，狀態則切換回 **`PLAYER_DISCARD`**。

#### (6) 玩家摸牌 (從牌堆拿一張牌)
- **發送格式**: `{"game_id": "majiang_xxx", "type": "draw_tile"}`
- **可用階段**: `PLAYER_DRAW` **(注意：目前尚未加上嚴格安全鎖，可以後續補上該接口的安全卡控)**
- **用途/邏輯**: 從 Redis 牌堆頂部抽出一張牌放進手牌中，狀態變回 `PLAYER_DISCARD`。

---

### **資訊與輔助指令 (不受階段限制)**

#### (1) 進入下一局 (Next Round)
- **發送格式**: `{"game_id": "majiang_xxx", "type": "next_round"}`
- **用途/邏輯**:
  單局結算後 (`ROUND_OVER` 階段) 呼叫，系統會推進局數（如從東風東 1-1 進入 東風南 1-2）。
  決定是否連莊後，將 [Stage](file:///d:/GoProjects/webMajiangGame/models/game_round.go#117-118) 切回 `DEALING`，隨時等待再次洗牌。若來到 4-4 (北風北) 後打完，則回傳結束標記 `finished: true`。

#### (2) 取得當前遊戲狀態
- **發送格式**: `{"game_id": "majiang_xxx", "type": "get_state"}`
- **用途/邏輯**: 回傳 [GameState](file:///d:/GoProjects/webMajiangGame/models/game_round.go#132-146) 重點資訊（包含 [Stage](file:///d:/GoProjects/webMajiangGame/models/game_round.go#117-118), `CurrentPlayerID`, `DealerPlayerID`）。中斷連線重入房時可呼叫此接口回復畫面。

#### (3) 取得當前牌桌存牌
- **發送格式**: `{"game_id": "majiang_xxx", "type": "get_hands"}`
- **用途/邏輯**: 查詢目前各家 1-4 號玩家的 Redis 牌池與長度 (Count)。

#### (4) 取得牌堆剩餘數量
- **發送格式**: `{"game_id": "majiang_xxx", "type": "get_deck_count"}`
- **用途/邏輯**: 回傳整場 144 張牌摸到剩幾張 (用於判定荒莊流局)，回傳 `"deck_count": 87` 等數值。
