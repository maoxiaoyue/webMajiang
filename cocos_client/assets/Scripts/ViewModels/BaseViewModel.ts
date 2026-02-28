import { BaseModel } from '../Models/BaseModel';

/**
 * BaseViewModel: 邏輯與視圖的中間層
 * 負責將 Model 的數據轉換為 View 可以直接使用的格式，並處理來自 View 的指令以更新 Model
 */
export class BaseViewModel<T extends BaseModel> {

    protected model!: T;

    constructor(model?: T) {
        if (model) {
            this.model = model;
        }
    }

    /**
     * 設置對應的 Model
     * @param model 
     */
    public setModel(model: T) {
        this.model = model;
    }

    /**
     * 獲取對應的 Model
     */
    public getModel(): T {
        return this.model;
    }
}
