import { _decorator, Component, Node, Label, Color, UITransform, Graphics, Size } from 'cc';
import { EventMgr } from '../../Events/EventMgr';

const { ccclass, property } = _decorator;

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
     * @param seatIndex 座位 (0=自己, 1=右, 2=對面, 3=左)
     * @param seatWind 門風文字 (東/南/西/北)
     * @param isSelf 是否為本機玩家
     */
    public initUI(seatIndex: number, seatWind: string, isSelf: boolean): void {
        this._seatIndex = seatIndex;
        this._isSelf = isSelf;

        // 門風標籤 (左上角圓形徽章)
        const windBadge = new Node('WindBadge');
        const windBadgeT = windBadge.addComponent(UITransform);
        windBadgeT.setContentSize(new Size(28, 28));
        windBadge.setPosition(-48, 18, 0);

        const windG = windBadge.addComponent(Graphics);
        windG.fillColor = this.getWindColor(seatWind);
        windG.circle(0, 0, 14);
        windG.fill();

        const windLabelNode = new Node('WindText');
        this._seatWindLabel = windLabelNode.addComponent(Label);
        this._seatWindLabel.string = seatWind;
        this._seatWindLabel.fontSize = 16;
        this._seatWindLabel.color = Color.WHITE;
        this._seatWindLabel.isBold = true;
        windBadge.addChild(windLabelNode);

        this.node.addChild(windBadge);

        // 玩家名稱
        const nameNode = new Node('NameLabel');
        this._nameLabel = nameNode.addComponent(Label);
        this._nameLabel.string = isSelf ? '我' : `玩家${seatIndex + 1}`;
        this._nameLabel.fontSize = 18;
        this._nameLabel.color = isSelf ? new Color(255, 220, 100, 255) : Color.WHITE;
        this._nameLabel.isBold = isSelf;
        nameNode.setPosition(12, 18, 0);
        this.node.addChild(nameNode);

        // 分數
        const scoreNode = new Node('ScoreLabel');
        this._scoreLabel = scoreNode.addComponent(Label);
        this._scoreLabel.string = '0 分';
        this._scoreLabel.fontSize = 20;
        this._scoreLabel.color = new Color(100, 255, 100, 255);
        this._scoreLabel.isBold = true;
        scoreNode.setPosition(0, -12, 0);
        this.node.addChild(scoreNode);

        // 狀態指示燈 (小圓點)
        this._statusIndicator = new Node('StatusDot');
        const dotT = this._statusIndicator.addComponent(UITransform);
        dotT.setContentSize(new Size(10, 10));
        const dotG = this._statusIndicator.addComponent(Graphics);
        dotG.fillColor = new Color(100, 100, 100, 200); // 灰色=等待中
        dotG.circle(0, 0, 5);
        dotG.fill();
        this._statusIndicator.setPosition(55, 18, 0);
        this.node.addChild(this._statusIndicator);

        // 監聽遊戲狀態變更
        EventMgr.on('game_state_changed', this.onGameStateChanged, this);
    }

    protected onDestroy(): void {
        EventMgr.off('game_state_changed', this.onGameStateChanged, this);
    }

    /**
     * 更新玩家資訊
     */
    public updateInfo(name: string, score: number, isCurrentTurn: boolean): void {
        if (this._nameLabel && name) {
            this._nameLabel.string = name;
        }
        if (this._scoreLabel) {
            this._scoreLabel.string = `${score} 分`;
            // 正分綠色，負分紅色
            this._scoreLabel.color = score >= 0
                ? new Color(100, 255, 100, 255)
                : new Color(255, 100, 100, 255);
        }
        this.setTurnIndicator(isCurrentTurn);
    }

    /**
     * 設定當前出牌指示
     */
    public setTurnIndicator(isCurrentTurn: boolean): void {
        if (!this._statusIndicator) return;

        const dotG = this._statusIndicator.getComponent(Graphics);
        if (!dotG) return;

        dotG.clear();
        if (isCurrentTurn) {
            dotG.fillColor = new Color(50, 255, 50, 255); // 綠色 = 輪到此玩家
            dotG.circle(0, 0, 6);
            dotG.fill();
        } else {
            dotG.fillColor = new Color(100, 100, 100, 200); // 灰色 = 等待
            dotG.circle(0, 0, 5);
            dotG.fill();
        }
    }

    /**
     * 更新門風
     */
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
            case '東': return new Color(200, 50, 50, 255);   // 紅
            case '南': return new Color(50, 150, 50, 255);   // 綠
            case '西': return new Color(50, 100, 200, 255);  // 藍
            case '北': return new Color(150, 100, 200, 255); // 紫
            default:   return new Color(150, 150, 150, 255); // 灰
        }
    }
}
