import { BaseViewModel } from '../BaseViewModel';
import { GameModel, PlayerData } from '../../Models/Game/GameModel';
import { EventMgr } from '../../Events/EventMgr';
import { NetworkMgr } from '../../Core/NetworkMgr';

/**
 * 麻將主遊戲的邏輯層 ViewModel
 */
export class GameViewModel extends BaseViewModel<GameModel> {

    constructor() {
        super(new GameModel());
        // 從全域讀取自己的 player ID
        const selfId = (window as any).__SELF_PLAYER_ID__ || '1';
        this.model.selfPlayerId = selfId;
        this.initServerListeners();
    }

    /**
     * 初始化與伺服器或底層網路層的監聽
     */
    private initServerListeners() {
        // 監聽來自 NetworkMgr 的動態事件 (對應 WebSocket 返回的 `action` 欄位)
        EventMgr.on("ws_sync_state", this.handleSyncState, this);
        EventMgr.on("ws_deal_tiles", this.handleDealTiles, this);
        EventMgr.on(NetworkMgr.EVENT_CONNECTED, this.onServerConnected, this);
        EventMgr.on(NetworkMgr.EVENT_DISCONNECTED, this.onServerDisconnected, this);
    }

    private onServerConnected() {
        console.log("[GameViewModel] 伺服器已連線，可發送加入房間等請求");
    }

    private onServerDisconnected() {
        console.warn("[GameViewModel] 伺服器斷線");
    }

    private handleSyncState(data: any) {
        if (!data) return;
        // protobuf decoder 回傳 snake_case 欄位名
        this.model.roomId = data.room_id || "";
        this.model.currentWind = data.current_wind || 0;
        this.model.remainingTiles = data.remaining_tiles ?? 144;
        this.model.currentTurnPlayerId = data.current_turn_player_id || "";
        // 處理 GameState 可能是帶有 scoreResults 的 JSON 字串情況
        let rawGameState = data.game_state || "waiting";
        this.model.scoreResults = null;
        if (rawGameState.startsWith("{")) {
            try {
                const parsed = JSON.parse(rawGameState);
                this.model.gameState = parsed.stage || "complete";
                this.model.scoreResults = parsed.score_results || null;

                if (this.model.scoreResults) {
                    console.log("[GameViewModel] Received ScoreResults:", this.model.scoreResults);
                }
            } catch (e) {
                console.warn("[GameViewModel] failed to parse gameState JSON", e);
                this.model.gameState = rawGameState;
            }
        } else {
            this.model.gameState = rawGameState;
        }

        this.model.winnerIds = data.winner_ids || [];

        let lastDiscardedTileId = -1;
        if (data.players && Array.isArray(data.players)) {
            // 找到自己的絕對座位
            const selfPlayer = data.players.find((p: any) => p.id === this.model.selfPlayerId);
            if (selfPlayer) {
                this.model.selfSeatIndex = selfPlayer.seat || 0;
            }
            const selfSeat = this.model.selfSeatIndex >= 0 ? this.model.selfSeatIndex : 0;

            // 在 WAIT_ACTION 階段，currentTurnPlayerId 還是剛出牌的玩家
            const currentPlayer = data.players.find((p: any) => p.id === data.current_turn_player_id);
            if (currentPlayer && currentPlayer.discarded_tiles && currentPlayer.discarded_tiles.length > 0) {
                lastDiscardedTileId = currentPlayer.discarded_tiles[currentPlayer.discarded_tiles.length - 1];
            }

            this.model.players = data.players.map((p: any) => {
                const absSeat = p.seat || 0;
                // 計算相對座位: 0=自己, 1=右(下家), 2=對面, 3=左(上家)
                const relativeSeat = (absSeat - selfSeat + 4) % 4;
                return {
                    id: p.id || "",
                    name: p.name || "",
                    seat: relativeSeat,
                    score: p.score || 0,
                    handTiles: p.hand_tiles || [],
                    discardedTiles: p.discarded_tiles || [],
                    melds: (p.melds || []).map((m: any) => ({
                        type: m.type || 0,
                        tiles: m.tiles || []
                    })),
                    flowers: p.flowers || []
                } as PlayerData;
            });
        }

        this.model.lastDiscardedTileId = lastDiscardedTileId;
        this.model.updateGameState(data);
    }

    private handleDealTiles(data: any) {
        if (!data || !data.tiles) return;
        // 將新摸到的牌加入到本機玩家手牌
        EventMgr.emit("deal_tiles_received", data.tiles);
    }

    /**
     * 玩家嘗試出牌 (來自 View 層的互動)
     */
    public attemptDiscardTile(tileId: number): boolean {
        if (this.model.currentTurnPlayerId !== this.model.selfPlayerId) {
            console.warn(`還沒輪到你出牌！(current=${this.model.currentTurnPlayerId}, self=${this.model.selfPlayerId})`);
            return false;
        }

        NetworkMgr.instance.send("discard_tile", { action_type: 1, tile_id: tileId });
        return true;
    }
}
