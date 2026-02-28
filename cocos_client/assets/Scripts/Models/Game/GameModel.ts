import { BaseModel } from '../BaseModel';

export interface PlayerData {
    id: string;
    name: string;
    seat: number; // 0: East, 1: South, 2: West, 3: North (相對位置)
    score: number;
    handTiles: number[]; // 手牌 ID 陣列
    discardedTiles: number[]; // 打出的牌
    meldTiles: any[]; // 吃碰槓的牌
}

/**
 * 麻將主遊戲的資料模型
 */
export class GameModel extends BaseModel {

    // 房間資訊
    public roomId: string = "";
    public currentWind: number = 0; // 圈風 (0: 東, 1: 南, 2: 西, 3: 北)
    public remainingTiles: number = 144; // 剩餘牌數

    // 玩家清單
    public players: PlayerData[] = [];

    // 遊戲狀態
    public currentTurnPlayerId: string = ""; // 當前出牌玩家
    public gameState: string = "waiting"; // waiting, playing, complete

    /**
     * 更新整個遊戲狀態
     */
    public updateGameState(state: any) {
        // TODO: 依照伺服器回傳格式進行賦值
        this.emitChange("game_state_changed", this);
    }

    /**
     * 更新特定玩家手牌
     */
    public updatePlayerHand(playerId: string, tiles: number[]) {
        const player = this.players.find(p => p.id === playerId);
        if (player) {
            player.handTiles = tiles;
            this.emitChange("player_hand_changed", { playerId, tiles });
        }
    }
}
