package models

import (
	"testing"
)

func TestCalculateScore_PingHu(t *testing.T) {
	ctx := ScoringContext{
		GameType: GameType16,
		ClosedHand: []Tile{
			{Type: Wan, Value: 2}, {Type: Wan, Value: 3}, {Type: Wan, Value: 4},
			{Type: Tong, Value: 5}, {Type: Tong, Value: 6}, {Type: Tong, Value: 7},
			{Type: Tiao, Value: 1}, {Type: Tiao, Value: 2}, {Type: Tiao, Value: 3},
			{Type: Tiao, Value: 4}, {Type: Tiao, Value: 5}, {Type: Tiao, Value: 6},
			{Type: Tong, Value: 9},
		},
		WinningTile: Tile{Type: Tong, Value: 9},
		IsSelfDrawn: false,
		IsDealer:    false,
		Melds:       []Meld{},
		Flowers:     []Tile{},
	}

	res := CalculateScore(ctx)
	// 平胡(2) + 門清(1) = 3
	if res.TotalTai != 3 {
		t.Errorf("Expected TotalTai 3, got %d. Patterns: %v", res.TotalTai, res.Patterns)
	}
	if res.Patterns["平胡"] != 2 {
		t.Errorf("Expected 平胡 2, got %v", res.Patterns)
	}
}

func TestCalculateScore_AllHonors(t *testing.T) {
	ctx := ScoringContext{
		GameType: GameType16,
		ClosedHand: []Tile{
			{Type: Wind, Value: 1}, {Type: Wind, Value: 1}, {Type: Wind, Value: 1},
			{Type: Wind, Value: 2}, {Type: Wind, Value: 2}, {Type: Wind, Value: 2},
			{Type: Wind, Value: 3}, {Type: Wind, Value: 3}, {Type: Wind, Value: 3},
			{Type: Wind, Value: 4}, {Type: Wind, Value: 4}, {Type: Wind, Value: 4},
			{Type: Dragon, Value: 1},
		},
		WinningTile:    Tile{Type: Dragon, Value: 1},
		IsSelfDrawn:    false,
		Melds:          []Meld{},
		PrevailingWind: East,
		SeatWind:       East,
	}

	res := CalculateScore(ctx)
	if res.Patterns["字一色"] != 16 {
		t.Errorf("Expected 字一色 16, got %v", res.Patterns)
	}
	if res.Patterns["四暗刻"] != 5 {
		t.Errorf("Expected 四暗刻 5, got %v", res.Patterns)
	}
}

func TestCalculateScore_FullFlush(t *testing.T) {
	ctx := ScoringContext{
		GameType: GameType16,
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
		t.Errorf("Expected 清一色 8, got %v", res.Patterns)
	}
	if res.Patterns["自摸"] != 1 {
		t.Errorf("Expected 自摸 1, got %v", res.Patterns)
	}
}

func TestCalculateScore_Dragons(t *testing.T) {
	ctx := ScoringContext{
		GameType: GameType16,
		ClosedHand: []Tile{
			{Type: Dragon, Value: 1}, {Type: Dragon, Value: 1}, {Type: Dragon, Value: 1},
			{Type: Dragon, Value: 2}, {Type: Dragon, Value: 2}, {Type: Dragon, Value: 2},
			{Type: Dragon, Value: 3}, {Type: Dragon, Value: 3}, {Type: Dragon, Value: 3},
			{Type: Wan, Value: 1}, {Type: Wan, Value: 2}, {Type: Wan, Value: 3},
			{Type: Wan, Value: 5},
		},
		WinningTile: Tile{Type: Wan, Value: 5},
		IsSelfDrawn: false,
		Melds:       []Meld{},
	}

	res := CalculateScore(ctx)
	if res.Patterns["中"] != 1 {
		t.Errorf("Expected 中 1, got %v", res.Patterns)
	}
	if res.Patterns["發"] != 1 {
		t.Errorf("Expected 發 1, got %v", res.Patterns)
	}
	if res.Patterns["白"] != 1 {
		t.Errorf("Expected 白 1, got %v", res.Patterns)
	}
	if res.Patterns["大三元"] != 8 {
		t.Errorf("Expected 大三元 8, got %v", res.Patterns)
	}
}

func TestCalculateScore_Straight(t *testing.T) {
	ctx := ScoringContext{
		GameType: GameType16,
		ClosedHand: []Tile{
			{Type: Wan, Value: 1}, {Type: Wan, Value: 2}, {Type: Wan, Value: 3},
			{Type: Wan, Value: 4}, {Type: Wan, Value: 5}, {Type: Wan, Value: 6},
			{Type: Wan, Value: 7}, {Type: Wan, Value: 8}, {Type: Wan, Value: 9},
			{Type: Tong, Value: 3}, {Type: Tong, Value: 4}, {Type: Tong, Value: 5},
			{Type: Tiao, Value: 7},
		},
		WinningTile: Tile{Type: Tiao, Value: 7},
		IsSelfDrawn: false,
		Melds:       []Meld{},
	}

	res := CalculateScore(ctx)
	if res.Patterns["一條龍"] != 2 {
		t.Errorf("Expected 一條龍 2, got %v", res.Patterns)
	}
}

func TestCalculateScore_MenQingYiMoSan(t *testing.T) {
	ctx := ScoringContext{
		GameType: GameType16,
		ClosedHand: []Tile{
			{Type: Wan, Value: 1}, {Type: Wan, Value: 2}, {Type: Wan, Value: 3},
			{Type: Tong, Value: 5}, {Type: Tong, Value: 6}, {Type: Tong, Value: 7},
			{Type: Tiao, Value: 1}, {Type: Tiao, Value: 2}, {Type: Tiao, Value: 3},
			{Type: Tiao, Value: 4}, {Type: Tiao, Value: 5}, {Type: Tiao, Value: 6},
			{Type: Tong, Value: 9},
		},
		WinningTile: Tile{Type: Tong, Value: 9},
		IsSelfDrawn: true,
		Melds:       []Meld{},
		Flowers:     []Tile{},
	}

	res := CalculateScore(ctx)
	// 門清(1) + 自摸(1) + 門清一摸三(3) = 5
	if res.Patterns["門清"] != 1 {
		t.Errorf("Expected 門清 1, got %v", res.Patterns)
	}
	if res.Patterns["自摸"] != 1 {
		t.Errorf("Expected 自摸 1, got %v", res.Patterns)
	}
	if res.Patterns["門清一摸三"] != 3 {
		t.Errorf("Expected 門清一摸三 3, got %v", res.Patterns)
	}
}

func TestCalculateScore_Flowers(t *testing.T) {
	ctx := ScoringContext{
		GameType: GameType16,
		ClosedHand: []Tile{
			{Type: Wan, Value: 1}, {Type: Wan, Value: 2}, {Type: Wan, Value: 3},
			{Type: Tong, Value: 5}, {Type: Tong, Value: 6}, {Type: Tong, Value: 7},
			{Type: Tiao, Value: 1}, {Type: Tiao, Value: 2}, {Type: Tiao, Value: 3},
			{Type: Tiao, Value: 4}, {Type: Tiao, Value: 5}, {Type: Tiao, Value: 6},
			{Type: Tong, Value: 9},
		},
		WinningTile: Tile{Type: Tong, Value: 9},
		IsSelfDrawn: false,
		Melds:       []Meld{},
		Flowers:     []Tile{{Type: Flower, Value: 1}, {Type: Flower, Value: 5}},
		SeatID:      1,
	}

	res := CalculateScore(ctx)
	if res.Patterns["正花"] != 2 {
		t.Errorf("Expected 正花 2, got %v", res.Patterns)
	}
}

func TestCalculateScore_WindTriplets(t *testing.T) {
	ctx := ScoringContext{
		GameType: GameType16,
		ClosedHand: []Tile{
			{Type: Wind, Value: 1}, {Type: Wind, Value: 1}, {Type: Wind, Value: 1},
			{Type: Wind, Value: 2}, {Type: Wind, Value: 2}, {Type: Wind, Value: 2},
			{Type: Wan, Value: 1}, {Type: Wan, Value: 2}, {Type: Wan, Value: 3},
			{Type: Tong, Value: 5}, {Type: Tong, Value: 6}, {Type: Tong, Value: 7},
			{Type: Tiao, Value: 9},
		},
		WinningTile:    Tile{Type: Tiao, Value: 9},
		IsSelfDrawn:    false,
		Melds:          []Meld{},
		PrevailingWind: East,
		SeatWind:       South,
	}

	res := CalculateScore(ctx)
	if res.Patterns["圈風刻"] != 1 {
		t.Errorf("Expected 圈風刻 1, got %v", res.Patterns)
	}
	if res.Patterns["門風刻"] != 1 {
		t.Errorf("Expected 門風刻 1, got %v", res.Patterns)
	}
}

func TestCalculateScore_TianHu(t *testing.T) {
	ctx := ScoringContext{
		GameType: GameType16,
		ClosedHand: []Tile{
			{Type: Wan, Value: 1}, {Type: Wan, Value: 2}, {Type: Wan, Value: 3},
			{Type: Tong, Value: 5}, {Type: Tong, Value: 6}, {Type: Tong, Value: 7},
			{Type: Tiao, Value: 1}, {Type: Tiao, Value: 2}, {Type: Tiao, Value: 3},
			{Type: Tiao, Value: 4}, {Type: Tiao, Value: 5}, {Type: Tiao, Value: 6},
			{Type: Tong, Value: 9},
		},
		WinningTile: Tile{Type: Tong, Value: 9},
		IsSelfDrawn: true,
		IsDealer:    true,
		IsFirstDraw: true,
		Melds:       []Meld{},
	}

	res := CalculateScore(ctx)
	if res.Patterns["天胡"] != 16 {
		t.Errorf("Expected 天胡 16, got %v", res.Patterns)
	}
	if res.TotalTai != 16 {
		t.Errorf("Expected TotalTai 16, got %d", res.TotalTai)
	}
}

func TestCalculateScore_ConsecutiveDealer(t *testing.T) {
	ctx := ScoringContext{
		GameType: GameType16,
		ClosedHand: []Tile{
			{Type: Wan, Value: 1}, {Type: Wan, Value: 2}, {Type: Wan, Value: 3},
			{Type: Tong, Value: 5}, {Type: Tong, Value: 6}, {Type: Tong, Value: 7},
			{Type: Tiao, Value: 1}, {Type: Tiao, Value: 2}, {Type: Tiao, Value: 3},
			{Type: Tiao, Value: 4}, {Type: Tiao, Value: 5}, {Type: Tiao, Value: 6},
			{Type: Tong, Value: 9},
		},
		WinningTile:       Tile{Type: Tong, Value: 9},
		IsSelfDrawn:       false,
		IsDealer:          true,
		Melds:             []Meld{},
		ConsecutiveDealer: 2,
	}

	res := CalculateScore(ctx)
	if res.Patterns["莊家"] != 1 {
		t.Errorf("Expected 莊家 1, got %v", res.Patterns)
	}
	if res.Patterns["連莊"] != 2 {
		t.Errorf("Expected 連莊 2, got %v", res.Patterns)
	}
}
