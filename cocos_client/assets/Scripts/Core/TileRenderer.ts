import { _decorator, Component, Node, Sprite, Label, Color, UITransform, SpriteFrame, resources, ImageAsset, Texture2D, Size, Graphics } from 'cc';
import { TileInfo, parseTileId, getTileImagePath, getTileTextDisplay, getTileName } from './TileConfig';

const { ccclass, property } = _decorator;

/** 牌面尺寸常數（對應 .pen 設計: 60x84） */
const TILE_WIDTH = 60;
const TILE_HEIGHT = 84;
const CONTENT_WIDTH = 50;
const CONTENT_HEIGHT = 70;
/** 側面尺寸 */
const SIDE_TOP_HEIGHT = 18;
const SIDE_RIGHT_WIDTH = 12;

/**
 * 牌的顯示模式
 */
export enum TileDisplayMode {
    /** 正面 (60x84) */
    FaceUp = 0,
    /** 背面 (60x84) */
    FaceDown = 1,
    /** 正面 + 上側面 (60x102)，立牌時用，有 3D 深度感 */
    FaceUpWithTop = 2,
    /** 背面 + 上側面 (60x102)，對手立牌 / 牌山 */
    FaceDownWithTop = 3,
    /** 正面 + 右側面 (72x84)，橫放的牌 */
    FaceUpWithRight = 4,
    /** 僅上側面 (60x18)，牌山堆疊時用 */
    SideTopOnly = 5,
}

/**
 * TileRenderer: 負責渲染單張麻將牌的元件
 *
 * 使用方式:
 *   const node = TileRenderer.createTileNode(tileId, TileDisplayMode.FaceUp);
 */
@ccclass('TileRenderer')
export class TileRenderer extends Component {

    private _tileId: number = 0;
    private _tileInfo: TileInfo | null = null;
    private _displayMode: TileDisplayMode = TileDisplayMode.FaceUp;

    get tileId(): number { return this._tileId; }
    get tileInfo(): TileInfo | null { return this._tileInfo; }
    get displayMode(): TileDisplayMode { return this._displayMode; }

    /**
     * 設定要顯示的牌
     * @param tileId 牌 ID (1-144)，0 表示未知牌
     * @param mode 顯示模式
     */
    public setTile(tileId: number, mode: TileDisplayMode = TileDisplayMode.FaceUp) {
        this._tileId = tileId;
        this._tileInfo = tileId > 0 ? parseTileId(tileId) : null;
        this._displayMode = mode;
        this.rebuild();
    }

    /** 切換顯示模式 */
    public setDisplayMode(mode: TileDisplayMode) {
        this._displayMode = mode;
        this.rebuild();
    }

    private rebuild() {
        this.node.removeAllChildren();

        const isFaceUp = this._displayMode === TileDisplayMode.FaceUp
            || this._displayMode === TileDisplayMode.FaceUpWithTop
            || this._displayMode === TileDisplayMode.FaceUpWithRight;

        const hasTop = this._displayMode === TileDisplayMode.FaceUpWithTop
            || this._displayMode === TileDisplayMode.FaceDownWithTop;

        const hasRight = this._displayMode === TileDisplayMode.FaceUpWithRight;

        const sideOnly = this._displayMode === TileDisplayMode.SideTopOnly;

        // 計算整體節點大小
        let totalW = TILE_WIDTH;
        let totalH = TILE_HEIGHT;
        if (hasTop) totalH = TILE_HEIGHT + SIDE_TOP_HEIGHT;
        if (hasRight) totalW = TILE_WIDTH + SIDE_RIGHT_WIDTH;
        if (sideOnly) { totalW = TILE_WIDTH; totalH = SIDE_TOP_HEIGHT; }

        let transform = this.node.getComponent(UITransform);
        if (!transform) transform = this.node.addComponent(UITransform);
        transform.setContentSize(new Size(totalW, totalH));

        if (sideOnly) {
            this.buildSideTop(0, 0, false);
            return;
        }

        // 上側面（在牌面正上方）
        if (hasTop) {
            const isBack = this._displayMode === TileDisplayMode.FaceDownWithTop;
            this.buildSideTop(0, totalH / 2 - SIDE_TOP_HEIGHT / 2, isBack);
        }

        // 牌面主體
        const faceY = hasTop ? -(SIDE_TOP_HEIGHT / 2) : 0;
        if (isFaceUp) {
            this.buildFace(0, faceY);
        } else {
            this.buildBack(0, faceY);
        }

        // 右側面
        if (hasRight) {
            this.buildSideRight(TILE_WIDTH / 2 + SIDE_RIGHT_WIDTH / 2, faceY);
        }
    }

    // ============================================
    // 正面
    // ============================================
    private buildFace(x: number, y: number) {
        const faceNode = new Node("face");
        const ft = faceNode.addComponent(UITransform);
        ft.setContentSize(new Size(TILE_WIDTH, TILE_HEIGHT));
        faceNode.setPosition(x, y, 0);

        // 背景用 Graphics 繪製圓角矩形
        this.drawTileBackground(faceNode, TILE_WIDTH, TILE_HEIGHT,
            new Color(255, 254, 245, 255), // #FFFEF5
            new Color(212, 201, 168, 255), // #D4C9A8 border
        );

        this.node.addChild(faceNode);

        // 載入正面圖片
        if (this._tileInfo) {
            const imgPath = getTileImagePath(this._tileInfo);
            if (imgPath) {
                this.loadTileImage(faceNode, imgPath);
            } else {
                // fallback 文字 (理論上不會走到這裡)
                const display = getTileTextDisplay(this._tileInfo);
                if (display) this.buildTextContent(faceNode, display);
            }
        }
    }

    // ============================================
    // 背面
    // ============================================
    private buildBack(x: number, y: number) {
        const backNode = new Node("back");
        const bt = backNode.addComponent(UITransform);
        bt.setContentSize(new Size(TILE_WIDTH, TILE_HEIGHT));
        backNode.setPosition(x, y, 0);

        // 綠色背景 + 金色邊框
        this.drawTileBackground(backNode, TILE_WIDTH, TILE_HEIGHT,
            new Color(27, 94, 32, 255),   // #1B5E20
            new Color(255, 215, 0, 255),  // #FFD700
        );

        // 載入背面圖片
        this.loadTileImage(backNode, "tiles/tile_back");

        this.node.addChild(backNode);
    }

    // ============================================
    // 上側面
    // ============================================
    private buildSideTop(x: number, y: number, isBack: boolean) {
        const sideNode = new Node("sideTop");
        const st = sideNode.addComponent(UITransform);
        st.setContentSize(new Size(TILE_WIDTH, SIDE_TOP_HEIGHT));
        sideNode.setPosition(x, y, 0);

        if (isBack) {
            // 背面上側：深綠色 + 金邊
            this.drawTileBackground(sideNode, TILE_WIDTH, SIDE_TOP_HEIGHT,
                new Color(30, 77, 30, 255),
                new Color(255, 215, 0, 255),
                [4, 4, 0, 0]
            );
        } else {
            // 正面上側：米色漸層效果
            this.drawTileBackground(sideNode, TILE_WIDTH, SIDE_TOP_HEIGHT,
                new Color(232, 223, 200, 255), // #E8DFC8
                new Color(196, 184, 152, 255), // #C4B898
                [4, 4, 0, 0]
            );
        }

        this.node.addChild(sideNode);
    }

    // ============================================
    // 右側面
    // ============================================
    private buildSideRight(x: number, y: number) {
        const sideNode = new Node("sideRight");
        const st = sideNode.addComponent(UITransform);
        st.setContentSize(new Size(SIDE_RIGHT_WIDTH, TILE_HEIGHT));
        sideNode.setPosition(x, y, 0);

        this.drawTileBackground(sideNode, SIDE_RIGHT_WIDTH, TILE_HEIGHT,
            new Color(232, 223, 200, 255),
            new Color(196, 184, 152, 255),
            [0, 4, 4, 0]
        );

        this.node.addChild(sideNode);
    }

    // ============================================
    // 工具方法
    // ============================================

    /** 用 Graphics 繪製圓角矩形背景 */
    private drawTileBackground(
        node: Node, w: number, h: number,
        fillColor: Color, strokeColor: Color,
        radius: number[] = [6, 6, 6, 6]
    ) {
        const g = node.addComponent(Graphics);
        const hw = w / 2, hh = h / 2;
        const r = radius[0]; // 簡化：四角統一使用第一個值
        g.fillColor = fillColor;
        g.strokeColor = strokeColor;
        g.lineWidth = 1;
        g.roundRect(-hw, -hh, w, h, r);
        g.fill();
        g.stroke();
    }

    /** 動態載入圖片到指定節點 */
    private loadTileImage(parentNode: Node, imgPath: string) {
        const contentNode = new Node("imgContent");
        const ct = contentNode.addComponent(UITransform);
        ct.setContentSize(new Size(CONTENT_WIDTH, CONTENT_HEIGHT));
        const sprite = contentNode.addComponent(Sprite);
        sprite.sizeMode = Sprite.SizeMode.CUSTOM;
        parentNode.addChild(contentNode);

        resources.load(imgPath + "/spriteFrame", SpriteFrame, (err, spriteFrame) => {
            if (err) {
                resources.load(imgPath, ImageAsset, (err2, imageAsset) => {
                    if (err2 || !sprite.isValid) {
                        console.warn(`[TileRenderer] 載入圖片失敗: ${imgPath}`, err2);
                        return;
                    }
                    const texture = new Texture2D();
                    texture.image = imageAsset;
                    const sf = new SpriteFrame();
                    sf.texture = texture;
                    sprite.spriteFrame = sf;
                });
                return;
            }
            if (sprite.isValid) {
                sprite.spriteFrame = spriteFrame;
            }
        });
    }

    /** 文字 fallback (備用) */
    private buildTextContent(parentNode: Node, display: { topText: string; topColor: string; topSize: number; bottomText?: string; bottomColor?: string; bottomSize?: number }) {
        const topNode = new Node("topText");
        const topLabel = topNode.addComponent(Label);
        topLabel.string = display.topText;
        topLabel.fontSize = display.topSize;
        topLabel.lineHeight = display.topSize + 4;
        topLabel.color = this.hexToColor(display.topColor);
        parentNode.addChild(topNode);

        if (display.bottomText) {
            const bottomNode = new Node("bottomText");
            const bottomLabel = bottomNode.addComponent(Label);
            bottomLabel.string = display.bottomText;
            bottomLabel.fontSize = display.bottomSize || 16;
            bottomLabel.lineHeight = (display.bottomSize || 16) + 4;
            bottomLabel.color = this.hexToColor(display.bottomColor || "#1A1A1A");
            topNode.setPosition(0, 12, 0);
            bottomNode.setPosition(0, -18, 0);
            parentNode.addChild(bottomNode);
        }
    }

    private hexToColor(hex: string): Color {
        hex = hex.replace('#', '');
        const r = parseInt(hex.substring(0, 2), 16);
        const g = parseInt(hex.substring(2, 4), 16);
        const b = parseInt(hex.substring(4, 6), 16);
        const a = hex.length >= 8 ? parseInt(hex.substring(6, 8), 16) : 255;
        return new Color(r, g, b, a);
    }

    // ============================================
    // 靜態工廠方法
    // ============================================

    /** 建立正面朝上的牌節點 */
    public static createTileNode(tileId: number, mode: TileDisplayMode = TileDisplayMode.FaceUp): Node {
        const node = new Node(`tile_${tileId}`);
        const renderer = node.addComponent(TileRenderer);
        renderer.setTile(tileId, mode);
        return node;
    }

    /** 建立正面+上側面的立牌 (本機手牌常用) */
    public static createStandingTile(tileId: number): Node {
        return TileRenderer.createTileNode(tileId, TileDisplayMode.FaceUpWithTop);
    }

    /** 建立背面+上側面的立牌 (對手手牌常用) */
    public static createOpponentTile(): Node {
        return TileRenderer.createTileNode(0, TileDisplayMode.FaceDownWithTop);
    }

    /** 建立正面朝上的平放牌 (棄牌區常用) */
    public static createDiscardTile(tileId: number): Node {
        return TileRenderer.createTileNode(tileId, TileDisplayMode.FaceUp);
    }
}
