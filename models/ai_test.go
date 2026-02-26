package models

import (
	"testing"
)

func TestGetBestDiscard(t *testing.T) {
	// 準備測試資料
	// 假設手牌為: 1萬, 2萬, 3萬 (順子), 5筒, 5筒, 5筒 (刻子), 東風, 南風 (孤立風牌), 8條, 9條 (搭子)
	hand := []Tile{
		{ID: 1, Type: Wan, Value: 1},
		{ID: 2, Type: Wan, Value: 2},
		{ID: 3, Type: Wan, Value: 3},

		{ID: 41, Type: Tong, Value: 5},
		{ID: 42, Type: Tong, Value: 5},
		{ID: 43, Type: Tong, Value: 5},

		{ID: 109, Type: Wind, Value: 1}, // 東 (無依無靠)
		{ID: 113, Type: Wind, Value: 2}, // 南 (無依無靠)

		{ID: 100, Type: Tiao, Value: 8},
		{ID: 105, Type: Tiao, Value: 9},
	}

	best := GetBestDiscard(hand)

	// 預期丟出的應該是「東」或「南」這兩張孤立風牌
	if !(best.Type == Wind && (best.Value == 1 || best.Value == 2)) {
		t.Errorf("Expected to discard East or South wind, got: %s", best.String())
	}
}

func TestGetBestDiscard_EdgeTile(t *testing.T) {
	// 沒有字牌的情況，孤立的邊張(1,9)應該要先被丟出
	// 2萬, 3萬, 4萬 (順子), 5萬, 5萬 (對子), 7筒, 8筒 (搭子), 1條 (孤立邊張), 5條 (孤立中張)
	hand := []Tile{
		{ID: 5, Type: Wan, Value: 2},
		{ID: 10, Type: Wan, Value: 3},
		{ID: 15, Type: Wan, Value: 4},

		{ID: 17, Type: Wan, Value: 5},
		{ID: 18, Type: Wan, Value: 5},

		{ID: 61, Type: Tong, Value: 7},
		{ID: 65, Type: Tong, Value: 8},

		{ID: 73, Type: Tiao, Value: 1}, // 孤立邊張
		{ID: 89, Type: Tiao, Value: 5}, // 孤立中張
	}

	best := GetBestDiscard(hand)

	// 預期丟出的應該是「1條」(孤立邊張優先度比中張更靠前)
	if !(best.Type == Tiao && best.Value == 1) {
		t.Errorf("Expected to discard isolated edge tile (1 Tiao), got: %s", best.String())
	}
}
