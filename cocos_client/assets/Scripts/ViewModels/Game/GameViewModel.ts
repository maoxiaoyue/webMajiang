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
        this.model.roomId = data.roomId || "";
        this.model.currentWind = data.currentWind || 0;
        this.model.remainingTiles = data.remainingTiles ?? 144;
        this.model.currentTurnPlayerId = data.currentTurnPlayerId || "";
        this.model.gameState = data.gameState || "waiting";

        if (data.players && Array.isArray(data.players)) {
            this.model.players = data.players.map((p: any) => ({
                id: p.id || "",
                name: p.name || "",
                seat: p.seat || 0,
                score: p.score || 0,
                handTiles: p.handTiles || [],
                discardedTiles: p.discardedTiles || [],
                meldTiles: p.melds || []
            } as PlayerData));
        }

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
        if (this.model.currentTurnPlayerId !== "my_player_id") {
            console.warn("還沒輪到你出牌！");
            return false;
        }

        NetworkMgr.instance.send("discard_tile", { tileId: tileId, actionType: 1 });
        return true;
    }
}
