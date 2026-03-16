import { _decorator, Component, Node, Sprite, Label, Color, UITransform, SpriteFrame, resources, ImageAsset, Texture2D, Size, Layers, Vec3 } from 'cc';
import { TileInfo, parseTileId, getTileImagePath, getTileTextDisplay } from './TileConfig';

const { ccclass, property } = _decorator;

const UI_2D = Layers.Enum.UI_2D;

function uiNode(name: string): Node {
    const n = new Node(name);
    n.layer = UI_2D;
    return n;
}

/** 牌面顯示尺寸 (圖片原始比例 960:1166 ≈ 0.823) */
const TILE_WIDTH = 36;
const TILE_HEIGHT = 44;

/** 牌背顯示尺寸 (拉伸為長方形，與牌面同比例) */
const BACK_WIDTH = 36;
const BACK_HEIGHT = 44;

/**
 * 牌的顯示模式
 */
export enum TileDisplayMode {
    /** 正面朝上 */
    FaceUp = 0,
    /** 背面朝上 */
    FaceDown = 1,
}

/**
 * TileRenderer: 渲染單張麻將牌
 *
 * 圖片素材已經是立體牌面，直接顯示即可。
 */
@ccclass('TileRenderer')
export class TileRenderer extends Component {

    private _tileId: number = 0;
    private _tileInfo: TileInfo | null = null;
    private _displayMode: TileDisplayMode = TileDisplayMode.FaceUp;

    get tileId(): number { return this._tileId; }
    get tileInfo(): TileInfo | null { return this._tileInfo; }
    get displayMode(): TileDisplayMode { return this._displayMode; }

    public setTile(tileId: number, mode: TileDisplayMode = TileDisplayMode.FaceUp) {
        this._tileId = tileId;
        this._tileInfo = tileId > 0 ? parseTileId(tileId) : null;
        this._displayMode = mode;
        this.rebuild();
    }

    public setDisplayMode(mode: TileDisplayMode) {
        this._displayMode = mode;
        this.rebuild();
    }

    private rebuild() {
        this.node.layer = UI_2D;
        this.node.removeAllChildren();

        const isFaceUp = this._displayMode === TileDisplayMode.FaceUp;

        let w: number, h: number;
        if (isFaceUp) {
            w = TILE_WIDTH;
            h = TILE_HEIGHT;
        } else {
            w = BACK_WIDTH;
            h = BACK_HEIGHT;
        }

        let transform = this.node.getComponent(UITransform);
        if (!transform) transform = this.node.addComponent(UITransform);
        transform.setContentSize(new Size(w, h));

        if (isFaceUp && this._tileInfo) {
            const imgPath = getTileImagePath(this._tileInfo);
            if (imgPath) {
                this.loadImage(imgPath, w, h);
            } else {
                const display = getTileTextDisplay(this._tileInfo);
                if (display) this.buildTextFallback(display);
            }
        } else {
            // 背面
            this.loadImage('tiles/tile_back', w, h);
        }
    }

    /** 載入圖片到自身節點 */
    private loadImage(imgPath: string, w: number, h: number) {
        const spriteNode = uiNode('sprite');
        const st = spriteNode.addComponent(UITransform);
        st.setContentSize(new Size(w, h));
        const sprite = spriteNode.addComponent(Sprite);
        sprite.sizeMode = Sprite.SizeMode.CUSTOM;
        this.node.addChild(spriteNode);

        resources.load(imgPath + '/spriteFrame', SpriteFrame, (err, spriteFrame) => {
            if (err) {
                resources.load(imgPath, ImageAsset, (err2, imageAsset) => {
                    if (err2 || !sprite.isValid) return;
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

    /** 文字 fallback */
    private buildTextFallback(display: { topText: string; topColor: string; topSize: number; bottomText?: string; bottomColor?: string; bottomSize?: number }) {
        const node = uiNode('text');
        const label = node.addComponent(Label);
        label.string = display.topText;
        label.fontSize = display.topSize;
        label.color = this.hexToColor(display.topColor);
        this.node.addChild(node);
    }

    private hexToColor(hex: string): Color {
        hex = hex.replace('#', '');
        const r = parseInt(hex.substring(0, 2), 16);
        const g = parseInt(hex.substring(2, 4), 16);
        const b = parseInt(hex.substring(4, 6), 16);
        return new Color(r, g, b, 255);
    }

    // ============================================
    // 靜態工廠方法
    // ============================================

    /** 建立正面牌 */
    public static createTileNode(tileId: number, mode: TileDisplayMode = TileDisplayMode.FaceUp): Node {
        const node = uiNode(`tile_${tileId}`);
        const renderer = node.addComponent(TileRenderer);
        renderer.setTile(tileId, mode);
        return node;
    }

    /** 建立正面立牌 (自己手牌) */
    public static createStandingTile(tileId: number): Node {
        return TileRenderer.createTileNode(tileId, TileDisplayMode.FaceUp);
    }

    /** 建立背面牌 (對手手牌) */
    public static createOpponentTile(): Node {
        return TileRenderer.createTileNode(0, TileDisplayMode.FaceDown);
    }

    /** 建立正面平放牌 (棄牌區) */
    public static createDiscardTile(tileId: number): Node {
        return TileRenderer.createTileNode(tileId, TileDisplayMode.FaceUp);
    }
}
