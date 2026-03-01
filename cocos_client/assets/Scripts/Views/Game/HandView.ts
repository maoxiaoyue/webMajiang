import { _decorator, Component, Node } from 'cc';
import { TileRenderer } from '../../Core/TileRenderer';
import { EventMgr } from '../../Events/EventMgr';
import { NetworkMgr } from '../../Core/NetworkMgr';

const { ccclass, property } = _decorator;

/** 手牌槽間距 (px) */
const TILE_SPACING = 64;
/** 發牌逐張延遲 (秒) */
const DEAL_DELAY_SEC = 0.12;
/** 發牌飛入動畫時長 (秒) */
const DEAL_ANIM_DURATION = 0.18;

/**
 * HandView — 玩家手牌顯示元件
 *
 * 掛在場景中代表「本機玩家手牌容器」的 Node 上。
 *
 * 功能：
 *   1. dealTiles(tileIds)     — 逐張（依發牌順序）飛入顯示，不排序
 *   2. sortAndRedraw(sorted)  — 排序後清空重繪
 *   3. setOpponentCount(n)    — 對手模式，顯示 n 張牌背
 *   4. requestDeal()          — 向伺服器發送 deal_tiles 指令
 *
 * 事件訂閱（isSelf=true 時）：
 *   - ws_deal_tiles_res  → 收到手牌 tileId list，開始逐張動畫
 *   - ws_sort_hand_res   → 收到排序後 tileId list，重繪
 */
@ccclass('HandView')
export class HandView extends Component {

    /** true = 本機玩家（顯示牌正面），false = 對手（顯示牌背） */
    @property({ tooltip: '是否為本機玩家（顯示正面）' })
    public isSelf: boolean = true;

    /** 房間 ID（發送 WebSocket 時帶入） */
    @property({ tooltip: '房間 ID，留空使用 default_room' })
    public roomId: string = 'default_room';

    // ---- 內部狀態 ----
    private _tileIds: number[] = [];
    private _tileNodes: Node[] = [];
    private _isDealing: boolean = false;
    private _onTileClickCallback: ((tileId: number, node: Node) => void) | null = null;

    // ============================================================
    // 生命週期
    // ============================================================

    protected onLoad(): void {
        if (this.isSelf) {
            // 收到伺服器發牌結果
            EventMgr.on('ws_deal_tiles_res', this._onDealTilesRes, this);
            // 收到排序結果
            EventMgr.on('ws_sort_hand_res', this._onSortHandRes, this);
        }
    }

    protected onDestroy(): void {
        EventMgr.off('ws_deal_tiles_res', this._onDealTilesRes, this);
        EventMgr.off('ws_sort_hand_res', this._onSortHandRes, this);
    }

    // ============================================================
    // 公開 API
    // ============================================================

    /** 向伺服器請求發牌 */
    public requestDeal(): void {
        NetworkMgr.instance.send('deal_tiles', { roomId: this.roomId });
    }

    /**
     * 逐張顯示手牌（依摸牌順序，不排序）
     * 通常由 ws_deal_tiles_res 事件驅動，也可手動呼叫測試
     */
    public dealTiles(tileIds: number[]): void {
        if (this._isDealing) return;
        this._isDealing = true;
        this.clearHand();
        this._tileIds = [...tileIds];

        for (let i = 0; i < tileIds.length; i++) {
            // 使用 Cocos Component.scheduleOnce（this 是 Component）
            const capturedI = i;
            const capturedId = tileIds[i];
            const delay = capturedI * DEAL_DELAY_SEC;

            this.scheduleOnce(() => {
                this._addTileAnimated(capturedId, capturedI, tileIds.length);
                if (capturedI === tileIds.length - 1) {
                    this._isDealing = false;
                    // 通知伺服器執行排序
                    this._requestSort();
                }
            }, delay);
        }
    }

    /**
     * 清空並依排序後的 tileIds 重繪（無動畫）
     */
    public sortAndRedraw(sortedIds: number[]): void {
        this.clearHand();
        this._tileIds = [...sortedIds];
        for (let i = 0; i < sortedIds.length; i++) {
            this._addTileImmediate(sortedIds[i], i, sortedIds.length);
        }
    }

    /**
     * 對手模式：只顯示 n 張牌背
     */
    public setOpponentCount(count: number): void {
        this.clearHand();
        for (let i = 0; i < count; i++) {
            this._addOpponentTile(i, count);
        }
    }

    /** 清空所有手牌節點 */
    public clearHand(): void {
        for (const n of this._tileNodes) {
            n.removeFromParent();
            n.destroy();
        }
        this._tileNodes = [];
        this._tileIds = [];
    }

    /** 設定點牌回呼（供 GameView 等外層接管點牌事件） */
    public setOnTileClick(cb: (tileId: number, node: Node) => void): void {
        this._onTileClickCallback = cb;
    }

    /** 目前手牌 ID 列表 */
    public get tileIds(): number[] { return [...this._tileIds]; }

    // ============================================================
    // 私有輔助
    // ============================================================

    /** 計算第 index 張牌的 X 座標（讓整排置中） */
    private _calcX(index: number, total: number): number {
        const totalWidth = (total - 1) * TILE_SPACING;
        return index * TILE_SPACING - totalWidth / 2;
    }

    /** 加入一張牌（有飛入動畫） */
    private _addTileAnimated(tileId: number, index: number, total: number): void {
        const node = TileRenderer.createStandingTile(tileId);
        const targetX = this._calcX(index, total);

        // 直接設定到目標位置（暫不使用 tween 動畫以避開 API 相容性問題）
        node.setPosition(targetX, 0, 0);
        node.parent = this.node;
        this._tileNodes.push(node);

        if (this.isSelf) {
            node.on(Node.EventType.TOUCH_END, () => {
                this._onTileClickCallback?.(tileId, node);
            });
        }
    }

    /** 加入一張牌（無動畫，供排序後重繪用） */
    private _addTileImmediate(tileId: number, index: number, total: number): void {
        const node = TileRenderer.createStandingTile(tileId);
        node.setPosition(this._calcX(index, total), 0, 0);
        node.parent = this.node;
        this._tileNodes.push(node);

        node.on(Node.EventType.TOUCH_END, () => {
            this._onTileClickCallback?.(tileId, node);
        });
    }

    /** 加入一張對手牌背 */
    private _addOpponentTile(index: number, total: number): void {
        const node = TileRenderer.createOpponentTile();
        node.setPosition(this._calcX(index, total), 0, 0);
        node.parent = this.node;
        this._tileNodes.push(node);
    }

    /** 所有牌發完後，向伺服器請求排序 */
    private _requestSort(): void {
        NetworkMgr.instance.send('sort_hand', { roomId: this.roomId });
    }

    // ============================================================
    // WebSocket 事件處理
    // ============================================================

    private _onDealTilesRes(data: any): void {
        const tileIds: number[] = data?.tiles ?? [];
        if (tileIds.length === 0) {
            console.warn('[HandView] deal_tiles_res: 收到空手牌');
            return;
        }
        console.log(`[HandView] 收到發牌 ${tileIds.length} 張，開始逐張動畫`);
        this.dealTiles(tileIds);
    }

    private _onSortHandRes(data: any): void {
        const sortedIds: number[] = data?.tiles ?? [];
        if (sortedIds.length === 0) return;
        console.log('[HandView] 排序完成，重繪手牌');
        this.sortAndRedraw(sortedIds);
    }
}
