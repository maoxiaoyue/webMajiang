import {
    _decorator, Component, Node, Canvas, Camera, UITransform, Widget,
    Label, Graphics, Color, Size, Vec3, view, Sprite, SpriteFrame,
    director, game, Game, Layout, Layers
} from 'cc';
import { GameView } from './GameView';
import { HandView } from './HandView';
import { PlayerInfoView } from './PlayerInfoView';
import { NetworkMgr } from '../../Core/NetworkMgr';
import { ErrorReporter } from '../../Core/ErrorReporter';
import { EventMgr } from '../../Events/EventMgr';

const { ccclass, property } = _decorator;

/** UI_2D 圖層 (1 << 25)，必須與 Camera.visibility 一致才能被渲染 */
const UI_2D = Layers.Enum.UI_2D;

/** 設計解析度 */
const DESIGN_WIDTH = 1280;
const DESIGN_HEIGHT = 720;

/** 邊緣留白 */
const MARGIN = 30;

/** 牌桌顏色 */
const TABLE_BG_COLOR = new Color(7, 82, 45, 255);        // 深綠色桌面
const TABLE_BORDER_COLOR = new Color(90, 60, 30, 255);    // 木質邊框

/**
 * 建立一個已設定好 UI_2D layer 的節點
 */
function uiNode(name: string): Node {
    const n = new Node(name);
    n.layer = UI_2D;
    return n;
}

/**
 * GameSceneSetup: 遊戲場景啟動器
 *
 * 負責在 main.scene 載入後，以程式碼建構完整的麻將桌面 UI：
 *   - 綠色牌桌背景
 *   - 中央資訊面板 (圈風、剩餘牌數)
 *   - 四位玩家資訊面板 (姓名、分數、門風)
 *   - 四個手牌容器
 *   - 四個棄牌區容器
 *   - 掛載 GameView 與 HandView 元件
 */
@ccclass('GameSceneSetup')
export class GameSceneSetup extends Component {

    private _gameView: GameView | null = null;

    protected onLoad(): void {
        // 確保自身節點也在 UI_2D layer
        this.node.layer = UI_2D;
        ErrorReporter.init();
        this.buildGameTable();
        this.connectToServer();
    }

    // ================================================================
    // 建構牌桌
    // ================================================================

    private buildGameTable(): void {
        const rootNode = this.node;

        // 1. 牌桌背景
        this.createTableBackground(rootNode);

        // 2. 中央資訊面板
        const centerPanel = this.createCenterInfoPanel(rootNode);

        // 3. 棄牌區 (4 個)
        const discardNodes = this.createDiscardAreas(rootNode);

        // 4. 手牌容器 (4 個)
        const handNodes = this.createHandAreas(rootNode);

        // 5. 玩家資訊面板 (4 個)
        const playerInfoNodes = this.createPlayerInfoPanels(rootNode);

        // 6. 掛載 GameView 並綁定節點
        this._gameView = rootNode.addComponent(GameView);
        this._gameView.centerInfoPanel = centerPanel;
        this._gameView.playerHandNodes = handNodes;
        this._gameView.playerDiscardNodes = discardNodes;

        // 綁定中央面板上的 Label
        const remainLabel = centerPanel.getChildByName('RemainLabel');
        const windLabel = centerPanel.getChildByName('WindLabel');
        if (remainLabel) {
            this._gameView.remainingTilesLabel = remainLabel.getComponent(Label)!;
        }
        if (windLabel) {
            this._gameView.currentWindLabel = windLabel.getComponent(Label)!;
        }

        // 7. 為自己的手牌區掛 HandView
        const selfHandView = handNodes[0].addComponent(HandView);
        selfHandView.isSelf = true;

        // 8. 顯示 demo 手牌（測試用，連線後會被真實資料取代）
        this.showDemoHand(selfHandView, handNodes);

        console.log('[GameSceneSetup] 牌桌 UI 建構完成');
    }

    // ================================================================
    // 牌桌背景
    // ================================================================

    private createTableBackground(parent: Node): Node {
        const bgNode = uiNode('TableBackground');
        const transform = bgNode.addComponent(UITransform);
        transform.setContentSize(new Size(DESIGN_WIDTH, DESIGN_HEIGHT));

        // 外層邊框 (木質色)
        const outerG = bgNode.addComponent(Graphics);
        outerG.fillColor = TABLE_BORDER_COLOR;
        outerG.roundRect(-DESIGN_WIDTH / 2, -DESIGN_HEIGHT / 2, DESIGN_WIDTH, DESIGN_HEIGHT, 16);
        outerG.fill();

        // 內層桌面 (深綠)
        const innerNode = uiNode('TableInner');
        const innerT = innerNode.addComponent(UITransform);
        const innerW = DESIGN_WIDTH - 20;
        const innerH = DESIGN_HEIGHT - 20;
        innerT.setContentSize(new Size(innerW, innerH));
        const innerG = innerNode.addComponent(Graphics);
        innerG.fillColor = TABLE_BG_COLOR;
        innerG.roundRect(-innerW / 2, -innerH / 2, innerW, innerH, 12);
        innerG.fill();
        // 中心裝飾線 (菱形框)
        innerG.strokeColor = new Color(255, 255, 255, 30);
        innerG.lineWidth = 2;
        const dSize = 200;
        innerG.moveTo(0, dSize);
        innerG.lineTo(dSize, 0);
        innerG.lineTo(0, -dSize);
        innerG.lineTo(-dSize, 0);
        innerG.close();
        innerG.stroke();

        bgNode.addChild(innerNode);
        parent.addChild(bgNode);
        bgNode.setSiblingIndex(0);
        return bgNode;
    }

    // ================================================================
    // 中央資訊面板
    // ================================================================

    private createCenterInfoPanel(parent: Node): Node {
        const panel = uiNode('CenterInfoPanel');
        const pTransform = panel.addComponent(UITransform);
        pTransform.setContentSize(new Size(160, 160));

        // 背景子節點
        const bgNode = uiNode('CenterBg');
        const bgT = bgNode.addComponent(UITransform);
        bgT.setContentSize(new Size(160, 160));
        const g = bgNode.addComponent(Graphics);
        g.fillColor = new Color(0, 0, 0, 140);
        g.roundRect(-80, -80, 160, 160, 16);
        g.fill();
        g.strokeColor = new Color(200, 170, 80, 200);
        g.lineWidth = 2;
        g.roundRect(-80, -80, 160, 160, 16);
        g.stroke();
        // 分隔線
        g.strokeColor = new Color(200, 170, 80, 100);
        g.lineWidth = 1;
        g.moveTo(-50, -10);
        g.lineTo(50, -10);
        g.stroke();
        panel.addChild(bgNode);

        // 圈風標題
        const windTitleNode = uiNode('WindTitle');
        const windTitleLabel = windTitleNode.addComponent(Label);
        windTitleLabel.string = '圈風';
        windTitleLabel.fontSize = 16;
        windTitleLabel.color = new Color(200, 170, 80, 255);
        windTitleNode.setPosition(0, 55, 0);
        panel.addChild(windTitleNode);

        // 圈風值 (大字)
        const windNode = uiNode('WindLabel');
        const windLabel = windNode.addComponent(Label);
        windLabel.string = '東';
        windLabel.fontSize = 42;
        windLabel.color = Color.WHITE;
        windLabel.isBold = true;
        windNode.setPosition(0, 20, 0);
        panel.addChild(windNode);

        // 剩餘牌數
        const remainNode = uiNode('RemainLabel');
        const remainLabel = remainNode.addComponent(Label);
        remainLabel.string = '剩餘: 144';
        remainLabel.fontSize = 18;
        remainLabel.color = new Color(180, 220, 180, 255);
        remainNode.setPosition(0, -40, 0);
        panel.addChild(remainNode);

        // 局數提示
        const roundNode = uiNode('RoundLabel');
        const roundLabel = roundNode.addComponent(Label);
        roundLabel.string = '第 1 局';
        roundLabel.fontSize = 14;
        roundLabel.color = new Color(150, 150, 150, 255);
        roundNode.setPosition(0, -65, 0);
        panel.addChild(roundNode);

        parent.addChild(panel);
        return panel;
    }

    // ================================================================
    // 棄牌區 (4 玩家)
    // ================================================================

    private createDiscardAreas(parent: Node): Node[] {
        const areas: Node[] = [];
        const discardConfigs = [
            { x: 0, y: -110, w: 350, h: 90 },     // 0: 自己 (下方)
            { x: 200, y: 0, w: 90, h: 250 },      // 1: 右方
            { x: 0, y: 110, w: 350, h: 90 },      // 2: 對面 (上方)
            { x: -200, y: 0, w: 90, h: 250 },     // 3: 左方
        ];

        for (let i = 0; i < 4; i++) {
            const cfg = discardConfigs[i];
            const node = uiNode(`DiscardArea_${i}`);
            const transform = node.addComponent(UITransform);
            transform.setContentSize(new Size(cfg.w, cfg.h));
            node.setPosition(cfg.x, cfg.y, 0);

            // 半透明背景標示棄牌區
            const g = node.addComponent(Graphics);
            g.fillColor = new Color(0, 0, 0, 25);
            g.roundRect(-cfg.w / 2, -cfg.h / 2, cfg.w, cfg.h, 4);
            g.fill();

            parent.addChild(node);
            areas.push(node);
        }
        return areas;
    }

    // ================================================================
    // 手牌區 (4 玩家)
    // ================================================================

    private createHandAreas(parent: Node): Node[] {
        const areas: Node[] = [];
        // 手牌區: 下(自己)、右(下家)、上(對家)、左(上家)
        // 右方和左方的手牌容器要旋轉 90°，牌變成直的
        const handConfigs = [
            { x: 0, y: -210, w: 700, h: 60, rot: 0 },      // 0: 自己 (下方，橫排)
            { x: 420, y: 0, w: 550, h: 50, rot: 90 },      // 1: 右方 (旋轉90°，直排)
            { x: 0, y: 210, w: 550, h: 50, rot: 180 },     // 2: 對面 (上方，反轉)
            { x: -420, y: 0, w: 550, h: 50, rot: -90 },    // 3: 左方 (旋轉-90°，直排)
        ];

        for (let i = 0; i < 4; i++) {
            const cfg = handConfigs[i];
            const node = uiNode(`HandArea_${i}`);
            const transform = node.addComponent(UITransform);
            transform.setContentSize(new Size(cfg.w, cfg.h));
            node.setPosition(cfg.x, cfg.y, 0);

            // 旋轉手牌容器
            if (cfg.rot !== 0) {
                node.setRotationFromEuler(0, 0, cfg.rot);
            }

            parent.addChild(node);
            areas.push(node);
        }
        return areas;
    }

    // ================================================================
    // 玩家資訊面板
    // ================================================================

    private createPlayerInfoPanels(parent: Node): Node[] {
        const panels: Node[] = [];
        const infoConfigs = [
            { x: 400, y: -250, seat: '東' },     // 0: 自己 (右下角)
            { x: 520, y: 180, seat: '南' },      // 1: 右方 (右側)
            { x: -400, y: 255, seat: '西' },     // 2: 對面 (左上角)
            { x: -520, y: -180, seat: '北' },    // 3: 左方 (左側)
        ];

        const PW = 100, PH = 36;
        for (let i = 0; i < 4; i++) {
            const cfg = infoConfigs[i];
            const node = uiNode(`PlayerInfo_${i}`);
            const transform = node.addComponent(UITransform);
            transform.setContentSize(new Size(PW, PH));
            node.setPosition(cfg.x, cfg.y, 0);

            // 背景
            const g = node.addComponent(Graphics);
            g.fillColor = new Color(0, 0, 0, 120);
            g.roundRect(-PW / 2, -PH / 2, PW, PH, 6);
            g.fill();
            if (i === 0) {
                g.strokeColor = new Color(200, 170, 80, 200);
                g.lineWidth = 1;
                g.roundRect(-PW / 2, -PH / 2, PW, PH, 6);
                g.stroke();
            }

            // 掛載 PlayerInfoView 元件
            const infoView = node.addComponent(PlayerInfoView);
            infoView.initUI(i, cfg.seat, i === 0);

            parent.addChild(node);
            panels.push(node);
        }
        return panels;
    }

    // ================================================================
    // Demo 手牌（測試渲染）
    // ================================================================

    private showDemoHand(selfHandView: HandView, handNodes: Node[]): void {
        // 自己的手牌：16 張示範牌
        // ID: 1=一萬, 5=二萬, 9=三萬, 13=四萬, 17=五萬, 21=六萬, 25=七萬, 29=八萬, 33=九萬
        //     37=一筒, 41=二筒, 45=三筒, 49=四筒, 109=東, 113=南, 117=西
        const demoTiles = [1, 5, 9, 13, 17, 21, 25, 29, 33, 37, 41, 45, 49, 109, 113, 117];
        selfHandView.sortAndRedraw(demoTiles);

        // 對手手牌：顯示牌背（對手容器已旋轉，牌會自動跟著轉）
        for (let i = 1; i <= 3; i++) {
            const handView = handNodes[i].addComponent(HandView);
            handView.isSelf = false;
            // 對手牌稍小
            handNodes[i].setScale(0.6, 0.6, 1);
            handView.setOpponentCount(13);
        }
    }

    // ================================================================
    // 連線到遊戲伺服器
    // ================================================================

    private connectToServer(): void {
        const lobbyParams = (window as any).__LOBBY_PARAMS__;
        if (!lobbyParams) {
            console.log('[GameSceneSetup] 無 LobbyBridge 參數，等待手動連線');
            return;
        }

        const wsUrl = lobbyParams.wsUrl || 'wss://lobby.cxwoo.com/game-ws';
        console.log(`[GameSceneSetup] 自動連線 WebSocket: ${wsUrl}`);

        NetworkMgr.instance.connect(wsUrl);

        // 連線成功後自動加入房間
        EventMgr.once(NetworkMgr.EVENT_CONNECTED, () => {
            console.log('[GameSceneSetup] WebSocket 已連線，發送 join_room');
            NetworkMgr.instance.send('join_room', {
                gameId: lobbyParams.gameId,
                token: lobbyParams.token,
                seatId: lobbyParams.seatId,
            });
        });
    }
}
