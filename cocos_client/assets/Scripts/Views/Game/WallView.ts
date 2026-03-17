import {
    _decorator, Component, Node, Graphics, Color, UITransform, Size,
    Layers, Vec3, tween
} from 'cc';

const { ccclass } = _decorator;

const UI_2D = Layers.Enum.UI_2D;

function uiNode(name: string): Node {
    const n = new Node(name);
    n.layer = UI_2D;
    return n;
}

// ── 牌墩尺寸 (俯視圖) ──────────────────────────────
const STACK_W = 12;       // 每墩寬度 (px)
const STACK_H = 8;        // 頂牌高度 (px)
const DEPTH_H = 3;        // 底牌露出厚度 (px)，表現「一墩兩張上下疊」
const GAP = 1;            // 墩與墩之間的間距

// ── 牌面色彩 ────────────────────────────────────────
const TILE_BACK_COLOR = new Color(45, 110, 55, 255);       // 牌背 (深綠)
const TILE_EDGE_COLOR = new Color(210, 200, 180, 255);     // 底牌露出邊 (象牙色)
const TILE_HIGHLIGHT  = new Color(70, 140, 80, 150);       // 頂牌亮邊

// ── 動畫設定 ────────────────────────────────────────
const INITIAL_DELAY   = 0.3;     // 動畫開始前等待
const DELAY_PER_STACK = 0.022;   // 每墩延遲 (秒)
const ANIM_DURATION   = 0.12;    // 每墩出現動畫時長

// ── 四面牆旋轉角度 (讓 depth 朝外) ──────────────────
//   南(0°): depth 朝下  東(90°): depth 朝右
//   北(180°): depth 朝上  西(-90°): depth 朝左
const WALL_ROTATIONS = [0, 90, 180, -90];

/**
 * WallView: 麻將牌牆
 *
 * 在牌桌中央四周顯示砌好的牌牆：
 *   - 13 張麻將 → 136 張 = 17 墩 × 2 張 × 4 面
 *   - 16 張麻將 → 144 張 = 18 墩 × 2 張 × 4 面
 *
 * 每墩為兩張牌上下疊起 (面朝下)，以 Graphics 繪製。
 * 四面牆圍成正方形，配合順時針動畫依序出現。
 */
@ccclass('WallView')
export class WallView extends Component {

    private _wallNodes: Node[] = [];
    private _allStacks: Node[] = [];

    /**
     * 建立牌牆 + 砌牌動畫
     * @param stacksPerWall 每面牆的墩數 (17 = 13 張, 18 = 16 張)
     */
    public buildWall(stacksPerWall: number = 18): void {
        this.clearWall();

        const totalW = stacksPerWall * (STACK_W + GAP) - GAP;
        // wallDist = totalW / 2 → 四面牆恰好圍成正方形
        const wallDist = totalW / 2;

        for (let w = 0; w < 4; w++) {
            const wallNode = uiNode(`Wall_${w}`);

            // 定位四面牆中心
            switch (w) {
                case 0: wallNode.setPosition(0, -wallDist, 0); break;    // 南
                case 1: wallNode.setPosition(wallDist, 0, 0); break;     // 東
                case 2: wallNode.setPosition(0, wallDist, 0); break;     // 北
                case 3: wallNode.setPosition(-wallDist, 0, 0); break;    // 西
            }

            // 旋轉使 depth 朝外
            const rot = WALL_ROTATIONS[w];
            if (rot !== 0) {
                wallNode.setRotationFromEuler(0, 0, rot);
            }

            // 生成每墩
            for (let s = 0; s < stacksPerWall; s++) {
                const stackNode = this.createStackNode();
                const x = s * (STACK_W + GAP) - totalW / 2 + STACK_W / 2;
                stackNode.setPosition(x, 0, 0);
                stackNode.setScale(0, 0, 1); // 初始隱藏
                wallNode.addChild(stackNode);
                this._allStacks.push(stackNode);
            }

            this.node.addChild(wallNode);
            this._wallNodes.push(wallNode);
        }

        // ── 順時針砌牌動畫：南 → 東 → 北 → 西 ──
        let delay = INITIAL_DELAY;
        for (const stack of this._allStacks) {
            tween(stack)
                .delay(delay)
                .to(ANIM_DURATION, { scale: new Vec3(1, 1, 1) }, { easing: 'backOut' })
                .start();
            delay += DELAY_PER_STACK;
        }

        const totalStacks = stacksPerWall * 4;
        console.log(`[WallView] 砌牌動畫開始: ${stacksPerWall} 墩/面, 共 ${totalStacks} 墩 (${totalStacks * 2} 張)`);
    }

    /**
     * 清除牌牆
     */
    public clearWall(): void {
        this._allStacks = [];
        for (const wn of this._wallNodes) {
            wn.removeFromParent();
            wn.destroy();
        }
        this._wallNodes = [];
    }

    // ── 建立單個牌墩 (2 張牌上下疊) ────────────────

    private createStackNode(): Node {
        const node = uiNode('stack');
        const t = node.addComponent(UITransform);
        t.setContentSize(new Size(STACK_W, STACK_H + DEPTH_H));

        const g = node.addComponent(Graphics);

        const hw = STACK_W / 2;
        const hh = (STACK_H + DEPTH_H) / 2;

        // 1) 底牌露出的邊 (象牙色，模擬牌的側面)
        g.fillColor = TILE_EDGE_COLOR;
        g.roundRect(-hw, -hh, STACK_W, DEPTH_H, 1);
        g.fill();

        // 2) 頂牌背面 (深綠)
        g.fillColor = TILE_BACK_COLOR;
        g.roundRect(-hw, -hh + DEPTH_H, STACK_W, STACK_H, 1);
        g.fill();

        // 3) 頂牌亮邊 (微光澤)
        g.strokeColor = TILE_HIGHLIGHT;
        g.lineWidth = 0.5;
        g.roundRect(-hw + 1, -hh + DEPTH_H + 1, STACK_W - 2, STACK_H - 2, 1);
        g.stroke();

        return node;
    }
}
