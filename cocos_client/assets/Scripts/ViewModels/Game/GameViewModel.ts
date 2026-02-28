import { BaseViewModel } from '../BaseViewModel';
import { GameModel } from '../../Models/Game/GameModel';
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

        // 可在此處呼叫 connect 或是由外部統一管理連線生命週期
        // NetworkMgr.instance.connect("ws://localhost:8080/ws");
    }

    private onServerConnected() {
        console.log("[GameViewModel] 伺服器已連線，可發送加入房間等請求");
        // NetworkMgr.instance.send("join_room", { roomId: "123" });
    }

    private onServerDisconnected() {
        console.warn("[GameViewModel] 伺服器斷線");
    }

    private handleSyncState(data: any) {
        this.model.updateGameState(data);
    }

    private handleDealTiles(data: any) {
        // 處理發牌更新邏輯
    }

    /**
     * 玩家嘗試出牌 (來自 View 層的互動)
     */
    public attemptDiscardTile(tileId: number): boolean {
        // 1. 檢查是否輪到自己
        if (this.model.currentTurnPlayerId !== "my_player_id") {
            console.warn("還沒輪到你出牌！");
            return false;
        }

        // 2. 將出牌指令送往伺服器
        NetworkMgr.instance.send("discard_tile", { tile: tileId });

        return true;
    }
}
