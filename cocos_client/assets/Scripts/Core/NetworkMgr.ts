import { EventMgr } from '../Events/EventMgr';

export class NetworkMgr {
    private static _instance: NetworkMgr = null!;
    public static get instance(): NetworkMgr {
        if (!this._instance) {
            this._instance = new NetworkMgr();
        }
        return this._instance;
    }

    private ws: WebSocket | null = null;
    private isConnected: boolean = false;
    private reconnectTimer: any = null;
    private url: string = "";

    // 定義連線的事件常數
    public static readonly EVENT_CONNECTED = "net_connected";
    public static readonly EVENT_DISCONNECTED = "net_disconnected";
    public static readonly EVENT_ERROR = "net_error";
    // 從 Go Server 收到訊息的統一事件
    public static readonly EVENT_MESSAGE_RECEIVED = "net_message_received";

    private constructor() { }

    /**
     * 初始化連線
     * @param url WebSocket 的連線網址 (例如 ws://localhost:8080/ws)
     */
    public connect(url: string) {
        if (this.ws) {
            if (this.ws.readyState === WebSocket.OPEN) return;
            this.ws.close();
        }

        this.url = url;
        console.log(`[NetworkMgr] 嘗試連線至: ${this.url}`);
        this.ws = new WebSocket(url);
        this.ws.binaryType = "arraybuffer"; // 使用二進位傳輸

        this.ws.onopen = this.onOpen.bind(this);
        this.ws.onmessage = this.onMessage.bind(this);
        this.ws.onerror = this.onError.bind(this);
        this.ws.onclose = this.onClose.bind(this);
    }

    /**
     * 關閉連線
     */
    public disconnect() {
        if (this.ws) {
            this.ws.close();
        }
    }

    /**
     * 傳送資料給伺服器
     * @param action 對應的 action (如 "player_action", "join_room")
     * @param data 一個 Uint8Array 的 payload，或者是可以直接帶入的 JSON 讓對應的 proto 產生
     */
    public send(action: string, data: any = {}) {
        if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
            console.warn("[NetworkMgr] 未連線，無法發送資料!");
            return false;
        }

        // 必須把傳入的 JS Object 依照 action 名稱轉出對應的 Uint8Array data (例如 JoinRoomReq)
        let innerData = new Uint8Array();
        try {
            if (action === "join_room") {
                innerData = window.mahjong_pb.encodeJoinRoomReq(data) as unknown as Uint8Array;
            } else if (action === "discard_tile" || action === "player_action") {
                // 若兩者共用 pb.PlayerActionData
                innerData = window.mahjong_pb.encodePlayerActionData(data) as unknown as Uint8Array;
            }
        } catch (e) {
            console.error(`[NetworkMgr] proto encode payload err for action ${action}`, e);
            return false;
        }

        const msg = {
            action: action,
            data: innerData
        };

        const buffer = window.mahjong_pb.encodeWSMessage(msg) as unknown as ArrayBuffer;
        this.ws.send(buffer);
        // console.log(`[NetworkMgr] 發送 Proto 訊息: ${action}`, buffer.byteLength);
        return true;
    }

    private onOpen(ev: Event) {
        console.log("[NetworkMgr] 連線成功!");
        this.isConnected = true;
        if (this.reconnectTimer) {
            clearTimeout(this.reconnectTimer);
            this.reconnectTimer = null;
        }
        EventMgr.emit(NetworkMgr.EVENT_CONNECTED);
    }

    private onMessage(ev: MessageEvent) {
        try {
            // 透過 WebSocket 設定 binaryType = "arraybuffer"，所以 ev.data 預期是 ArrayBuffer
            if (!(ev.data instanceof ArrayBuffer)) {
                console.warn("[NetworkMgr] 收到非二進位(非 Protobuf) 的資料");
                return;
            }

            const uint8Array = new Uint8Array(ev.data);
            const msg = window.mahjong_pb.decodeWSMessage(uint8Array);

            // 解析 inner Data
            let innerData: any = {};
            try {
                if (msg.action === "sync_state") {
                    innerData = window.mahjong_pb.decodeSyncStateData(msg.data);
                } else if (msg.action === "deal_tiles") {
                    innerData = window.mahjong_pb.decodeDealTilesData(msg.data);
                } else if (msg.action.endsWith("_res")) {
                    if (msg.action === "join_room_res") {
                        innerData = window.mahjong_pb.decodeJoinRoomRes(msg.data);
                    } else if (msg.action === "player_action_res") {
                        innerData = window.mahjong_pb.decodePlayerActionRes(msg.data);
                    }
                }
            } catch (innerErr) {
                console.error(`[NetworkMgr] Failed to decode inner proto for ${msg.action}`, innerErr);
            }

            // console.log(`[NetworkMgr] 收到 Proto 訊息 ${msg.action}:`, innerData);

            // 將解析後的物件透過 EventMgr 拋出給 ViewModel 訂閱者
            EventMgr.emit(NetworkMgr.EVENT_MESSAGE_RECEIVED, { action: msg.action, data: innerData });

            // 動態拋出對應的事件名稱
            EventMgr.emit(`ws_${msg.action}`, innerData);

        } catch (e) {
            console.error("[NetworkMgr] 解析 Protobuf 失敗", e);
        }
    }

    private onError(ev: Event) {
        console.error("[NetworkMgr] 連線錯誤!", ev);
        EventMgr.emit(NetworkMgr.EVENT_ERROR);
    }

    private onClose(ev: CloseEvent) {
        console.log("[NetworkMgr] 連線關閉!", ev.code, ev.reason);
        this.isConnected = false;
        EventMgr.emit(NetworkMgr.EVENT_DISCONNECTED);

        // 可看需求決定是否實作斷線重連機制
        // this.scheduleReconnect();
    }

    private scheduleReconnect() {
        if (this.reconnectTimer) return;
        console.log("[NetworkMgr] 3秒後嘗試重新連線...");
        this.reconnectTimer = setTimeout(() => {
            this.reconnectTimer = null;
            this.connect(this.url);
        }, 3000);
    }
}
