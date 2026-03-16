import { _decorator, Component, Node, Label, Color, UITransform, Graphics, Size, Layers } from 'cc';
import { EventMgr } from '../../Events/EventMgr';

const { ccclass, property } = _decorator;

const UI_2D = Layers.Enum.UI_2D;

/** 建立 UI_2D layer 的節點 */
function uiNode(name: string): Node {
    const n = new Node(name);
    n.layer = UI_2D;
    return n;
}

/**
 * PlayerInfoView: 顯示單位玩家的資訊
 *   - 門風 (東/南/西/北)
 *   - 玩家名稱
 *   - 分數
 *   - 出牌中 / 等待 狀態指示
 */
@ccclass('PlayerInfoView')
export class PlayerInfoView extends Component {

    private _seatIndex: number = 0;
    private _seatWindLabel: Label | null = null;
    private _nameLabel: Label | null = null;
    private _scoreLabel: Label | null = null;
    private _statusIndicator: Node | null = null;
    private _isSelf: boolean = false;

    /**
     * 初始化玩家資訊 UI
     */
    public initUI(seatIndex: number, seatWind: string, isSelf: boolean): void {
        this._seatIndex = seatIndex;
        this._isSelf = isSelf;

        // 門風標籤 (小圓形徽章)
        const windBadge = uiNode('WindBadge');
        const windBadgeT = windBadge.addComponent(UITransform);
        windBadgeT.setContentSize(new Size(18, 18));
        windBadge.setPosition(-36, 0, 0);

        const windG = windBadge.addComponent(Graphics);
        windG.fillColor = this.getWindColor(seatWind);
        windG.circle(0, 0, 9);
        windG.fill();

        const windLabelNode = uiNode('WindText');
        this._seatWindLabel = windLabelNode.addComponent(Label);
        this._seatWindLabel.string = seatWind;
        this._seatWindLabel.fontSize = 11;
        this._seatWindLabel.color = Color.WHITE;
        this._seatWindLabel.isBold = true;
        windBadge.addChild(windLabelNode);

        this.node.addChild(windBadge);

        // 玩家名稱
        const nameNode = uiNode('NameLabel');
        this._nameLabel = nameNode.addComponent(Label);
        this._nameLabel.string = isSelf ? '我' : `玩家${seatIndex + 1}`;
        this._nameLabel.fontSize = 12;
        this._nameLabel.color = isSelf ? new Color(255, 220, 100, 255) : Color.WHITE;
        this._nameLabel.isBold = isSelf;
        nameNode.setPosition(-8, 8, 0);
        this.node.addChild(nameNode);

        // 分數
        const scoreNode = uiNode('ScoreLabel');
        this._scoreLabel = scoreNode.addComponent(Label);
        this._scoreLabel.string = '0 分';
        this._scoreLabel.fontSize = 12;
        this._scoreLabel.color = new Color(100, 255, 100, 255);
        this._scoreLabel.isBold = true;
        scoreNode.setPosition(-8, -8, 0);
        this.node.addChild(scoreNode);

        // 狀態指示燈 (小圓點)
        this._statusIndicator = uiNode('StatusDot');
        const dotT = this._statusIndicator.addComponent(UITransform);
        dotT.setContentSize(new Size(6, 6));
        const dotG = this._statusIndicator.addComponent(Graphics);
        dotG.fillColor = new Color(100, 100, 100, 200);
        dotG.circle(0, 0, 3);
        dotG.fill();
        this._statusIndicator.setPosition(40, 8, 0);
        this.node.addChild(this._statusIndicator);

        // 監聽遊戲狀態變更
        EventMgr.on('game_state_changed', this.onGameStateChanged, this);
    }

    protected onDestroy(): void {
        EventMgr.off('game_state_changed', this.onGameStateChanged, this);
    }

    public updateInfo(name: string, score: number, isCurrentTurn: boolean): void {
        if (this._nameLabel && name) {
            this._nameLabel.string = name;
        }
        if (this._scoreLabel) {
            this._scoreLabel.string = `${score} 分`;
            this._scoreLabel.color = score >= 0
                ? new Color(100, 255, 100, 255)
                : new Color(255, 100, 100, 255);
        }
        this.setTurnIndicator(isCurrentTurn);
    }

    public setTurnIndicator(isCurrentTurn: boolean): void {
        if (!this._statusIndicator) return;
        const dotG = this._statusIndicator.getComponent(Graphics);
        if (!dotG) return;

        dotG.clear();
        if (isCurrentTurn) {
            dotG.fillColor = new Color(50, 255, 50, 255);
            dotG.circle(0, 0, 6);
            dotG.fill();
        } else {
            dotG.fillColor = new Color(100, 100, 100, 200);
            dotG.circle(0, 0, 5);
            dotG.fill();
        }
    }

    public updateSeatWind(wind: string): void {
        if (this._seatWindLabel) {
            this._seatWindLabel.string = wind;
        }
    }

    private onGameStateChanged(modelInfo: any): void {
        if (!modelInfo || !modelInfo.players) return;
        const player = modelInfo.players.find((p: any) => p.seat === this._seatIndex);
        if (player) {
            const isCurrentTurn = modelInfo.currentTurnPlayerId === player.id;
            this.updateInfo(player.name || this._nameLabel?.string || '', player.score || 0, isCurrentTurn);
        }
    }

    private getWindColor(wind: string): Color {
        switch (wind) {
            case '東': return new Color(200, 50, 50, 255);
            case '南': return new Color(50, 150, 50, 255);
            case '西': return new Color(50, 100, 200, 255);
            case '北': return new Color(150, 100, 200, 255);
            default:   return new Color(150, 150, 150, 255);
        }
    }
}
