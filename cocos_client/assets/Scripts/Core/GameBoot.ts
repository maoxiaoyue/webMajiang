/**
 * GameBoot: 自動啟動器
 *
 * 此模組在 Cocos Creator 打包時會被引入，
 * 利用 director.on(EVENT_AFTER_SCENE_LAUNCH) 在場景載入後自動建構遊戲 UI。
 *
 * 運作機制：
 *   1. Cocos 引擎載入 main.scene（含 Canvas + Camera2D）
 *   2. 本模組監聽場景啟動事件
 *   3. 找到 Canvas 節點，建立 GameRoot 子節點
 *   4. 在 GameRoot 上掛載 GameSceneSetup 組件
 *   5. GameSceneSetup.onLoad() 建構完整牌桌 UI
 */
import { _decorator, director, Director, Node, find, Component } from 'cc';
import { GameSceneSetup } from '../Views/Game/GameSceneSetup';

const { ccclass } = _decorator;

/**
 * 使用 @ccclass 確保此模組被 Cocos Creator 打包系統收錄。
 * 類別本身不需要掛到場景上，純粹利用模組載入時的副作用。
 */
@ccclass('GameBoot')
class GameBoot extends Component {}

// 使用 director 事件在場景載入後自動執行
director.on(Director.EVENT_AFTER_SCENE_LAUNCH, () => {
    // 尋找 Canvas 節點
    const canvas = find('Canvas');
    if (!canvas) {
        console.error('[GameBoot] 找不到 Canvas 節點！請確認 main.scene 中有 Canvas。');
        return;
    }

    // 避免重複掛載
    if (canvas.getComponentInChildren(GameSceneSetup)) {
        console.log('[GameBoot] GameSceneSetup 已存在，跳過');
        return;
    }

    // 建立 GameRoot 節點並掛載 GameSceneSetup
    const gameRoot = new Node('GameRoot');
    canvas.addChild(gameRoot);

    gameRoot.addComponent(GameSceneSetup);
    console.log('[GameBoot] 遊戲場景自動初始化完成');
});
