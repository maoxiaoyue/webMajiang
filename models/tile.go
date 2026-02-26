package models

import "fmt"

// TileType 定義牌的類型
type TileType int

const (
	Wan    TileType = iota // 萬
	Tong                   // 筒
	Tiao                   // 條
	Wind                   // 風 (東南西北)
	Dragon                 // 元 (中發白)
	Flower                 // 花
)

// TileType 的字串表示
func (t TileType) String() string {
	switch t {
	case Wan:
		return "萬"
	case Tong:
		return "筒"
	case Tiao:
		return "條"
	case Wind:
		return "風"
	case Dragon:
		return "元"
	case Flower:
		return "花"
	default:
		return "未知"
	}
}

// WindName 風牌名稱對照
var WindName = map[int]string{
	1: "東",
	2: "南",
	3: "西",
	4: "北",
}

// DragonName 元牌名稱對照
var DragonName = map[int]string{
	1: "中",
	2: "發",
	3: "白",
}

// FlowerName 花牌名稱對照
var FlowerName = map[int]string{
	1: "梅",
	2: "蘭",
	3: "竹",
	4: "菊",
	5: "春",
	6: "夏",
	7: "秋",
	8: "冬",
}

// Tile 牌結構體
type Tile struct {
	ID    int      `json:"id"`    // 唯一 ID (1-144)
	Type  TileType `json:"type"`  // 類型
	Value int      `json:"value"` // 數值 (萬筒條: 1-9，風: 1-4，元: 1-3，花: 1-8)
}

// GenerateAllTiles 產生並回傳所有 144 張牌的對應 Map，包含編號 1 到 144，符合排列順序：萬、筒、條、風、元、花。
func GenerateAllTiles() map[int]Tile {
	tiles := make(map[int]Tile)
	id := 1

	// 萬 (ID: 1-36，1-9萬每種4張)
	for value := 1; value <= 9; value++ {
		for i := 0; i < 4; i++ {
			tiles[id] = Tile{ID: id, Type: Wan, Value: value}
			id++
		}
	}

	// 筒 (ID: 37-72，1-9筒每種4張)
	for value := 1; value <= 9; value++ {
		for i := 0; i < 4; i++ {
			tiles[id] = Tile{ID: id, Type: Tong, Value: value}
			id++
		}
	}

	// 條 (ID: 73-108，1-9條每種4張)
	for value := 1; value <= 9; value++ {
		for i := 0; i < 4; i++ {
			tiles[id] = Tile{ID: id, Type: Tiao, Value: value}
			id++
		}
	}

	// 風 (ID: 109-124，東南西北每種4張)
	for value := 1; value <= 4; value++ {
		for i := 0; i < 4; i++ {
			tiles[id] = Tile{ID: id, Type: Wind, Value: value}
			id++
		}
	}

	// 元 (ID: 125-136，中發白每種4張)
	for value := 1; value <= 3; value++ {
		for i := 0; i < 4; i++ {
			tiles[id] = Tile{ID: id, Type: Dragon, Value: value}
			id++
		}
	}

	// 花 (ID: 137-144，梅蘭竹菊春夏秋冬各1張)
	for value := 1; value <= 8; value++ {
		tiles[id] = Tile{ID: id, Type: Flower, Value: value}
		id++
	}

	return tiles
}

// String 回傳牌的可讀名稱
func (t Tile) String() string {
	switch t.Type {
	case Wan:
		return fmt.Sprintf("%d萬", t.Value)
	case Tong:
		return fmt.Sprintf("%d筒", t.Value)
	case Tiao:
		return fmt.Sprintf("%d條", t.Value)
	case Wind:
		return WindName[t.Value]
	case Dragon:
		return DragonName[t.Value]
	case Flower:
		return FlowerName[t.Value]
	default:
		return "未知牌"
	}
}

// Player 玩家結構體
type Player struct {
	ID    int    `json:"id"`    // 玩家編號 (0-3)
	Name  string `json:"name"`  // 玩家名稱
	IsBot bool   `json:"isBot"` // 是否為 AI 自動玩家
	Hand  []Tile `json:"hand"`  // 手牌
}

// Game 遊戲狀態結構體
type Game struct {
	Deck    []Tile    `json:"deck"`    // 海底（牌堆）
	Players [4]Player `json:"players"` // 四位玩家
}
