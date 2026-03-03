import { _decorator, Component, Node, Graphics, Label, Color, UITransform, Vec3 } from 'cc';
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

/** 副露與手牌的垂直間距 (px) */
const OUTDESK_Y_OFFSET = 100;
/** 副露牌之間的間距 (px) */
const MELD_SPACING = 55;
/** 每組副露之間的距離 (px) */
const MELD_GROUP_SPACING = 20;

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
    private _outdeskNodes: Node[] = []; // 副露與花牌節點
    private _isDealing: boolean = false;
    private _onTileClickCallback: ((tileId: number, node: Node) => void) | null = null;
    private _actionPanel: Node | null = null;

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
        for (const n of this._outdeskNodes) {
            n.removeFromParent();
            n.destroy();
        }
        this._tileNodes = [];
        this._outdeskNodes = [];
        this._tileIds = [];
    }

    /** 設定玩家副露 (碰/吃/槓) 與花牌 */
    public setOutdesk(melds: { type: number, tiles: number[] }[], flowers: number[]): void {
        // 先清空舊的副露節點
        for (const n of this._outdeskNodes) {
            n.removeFromParent();
            n.destroy();
        }
        this._outdeskNodes = [];

        // 計算起始位置 (最左側開始排)
        let currentX = -((16 * TILE_SPACING) / 2); // 預設以 16 張牌寬度作為基準左側
        const startY = OUTDESK_Y_OFFSET;

        // 1. 繪製花牌
        if (flowers && flowers.length > 0) {
            for (const fId of flowers) {
                // 花牌一律正面立在桌上，但為求美觀可以用 FaceUp 或者 FaceUpWithTop
                const node = TileRenderer.createStandingTile(fId);
                node.setPosition(currentX, startY, 0);
                node.parent = this.node;
                // 標示花牌半透明或稍微縮小以區別
                const uiTrans = node.getComponent(UITransform);
                if (uiTrans) {
                    node.scale = new Vec3(0.8, 0.8, 1);
                }
                this._outdeskNodes.push(node);
                currentX += (TILE_SPACING * 0.8) + 5;
            }
            currentX += MELD_GROUP_SPACING;
        }

        // 2. 繪製副露 (吃碰槓)
        if (melds && melds.length > 0) {
            for (const m of melds) {
                // type: 1(吃), 2(碰), 3(明槓), 4(暗槓), 5(加槓)
                const isHiddenKong = (m.type === 4);

                for (let i = 0; i < m.tiles.length; i++) {
                    const tId = m.tiles[i];
                    let node: Node;

                    if (isHiddenKong) {
                        // 暗槓不翻面，統一給對手背牌
                        node = TileRenderer.createOpponentTile();
                    } else {
                        // 明牌副露，平躺顯示，使用 FaceUp
                        node = TileRenderer.createTileNode(tId, 0); // 0 = FaceUp
                    }

                    node.setPosition(currentX, startY, 0);
                    node.parent = this.node;
                    this._outdeskNodes.push(node);
                    currentX += MELD_SPACING;

                    // 加槓 (第四張疊在第二張上面)
                    if (m.type === 5 && i === 3) {
                        const baseIndex = this._outdeskNodes.length - 3; // 取出第二張的位置
                        if (baseIndex >= 0) {
                            const baseNode = this._outdeskNodes[baseIndex];
                            node.setPosition(baseNode.position.x, startY + 20, 0); // Y 軸往上疊
                        }
                    }
                }
                currentX += MELD_GROUP_SPACING;
            }
        }
    }

    /** 設定點牌回呼（供 GameView 等外層接管點牌事件） */
    public setOnTileClick(cb: (tileId: number, node: Node) => void): void {
        this._onTileClickCallback = cb;
    }

    /** 目前手牌 ID 列表 */
    public get tileIds(): number[] { return [...this._tileIds]; }

    // ============================================================
    // 玩家宣告按鈕 (碰/吃/槓/胡/過)
    // ============================================================

    public showActionButtons(roomId: string, tileId: number): void {
        if (!this.isSelf) return;
        if (!this._actionPanel) {
            this._createActionPanel();
        }
        this._actionPanel!.active = true;

        // 定位在最新進牌（最右側）的正上方
        const rightmostX = this._calcX(this._tileIds.length, this._tileIds.length + 1);
        this._actionPanel!.setPosition(rightmostX, 150, 0);

        // 重新綁定事件以更新 tileId 與 roomId
        this._actionPanel!.children.forEach((btnNode: Node) => {
            btnNode.off(Node.EventType.TOUCH_END);
            const actionTypeStr = btnNode.name;
            let actionType = 6; // pass
            if (actionTypeStr === 'chow') actionType = 2;
            if (actionTypeStr === 'pong') actionType = 3;
            if (actionTypeStr === 'kong') actionType = 4;
            if (actionTypeStr === 'hu') actionType = 5;

            btnNode.on(Node.EventType.TOUCH_END, () => {
                console.log(`[HandView] 玩家宣告: ${actionTypeStr}, tileId: ${tileId}`);
                NetworkMgr.instance.send('player_action', {
                    roomId: roomId,
                    actionType: actionType,
                    tileId: tileId
                });
                this.hideActionButtons();
            });
        });
    }

    public hideActionButtons(): void {
        if (this._actionPanel) {
            this._actionPanel.active = false;
        }
    }

    private _createActionPanel(): void {
        this._actionPanel = new Node('ActionPanel');
        this._actionPanel.parent = this.node;

        const actions = [
            { name: 'chow', text: '吃', color: new Color(50, 150, 50, 240) },
            { name: 'pong', text: '碰', color: new Color(50, 100, 200, 240) },
            { name: 'kong', text: '槓', color: new Color(200, 150, 50, 240) },
            { name: 'hu', text: '胡', color: new Color(200, 50, 50, 240) },
            { name: 'pass', text: '過', color: new Color(100, 100, 100, 240) }
        ];

        const panelWidth = actions.length * 80;
        let startX = -panelWidth / 2 + 40;

        actions.forEach((act, index) => {
            const btnNode = new Node(act.name);
            btnNode.parent = this._actionPanel;
            btnNode.setPosition(startX + index * 80, 0, 0);

            // 新增 UITransform 以接收互動事件與設定大小
            const uiTrans = btnNode.addComponent(UITransform);
            uiTrans.setContentSize(70, 70);

            // 繪製圓角矩形背景
            const graphics = btnNode.addComponent(Graphics);
            graphics.fillColor = act.color;
            graphics.roundRect(-35, -35, 70, 70, 15);
            graphics.fill();

            // 加入文字標籤
            const labelNode = new Node('Label');
            labelNode.parent = btnNode;
            const label = labelNode.addComponent(Label);
            label.string = act.text;
            label.fontSize = 28;
            label.color = Color.WHITE;
            label.isBold = true;
        });
    }

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
