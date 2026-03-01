/**
 * 麻將牌類型與配置
 * 對應後端 models/tile.go 的 TileType
 */
export enum TileType {
    Wan = 0,     // 萬
    Tong = 1,    // 筒
    Tiao = 2,    // 條
    Wind = 3,    // 風 (東南西北)
    Dragon = 4,  // 元 (中發白)
    Flower = 5,  // 花
}

export interface TileInfo {
    type: TileType;
    value: number;
}

// 風牌名稱
const WindNames: Record<number, string> = { 1: "東", 2: "南", 3: "西", 4: "北" };
// 三元牌名稱
const DragonNames: Record<number, string> = { 1: "中", 2: "發", 3: "白" };
// 花牌名稱
const FlowerNames: Record<number, string> = {
    1: "梅", 2: "蘭", 3: "竹", 4: "菊",
    5: "春", 6: "夏", 7: "秋", 8: "冬"
};
// 萬子中文數字
const WanNumbers: Record<number, string> = {
    1: "一", 2: "二", 3: "三", 4: "四", 5: "五",
    6: "六", 7: "七", 8: "八", 9: "九"
};

/**
 * 根據牌 ID (1-144) 解析出 TileType 和 Value
 * 對應 Go 端 GenerateAllTiles() 的 ID 分配:
 *   1-36:  萬 (1-9萬，各4張)
 *   37-72: 筒 (1-9筒，各4張)
 *   73-108: 條 (1-9條，各4張)
 *   109-124: 風 (東南西北，各4張)
 *   125-136: 元 (中發白，各4張)
 *   137-144: 花 (梅蘭竹菊春夏秋冬，各1張)
 */
export function parseTileId(id: number): TileInfo {
    if (id >= 1 && id <= 36) {
        return { type: TileType.Wan, value: Math.ceil(id / 4) };
    } else if (id >= 37 && id <= 72) {
        return { type: TileType.Tong, value: Math.ceil((id - 36) / 4) };
    } else if (id >= 73 && id <= 108) {
        return { type: TileType.Tiao, value: Math.ceil((id - 72) / 4) };
    } else if (id >= 109 && id <= 124) {
        return { type: TileType.Wind, value: Math.ceil((id - 108) / 4) };
    } else if (id >= 125 && id <= 136) {
        return { type: TileType.Dragon, value: Math.ceil((id - 124) / 4) };
    } else if (id >= 137 && id <= 144) {
        return { type: TileType.Flower, value: id - 136 };
    }
    return { type: TileType.Wan, value: 1 }; // fallback
}

/**
 * 取得牌在 resources/tiles/ 下的圖片路徑 (不含副檔名)
 * 有 AI 圖片的牌回傳路徑，純文字牌回傳 null
 */
export function getTileImagePath(info: TileInfo): string | null {
    switch (info.type) {
        case TileType.Wan:
            return `tiles/wan_${info.value}`;
        case TileType.Tong:
            return `tiles/tong_${info.value}`;
        case TileType.Tiao:
            return `tiles/tiao_${info.value}`;
        case TileType.Wind:
            return `tiles/wind_${info.value}`;
        case TileType.Flower:
            return `tiles/flower_${info.value}`;
        case TileType.Dragon:
            return `tiles/dragon_${info.value}`;
        default:
            return null;
    }
}

/** 文字牌的顯示資訊 */
export interface TileTextDisplay {
    topText: string;
    topColor: string;
    topSize: number;
    bottomText?: string;
    bottomColor?: string;
    bottomSize?: number;
}

/**
 * 取得純文字牌的顯示資訊（萬子、字牌等）
 */
export function getTileTextDisplay(info: TileInfo): TileTextDisplay | null {
    // 所有牌種皆已有 AI 圖片，此函式保留作為 fallback
    return null;
}

/**
 * 取得牌的可讀名稱 (用於 debug)
 */
export function getTileName(info: TileInfo): string {
    switch (info.type) {
        case TileType.Wan: return `${info.value}萬`;
        case TileType.Tong: return `${info.value}筒`;
        case TileType.Tiao: return `${info.value}條`;
        case TileType.Wind: return WindNames[info.value] || "?風";
        case TileType.Dragon: return DragonNames[info.value] || "?元";
        case TileType.Flower: return FlowerNames[info.value] || "?花";
        default: return "未知";
    }
}
