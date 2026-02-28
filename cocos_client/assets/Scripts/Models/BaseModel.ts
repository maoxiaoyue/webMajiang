import { EventMgr } from '../Events/EventMgr';

/**
 * BaseModel: 數據層的基礎類
 * 負責維護數據狀態，並在數據改變時拋出事件通知 View/ViewModel
 */
export class BaseModel {
    /**
     * 發送數據變化事件給全域或綁定該事件的監聽者
     * @param eventName 事件名稱
     * @param data 傳遞的數據
     */
    protected emitChange(eventName: string, data?: any) {
        EventMgr.emit(eventName, data);
    }
}
