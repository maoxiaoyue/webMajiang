import { _decorator, Node, Label, Sprite } from 'cc';
import { BaseView } from '../BaseView';
import { GameViewModel } from '../../ViewModels/Game/GameViewModel';
import { EventMgr } from '../../Events/EventMgr';

const { ccclass, property } = _decorator;

/**
 * GameView: 麻將主遊戲房間的視覺統籌
 */
@ccclass('GameView')
export class GameView extends BaseView {

    @property(Node)
    public centerInfoPanel: Node = null!; // 中央資訊區塊 (風向、剩餘牌數)

    @property(Label)
    public remainingTilesLabel: Label = null!;

    @property(Label)
    public currentWindLabel: Label = null!;

    // 玩家手牌區塊陣列 (0: 本機玩家, 1: 右邊, 2: 對面, 3: 左邊)
    @property([Node])
    public playerHandNodes: Node[] = [];

    protected onLoad(): void {
        // 例項化對應的 ViewModel
        this.viewModel = new GameViewModel();

        super.onLoad(); // 呼叫父類以觸發 bindViewModel()
    }

    /**
     * 綁定 ViewModel 與 EventMgr 事件
     */
    protected bindViewModel() {
        EventMgr.on("game_state_changed", this.onGameStateChanged, this);
        EventMgr.on("player_hand_changed", this.onPlayerHandChanged, this);
    }

    protected onDestroy(): void {
        EventMgr.targetOff(this); // 移除所有綁定的事件
        super.onDestroy();
    }

    /**
     * 當遊戲整體狀態變更時更新畫面 (由 Model 出發 -> EventMgr 轉發)
     */
    private onGameStateChanged(modelInfo: any) {
        console.log("Game state changed:", modelInfo);
        if (this.remainingTilesLabel) {
            this.remainingTilesLabel.string = `剩餘: ${modelInfo.remainingTiles}`;
        }
    }

    /**
     * 當某個玩家手牌發生改變時更新畫面
     */
    private onPlayerHandChanged(data: { playerId: string, tiles: number[] }) {
        console.log(`Player ${data.playerId} hand updated:`, data.tiles);
        // TODO: 依照對應座位的手牌節點 (playerHandNodes) 重新生成麻將牌的 Prefab
    }

    /**
     * UI 點擊事件：當玩家選擇了一張手牌並點擊「出牌」
     */
    public onBtnDiscardClicked(tileId: number) {
        // 將 UI 操作委託給 ViewModel 處理業務邏輯
        const success = (this.viewModel as GameViewModel).attemptDiscardTile(tileId);

        if (success) {
            // 如需播放本機出牌動畫可在此執行
        }
    }
}
