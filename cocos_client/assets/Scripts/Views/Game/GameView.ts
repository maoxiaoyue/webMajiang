import { _decorator, Node, Label } from 'cc';
import { BaseView } from '../BaseView';
import { GameViewModel } from '../../ViewModels/Game/GameViewModel';
import { EventMgr } from '../../Events/EventMgr';
import { TileRenderer, TileDisplayMode } from '../../Core/TileRenderer';

const { ccclass, property } = _decorator;

/**
 * GameView: 麻將主遊戲房間的視覺統籌
 */
@ccclass('GameView')
export class GameView extends BaseView {

    @property(Node)
    public centerInfoPanel: Node = null!;

    @property(Label)
    public remainingTilesLabel: Label = null!;

    @property(Label)
    public currentWindLabel: Label = null!;

    // 玩家手牌容器 (0: 本機玩家, 1: 右邊, 2: 對面, 3: 左邊)
    @property([Node])
    public playerHandNodes: Node[] = [];

    // 玩家棄牌區容器
    @property([Node])
    public playerDiscardNodes: Node[] = [];

    private handTileRenderers: TileRenderer[] = [];

    protected onLoad(): void {
        this.viewModel = new GameViewModel();
        super.onLoad();
    }

    protected bindViewModel() {
        EventMgr.on("game_state_changed", this.onGameStateChanged, this);
        EventMgr.on("player_hand_changed", this.onPlayerHandChanged, this);
        EventMgr.on("deal_tiles_received", this.onDealTilesReceived, this);
    }

    protected onDestroy(): void {
        EventMgr.targetOff(this);
        super.onDestroy();
    }

    // ============================================
    // 手牌渲染
    // ============================================

    /**
     * 渲染本機玩家的手牌（正面 + 上側面，有 3D 深度感）
     */
    public renderHandTiles(tileIds: number[]) {
        const handNode = this.playerHandNodes[0];
        if (!handNode) return;

        handNode.removeAllChildren();
        this.handTileRenderers = [];

        for (const tileId of tileIds) {
            const tileNode = TileRenderer.createStandingTile(tileId);
            handNode.addChild(tileNode);

            const renderer = tileNode.getComponent(TileRenderer)!;
            this.handTileRenderers.push(renderer);

            tileNode.on(Node.EventType.TOUCH_END, () => {
                this.onTileClicked(tileId, tileNode);
            });
        }
    }

    /**
     * 渲染對手手牌（背面 + 上側面）
     */
    public renderOpponentHand(seatIndex: number, count: number) {
        const handNode = this.playerHandNodes[seatIndex];
        if (!handNode) return;

        handNode.removeAllChildren();
        for (let i = 0; i < count; i++) {
            const tileNode = TileRenderer.createOpponentTile();
            handNode.addChild(tileNode);
        }
    }

    /**
     * 渲染棄牌區（正面平放）
     */
    public renderDiscardTiles(seatIndex: number, tileIds: number[]) {
        const discardNode = this.playerDiscardNodes[seatIndex];
        if (!discardNode) return;

        discardNode.removeAllChildren();
        for (const tileId of tileIds) {
            const tileNode = TileRenderer.createDiscardTile(tileId);
            discardNode.addChild(tileNode);
        }
    }

    // ============================================
    // 事件處理
    // ============================================

    private onGameStateChanged(modelInfo: any) {
        if (this.remainingTilesLabel && modelInfo.remainingTiles != null) {
            this.remainingTilesLabel.string = `剩餘: ${modelInfo.remainingTiles}`;
        }
        if (this.currentWindLabel && modelInfo.currentWind != null) {
            const windNames = ["東", "南", "西", "北"];
            this.currentWindLabel.string = windNames[modelInfo.currentWind] || "東";
        }

        if (modelInfo.players) {
            for (const player of modelInfo.players) {
                if (player.seat === 0) {
                    this.renderHandTiles(player.handTiles || []);
                } else {
                    this.renderOpponentHand(player.seat, (player.handTiles || []).length);
                }
                this.renderDiscardTiles(player.seat, player.discardedTiles || []);
            }
        }
    }

    private onPlayerHandChanged(data: { playerId: string, tiles: number[] }) {
        this.renderHandTiles(data.tiles);
    }

    private onDealTilesReceived(tiles: number[]) {
        const handNode = this.playerHandNodes[0];
        if (!handNode) return;

        for (const tileId of tiles) {
            const tileNode = TileRenderer.createStandingTile(tileId);
            handNode.addChild(tileNode);

            const renderer = tileNode.getComponent(TileRenderer)!;
            this.handTileRenderers.push(renderer);

            tileNode.on(Node.EventType.TOUCH_END, () => {
                this.onTileClicked(tileId, tileNode);
            });
        }
    }

    // ============================================
    // 互動
    // ============================================

    private selectedTileNode: Node | null = null;

    private onTileClicked(tileId: number, tileNode: Node) {
        if (this.selectedTileNode === tileNode) {
            this.onBtnDiscardClicked(tileId);
            this.selectedTileNode = null;
            return;
        }

        if (this.selectedTileNode) {
            this.selectedTileNode.setPosition(
                this.selectedTileNode.position.x, 0, 0
            );
        }

        this.selectedTileNode = tileNode;
        tileNode.setPosition(tileNode.position.x, 15, 0);
    }

    public onBtnDiscardClicked(tileId: number) {
        const success = (this.viewModel as GameViewModel).attemptDiscardTile(tileId);
        if (success && this.selectedTileNode) {
            this.selectedTileNode.removeFromParent();
            this.selectedTileNode = null;
        }
    }
}
