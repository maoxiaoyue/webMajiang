import { EventTarget } from 'cc';

/**
 * 全局事件管理器 (Event Manager)
 * 基於 Cocos Creator 的 EventTarget 實現
 */
export const EventMgr = new EventTarget();

// 暴露給 LobbyBridge (index.html) 使用，讓 LobbyBridge 的 WebSocket 事件能轉發到 Cocos
(window as any).__COCOS_EVENT_MGR__ = EventMgr;
