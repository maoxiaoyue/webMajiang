/**
 * 基本的 Cocos Creator "cc" 模組假宣告檔 (Stub Declaration)
 * 讓 VSCode 可以在沒有開啟 Cocos Creator 的情況下識別 cc 模組，消除紅色波浪線。
 */
declare module 'cc' {
    export class EventTarget {
        on(type: string, callback: any, target?: any, useCapture?: any): any;
        off(type: string, callback?: any, target?: any): void;
        targetOff(target: any): void;
        once(type: string, callback: any, target?: any): any;
        emit(type: string, ...args: any[]): void;
    }

    export class Component {
        node: Node;
        name: string;
        uuid: string;
        protected onLoad(): void;
        protected start(): void;
        protected update(dt: number): void;
        protected lateUpdate(dt: number): void;
        protected onDestroy(): void;
        protected onEnable(): void;
        protected onDisable(): void;
    }

    export class Node extends EventTarget {
        name: string;
        uuid: string;
        parent: Node | null;
        children: Node[];
        active: boolean;
        getComponent<T extends Component>(type: { prototype: T }): T | null;
        getComponent<T extends Component>(className: string): T | null;
        addComponent<T extends Component>(type: { prototype: T }): T;
        addComponent<T extends Component>(className: string): T;
    }

    export class Label extends Component {
        string: string;
        horizontalAlign: number;
        verticalAlign: number;
        actualFontSize: number;
        fontSize: number;
    }

    export class Sprite extends Component {
        spriteFrame: any;
    }

    /**
     * Cocos Creator 裝飾器
     */
    export function _decorator(target: any): void;
    export namespace _decorator {
        export function ccclass(name?: string): ClassDecorator;
        export function property(options?: any): PropertyDecorator;
    }
}
