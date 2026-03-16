import {
    _decorator, Component, Node, Canvas, Camera, UITransform, Widget,
    Label, Graphics, Color, Size, Vec3, view, Sprite, SpriteFrame,
    director, game, Game, Layout
} from 'cc';
import { GameView } from './GameView';
import { HandView } from './HandView';
import { PlayerInfoView } from './PlayerInfoView';
import { NetworkMgr } from '../../Core/NetworkMgr';
import { ErrorReporter } from '../../Core/ErrorReporter';
import { EventMgr } from '../../Events/EventMgr';

const { ccclass, property } = _decorator;

/** 設計解析度 */
const DESIGN_WIDTH = 1280;
const DESIGN_HEIGHT = 720;

/** 牌桌顏色 */
const TABLE_BG_COLOR = new Color(7, 82, 45, 255);        // 深綠色桌面
const TABLE_BORDER_COLOR = new Color(90, 60, 30, 255);    // 木質邊框
const TABLE_CENTER_COLOR = new Color(10, 95, 52, 255);    // 中心區域稍亮

/** 玩家區域配置 (0=自己/下方, 1=右方, 2=對面/上方, 3=左方) */
const PLAYER_CONFIGS = [
    { name: '自己', handY: -280, discardY: -120, infoX: 500, infoY: -300, rotation: 0 },
    { name: '右家', handY: 0, discardY: 0, infoX: 540, infoY: 200, rotation: 0 },
    { name: '對家', handY: 280, discardY: 120, infoX: -500, infoY: 300, rotation: 0 },
    { name: '左家', handY: 0, discardY: 0, infoX: -540, infoY: -200, rotation: 0 },
];

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

        console.log('[GameSceneSetup] 牌桌 UI 建構完成');
    }

    // ================================================================
    // 牌桌背景
    // ================================================================

    private createTableBackground(parent: Node): Node {
        const bgNode = new Node('TableBackground');
        const transform = bgNode.addComponent(UITransform);
        transform.setContentSize(new Size(DESIGN_WIDTH, DESIGN_HEIGHT));

        // 外層邊框 (木質色)
        const outerG = bgNode.addComponent(Graphics);
        outerG.fillColor = TABLE_BORDER_COLOR;
        outerG.roundRect(-DESIGN_WIDTH / 2, -DESIGN_HEIGHT / 2, DESIGN_WIDTH, DESIGN_HEIGHT, 16);
        outerG.fill();

        // 內層桌面 (深綠)
        const innerNode = new Node('TableInner');
        const innerT = innerNode.addComponent(UITransform);
        const innerW = DESIGN_WIDTH - 20;
        const innerH = DESIGN_HEIGHT - 20;
        innerT.setContentSize(new Size(innerW, innerH));
        const innerG = innerNode.addComponent(Graphics);
        innerG.fillColor = TABLE_BG_COLOR;
        innerG.roundRect(-innerW / 2, -innerH / 2, innerW, innerH, 12);
        innerG.fill();
        // 中心裝飾線 (菱形框) - 同一 Graphics 繪製
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
        bgNode.setSiblingIndex(0); // 確保在最底層
        return bgNode;
    }

    // ================================================================
    // 中央資訊面板
    // ================================================================

    private createCenterInfoPanel(parent: Node): Node {
        const panel = new Node('CenterInfoPanel');
        const pTransform = panel.addComponent(UITransform);
        pTransform.setContentSize(new Size(160, 160));

        // 背景子節點 (半透明深色圓角矩形 + 金色邊框 + 分隔線)
        const bgNode = new Node('CenterBg');
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
        const windTitleNode = new Node('WindTitle');
        const windTitleLabel = windTitleNode.addComponent(Label);
        windTitleLabel.string = '圈風';
        windTitleLabel.fontSize = 16;
        windTitleLabel.color = new Color(200, 170, 80, 255);
        windTitleNode.setPosition(0, 55, 0);
        panel.addChild(windTitleNode);

        // 圈風值 (大字)
        const windNode = new Node('WindLabel');
        const windLabel = windNode.addComponent(Label);
        windLabel.string = '東';
        windLabel.fontSize = 42;
        windLabel.color = Color.WHITE;
        windLabel.isBold = true;
        windNode.setPosition(0, 20, 0);
        panel.addChild(windNode);

        // 剩餘牌數
        const remainNode = new Node('RemainLabel');
        const remainLabel = remainNode.addComponent(Label);
        remainLabel.string = '剩餘: 144';
        remainLabel.fontSize = 18;
        remainLabel.color = new Color(180, 220, 180, 255);
        remainNode.setPosition(0, -40, 0);
        panel.addChild(remainNode);

        // 局數提示
        const roundNode = new Node('RoundLabel');
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
        // 棄牌區圍繞中央，形成一個方形區域
        const discardConfigs = [
            { x: 0, y: -100, w: 400, h: 120 },    // 0: 自己 (下方)
            { x: 260, y: 0, w: 120, h: 300 },      // 1: 右方
            { x: 0, y: 100, w: 400, h: 120 },       // 2: 對面 (上方)
            { x: -260, y: 0, w: 120, h: 300 },     // 3: 左方
        ];

        for (let i = 0; i < 4; i++) {
            const cfg = discardConfigs[i];
            const node = new Node(`DiscardArea_${i}`);
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

        // 手牌容器位置
        const handConfigs = [
            { x: 0, y: -290, w: 1100, h: 110 },    // 0: 自己 (下方，最大)
            { x: 520, y: 30, w: 60, h: 500 },       // 1: 右方 (直排)
            { x: 0, y: 290, w: 900, h: 70 },        // 2: 對面 (上方，稍小)
            { x: -520, y: 30, w: 60, h: 500 },      // 3: 左方 (直排)
        ];

        for (let i = 0; i < 4; i++) {
            const cfg = handConfigs[i];
            const node = new Node(`HandArea_${i}`);
            const transform = node.addComponent(UITransform);
            transform.setContentSize(new Size(cfg.w, cfg.h));
            node.setPosition(cfg.x, cfg.y, 0);

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
            { x: 480, y: -290, seat: '東' },     // 0: 自己 (右下角)
            { x: 520, y: 280, seat: '南' },      // 1: 右方 (右上角)
            { x: -480, y: 290, seat: '西' },     // 2: 對面 (左上角)
            { x: -520, y: -280, seat: '北' },    // 3: 左方 (左下角)
        ];

        for (let i = 0; i < 4; i++) {
            const cfg = infoConfigs[i];
            const node = new Node(`PlayerInfo_${i}`);
            const transform = node.addComponent(UITransform);
            transform.setContentSize(new Size(140, 80));
            node.setPosition(cfg.x, cfg.y, 0);

            // 背景
            const g = node.addComponent(Graphics);
            g.fillColor = new Color(0, 0, 0, 120);
            g.roundRect(-70, -40, 140, 80, 8);
            g.fill();
            if (i === 0) {
                // 自己的面板用金色邊框高亮
                g.strokeColor = new Color(200, 170, 80, 200);
                g.lineWidth = 2;
                g.roundRect(-70, -40, 140, 80, 8);
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
