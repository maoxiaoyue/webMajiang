/**
 * GameBoot: 遊戲場景自動啟動器 (Component 版本)
 *
 * 掛載到場景中任意節點（如編輯器建立的 Canvas 節點）。
 * start() 會自動補齊 2D 渲染所需的元件：
 *   Canvas component、UITransform、Camera2D、Widget
 * 然後建立 GameRoot 掛載 GameSceneSetup。
 */
import {
    _decorator, Node, Component, Layers,
    Canvas, Camera, UITransform, Widget, Size, Color, view, find
} from 'cc';
import { GameSceneSetup } from '../Views/Game/GameSceneSetup';

const { ccclass } = _decorator;

const UI_2D = Layers.Enum.UI_2D;

@ccclass('GameBoot')
export class GameBoot extends Component {

    start(): void {
        console.log('[GameBoot] start() 開始初始化...');

        const canvasNode = this.node;

        // 1. 確保節點在 UI_2D layer
        canvasNode.layer = UI_2D;
        console.log(`[GameBoot] 設定 Canvas layer = UI_2D (${UI_2D})`);

        // 2. 補齊 UITransform
        if (!canvasNode.getComponent(UITransform)) {
            const uiTransform = canvasNode.addComponent(UITransform);
            const designSize = view.getDesignResolutionSize();
            uiTransform.setContentSize(new Size(designSize.width, designSize.height));
            console.log(`[GameBoot] 新增 UITransform (${designSize.width}x${designSize.height})`);
        }

        // 3. 補齊 Canvas component
        if (!canvasNode.getComponent(Canvas)) {
            canvasNode.addComponent(Canvas);
            console.log('[GameBoot] 新增 Canvas component');
        }

        // 4. 補齊 Widget（全螢幕自適應）
        if (!canvasNode.getComponent(Widget)) {
            const widget = canvasNode.addComponent(Widget);
            widget.isAlignTop = true;
            widget.isAlignBottom = true;
            widget.isAlignLeft = true;
            widget.isAlignRight = true;
            widget.top = 0;
            widget.bottom = 0;
            widget.left = 0;
            widget.right = 0;
            console.log('[GameBoot] 新增 Widget (全螢幕)');
        }

        // 5. 確保有 2D Camera
        this.ensure2DCamera(canvasNode);

        // 6. 避免重複掛載
        if (canvasNode.getComponentInChildren(GameSceneSetup)) {
            console.log('[GameBoot] GameSceneSetup 已存在，跳過');
            return;
        }

        // 7. 建立 GameRoot 並掛載 GameSceneSetup
        const gameRoot = new Node('GameRoot');
        gameRoot.layer = UI_2D;
        const rootTransform = gameRoot.addComponent(UITransform);
        rootTransform.setContentSize(new Size(1280, 720));
        canvasNode.addChild(gameRoot);
        gameRoot.addComponent(GameSceneSetup);

        console.log('[GameBoot] 遊戲場景初始化完成');
    }

    /**
     * 確保場景中有一台能看到 UI_2D layer 的 2D Camera。
     * 如果現有 Camera 都是 3D (perspective)，就在 canvasNode 下新建一台。
     */
    private ensure2DCamera(canvasNode: Node): void {
        // 檢查場景中是否已有 2D Camera (orthographic + visibility 含 UI_2D)
        const scene = canvasNode.scene;
        const allCameras = scene.getComponentsInChildren(Camera);
        for (const cam of allCameras) {
            // projection 0 = ortho, 1 = perspective
            if (cam.projection === 0 && (cam.visibility & UI_2D)) {
                console.log('[GameBoot] 找到現有 2D Camera，無需建立');
                return;
            }
        }

        // 沒有合適的 2D Camera → 建立一台
        const camNode = new Node('Camera2D');
        camNode.layer = UI_2D;
        camNode.setPosition(0, 0, 1000);

        const cam = camNode.addComponent(Camera);
        cam.projection = 0;           // orthographic
        cam.near = 0;
        cam.far = 2000;
        cam.orthoHeight = view.getDesignResolutionSize().height / 2;
        cam.visibility = UI_2D;
        cam.clearFlags = Camera.ClearFlag.SOLID_COLOR;
        cam.clearColor = new Color(0, 0, 0, 255);
        cam.priority = 1;  // 比 3D 相機高，確保最後渲染

        canvasNode.addChild(camNode);
        console.log('[GameBoot] 建立 2D Camera (orthographic, priority=1)');
    }
}
