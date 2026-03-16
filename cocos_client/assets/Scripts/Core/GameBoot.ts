/**
 * GameBoot: 自動啟動器
 *
 * 運作機制：
 *   1. Cocos 引擎載入 main.scene
 *   2. 本模組監聽 EVENT_AFTER_SCENE_LAUNCH
 *   3. 在場景中找到或建立 Canvas + 2D Camera
 *   4. 建立 GameRoot 子節點並掛載 GameSceneSetup
 *   5. GameSceneSetup.onLoad() 建構完整牌桌 UI
 */
import {
    _decorator, director, Director, Node, find, Component, Layers,
    Canvas, Camera, UITransform, Widget, Size, Color, view
} from 'cc';
import { GameSceneSetup } from '../Views/Game/GameSceneSetup';

const { ccclass } = _decorator;

/**
 * @ccclass 確保此模組被 Cocos Creator 打包系統收錄。
 * 類別本身不需掛到場景上，純粹利用模組載入時的副作用。
 */
@ccclass('GameBoot')
class GameBoot extends Component {}

// 場景載入後自動執行
director.on(Director.EVENT_AFTER_SCENE_LAUNCH, () => {
    const scene = director.getScene();
    if (!scene) {
        console.error('[GameBoot] director.getScene() 回傳 null');
        return;
    }

    // 1. 嘗試找到既有的 Canvas 節點 (按名稱 or 組件)
    let canvasNode = find('Canvas');
    if (!canvasNode) {
        // 遍歷場景根節點的子節點，看有沒有掛 Canvas 組件的
        scene.children.forEach((child) => {
            if (!canvasNode && child.getComponent(Canvas)) {
                canvasNode = child;
            }
        });
    }

    // 2. 如果完全找不到 Canvas，就自行建立
    if (!canvasNode) {
        console.log('[GameBoot] 場景中無 Canvas，自動建立 Canvas + Camera2D');
        canvasNode = createCanvasNode(scene);
    } else {
        console.log(`[GameBoot] 找到 Canvas 節點: "${canvasNode.name}"`);
    }

    // 3. 避免重複掛載
    if (canvasNode.getComponentInChildren(GameSceneSetup)) {
        console.log('[GameBoot] GameSceneSetup 已存在，跳過');
        return;
    }

    // 4. 建立 GameRoot 並掛載 GameSceneSetup
    const gameRoot = new Node('GameRoot');
    gameRoot.layer = Layers.Enum.UI_2D;
    const rootTransform = gameRoot.addComponent(UITransform);
    rootTransform.setContentSize(new Size(1280, 720));
    canvasNode.addChild(gameRoot);
    gameRoot.addComponent(GameSceneSetup);

    console.log('[GameBoot] 遊戲場景自動初始化完成');
});

/**
 * 建立 Canvas + Camera2D 節點並加入場景
 */
function createCanvasNode(scene: Node): Node {
    // Canvas 節點
    const canvasNode = new Node('Canvas');
    canvasNode.layer = Layers.Enum.UI_2D;

    const uiTransform = canvasNode.addComponent(UITransform);
    uiTransform.setContentSize(new Size(1280, 720));

    const widget = canvasNode.addComponent(Widget);
    widget.isAlignLeft = true;
    widget.isAlignRight = true;
    widget.isAlignTop = true;
    widget.isAlignBottom = true;
    widget.left = 0;
    widget.right = 0;
    widget.top = 0;
    widget.bottom = 0;

    // Camera 子節點
    const cameraNode = new Node('Camera');
    cameraNode.layer = Layers.Enum.UI_2D;
    cameraNode.setPosition(0, 0, 1000);

    const camUITransform = cameraNode.addComponent(UITransform);
    camUITransform.setContentSize(new Size(1280, 720));

    const cam = cameraNode.addComponent(Camera);
    cam.projection = Camera.ProjectionType.ORTHO;
    cam.orthoHeight = view.getVisibleSize().height / 2;
    cam.near = 0;
    cam.far = 2000;
    cam.clearFlags = Camera.ClearFlag.SOLID_COLOR;
    cam.clearColor = new Color(7, 82, 45, 255); // 深綠色背景
    cam.visibility = Layers.Enum.UI_2D;

    canvasNode.addChild(cameraNode);

    // 掛載 Canvas 組件並指定 Camera
    const canvasComp = canvasNode.addComponent(Canvas);
    canvasComp.cameraComponent = cam;
    canvasComp.alignCanvasWithScreen = true;

    scene.addChild(canvasNode);
    return canvasNode;
}
