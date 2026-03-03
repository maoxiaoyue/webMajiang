package models

import (
	"testing"
)

func TestCalculateScore_PingHu(t *testing.T) {
	// 平胡測試: 2萬3萬4萬, 5筒6筒7筒, 1條2條3條, 4條5條6條, 9筒9筒
	ctx := ScoringContext{
		ClosedHand: []Tile{
			{Type: Wan, Value: 2}, {Type: Wan, Value: 3}, {Type: Wan, Value: 4},
			{Type: Tong, Value: 5}, {Type: Tong, Value: 6}, {Type: Tong, Value: 7},
			{Type: Tiao, Value: 1}, {Type: Tiao, Value: 2}, {Type: Tiao, Value: 3},
			{Type: Tiao, Value: 4}, {Type: Tiao, Value: 5}, {Type: Tiao, Value: 6},
			{Type: Tong, Value: 9},
		},
		WinningTile: Tile{Type: Tong, Value: 9}, // 湊成對子
		IsSelfDrawn: false,
		IsDealer:    false,
		Melds:       []Meld{}, // 門清
		Flowers:     []Tile{},
	}

	res := CalculateScore(ctx)

	// 平胡 (2), 門清 (1) => 3台
	if res.TotalTai != 3 {
		t.Errorf("Expected TotalTai to be 3 (PingHu 2 + Concealed 1), got %d. Patterns: %v", res.TotalTai, res.Patterns)
	}
	if res.Patterns["平胡"] != 2 {
		t.Errorf("Expected PingHu 2, got %v", res.Patterns)
	}
}

func TestCalculateScore_AllHonors(t *testing.T) {
	// 字一色: 東東東 南南南 西西西 北北北 中中
	ctx := ScoringContext{
		ClosedHand: []Tile{
			{Type: Wind, Value: 1}, {Type: Wind, Value: 1}, {Type: Wind, Value: 1},
			{Type: Wind, Value: 2}, {Type: Wind, Value: 2}, {Type: Wind, Value: 2},
			{Type: Wind, Value: 3}, {Type: Wind, Value: 3}, {Type: Wind, Value: 3},
			{Type: Wind, Value: 4}, {Type: Wind, Value: 4}, {Type: Wind, Value: 4},
			{Type: Dragon, Value: 1},
		},
		WinningTile: Tile{Type: Dragon, Value: 1},
		IsSelfDrawn: false,
		Melds:       []Meld{},
	}

	res := CalculateScore(ctx)

	// 字一色(16), 門清(1), 四暗刻(4) = 21 (若嚴格算可能是 21 或更多組合，我們目前只要確認字一色有出現)
	if res.Patterns["字一色"] != 16 {
		t.Errorf("Expected All Honors 16, got %v", res.Patterns)
	}
	if res.Patterns["四暗刻"] != 4 {
		t.Errorf("Expected Four Concealed Triplets 4, got %v", res.Patterns)
	}
}

func TestCalculateScore_FullFlush(t *testing.T) {
	// 清一色: 全萬
	ctx := ScoringContext{
		ClosedHand: []Tile{
			{Type: Wan, Value: 1}, {Type: Wan, Value: 1}, {Type: Wan, Value: 1},
			{Type: Wan, Value: 2}, {Type: Wan, Value: 3}, {Type: Wan, Value: 4},
			{Type: Wan, Value: 5}, {Type: Wan, Value: 6}, {Type: Wan, Value: 7},
			{Type: Wan, Value: 8}, {Type: Wan, Value: 8}, {Type: Wan, Value: 8},
			{Type: Wan, Value: 9},
		},
		WinningTile: Tile{Type: Wan, Value: 9},
		IsSelfDrawn: true,
	}

	res := CalculateScore(ctx)
	if res.Patterns["清一色"] != 8 {
		t.Errorf("Expected Full Flush 8, got %v", res.Patterns)
	}
	if res.Patterns["自摸"] != 1 {
		t.Errorf("Expected Self-Drawn 1, got %v", res.Patterns)
	}
}
