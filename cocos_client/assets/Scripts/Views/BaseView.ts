import { Component, _decorator } from 'cc';
import { BaseViewModel } from '../ViewModels/BaseViewModel';
const { ccclass, property } = _decorator;

/**
 * BaseView: UI 視圖層的基礎類
 * 負責處理 Cocos UI 元件的顯示與互動，並將業務邏輯委託給 ViewModel
 */
@ccclass('BaseView')
export class BaseView extends Component {

    // 子類別需要定義與綁定對應的 ViewModel
    protected viewModel!: BaseViewModel<any>;

    /**
     * 子類實現：綁定 ViewModel 及其事件
     */
    protected bindViewModel() {
        // 例: EventMgr.on('data_change', this.updateUI, this);
    }

    protected onLoad() {
        this.bindViewModel();
    }

    protected onDestroy() {
        // 解除綁定以防記憶體洩漏 (若有註冊 EventMgr 的監聽器)
    }
}
