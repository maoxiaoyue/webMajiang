# 功能實作完成：登入、註冊與在線清單更新

以下是我們已完成的實作項目回顧，這些功能已經完整匯入遊戲伺服器中，並準備好被前端呼叫。

## 已經新增/修改的 API
- `POST /api/auth/register`
  接收 `{ "email": "...", "password": "..." }`。
  會建立使用者（先標記為未驗證）、寄送含有驗證 Token 的信件（終端機印出除錯訊息），並將 Token 寫入 Redis。
- `GET /api/auth/verify?token=...`
  驗證給定 Token，通過後，將該使用者信箱標記為 `is_verified: true`。
- `POST /api/auth/login`
  接收 `{ "email": "...", "password": "..." }`。
  通過帳密與 `is_verified` 檢查後，回傳 JWT Token，並立即將使用者以 `SADD user:online <id>:<username>` 加到線上清單，且設定 KeyDB 特有的 `EXPIREMEMBER user:online 600 <id>:<username>` （10分鐘）。

## 其餘核心改動
- **Middleware**: 新增 [middlewares/auth.go](file:///d:/GoProjects/webMajiangGame/middlewares/auth.go) ([AuthRequired](file:///d:/GoProjects/webMajiangGame/middlewares/auth.go#14-44))。需要驗證 Header 有 `Authorization: Bearer <jwt-token>`，通過後會重置此人在 `user:online` 的倒數 10 分鐘機制。
- **WebSocket 整合**: 修改 [controllers/ws_handler.go](file:///d:/GoProjects/webMajiangGame/controllers/ws_handler.go) 裡的各個動作與連線時機（包含 `join_room`, `deal_tiles`, `draw_tile`, `discard_tile` 等等...），皆會自動提取 [JoinRoomReq](file:///d:/GoProjects/webMajiangGame/proto/mahjong.proto#64-68) 或 `player_id`，觸發 [keepOnline(id)](file:///d:/GoProjects/webMajiangGame/controllers/ws_handler.go#17-31) 重置過期時間，確保任何遊戲動作皆會維持 10 分鐘上線狀態。
- **組態 ([config.yaml](file:///d:/GoProjects/webMajiangGame/config/config.yaml))**: 加入了 `smtp` 與 `jwt.secret` 預設空欄位，並於 [main.go](file:///d:/GoProjects/webMajiangGame/main.go) 完成初始化。

## 驗證計畫執行結果
1. [models/user_test.go](file:///d:/GoProjects/webMajiangGame/models/user_test.go) 單元測試順利通過，驗證 `models` 中的 [User](file:///d:/GoProjects/webMajiangGame/models/user.go#21-28) 新增、查閱、及 [KeepUserOnline](file:///d:/GoProjects/webMajiangGame/models/user.go#141-156) (`EXPIREMEMBER` 指令) 等機制皆正常運作。
2. `go build ./...` 確認目前所有檔案編譯正常零錯誤。
3. 寄信功能在開發階段會自動 Fallback 印出到終端機 (Test Mode Log)，以便您快速點擊連結驗證通過，不需要強硬綁定真實 SMTP，方便本地開發體驗。
