package models

// ScoreResult 儲存結算結果與所有達成的牌型名稱、總台數
type ScoreResult struct {
	TotalTai int
	Patterns map[string]int // ex: {"自摸": 1, "平胡": 2}
}

func NewScoreResult() ScoreResult {
	return ScoreResult{
		TotalTai: 0,
		Patterns: make(map[string]int),
	}
}

func (s *ScoreResult) AddPattern(name string, tai int) {
	if tai > 0 {
		s.Patterns[name] = tai
		s.TotalTai += tai
	}
}

// ScoringContext 計算台數所需的完整狀態
type ScoringContext struct {
	ClosedHand  []Tile // 玩家手中的暗牌 (不包含剛胡的那張)
	Melds       []Meld // 玩家打出的明牌/暗槓
	WinningTile Tile   // 使玩家胡牌的那張牌 (自摸或別人打的)
	IsSelfDrawn bool   // 是否為自摸
	IsDealer    bool   // 是否為莊家
	Flowers     []Tile // 抽到的花牌
}

// TileCombo 代表一組已解構的牌 (順子, 刻子, 雀頭)
type ComboType int

const (
	ComboPair     ComboType = 1 // 雀頭 (對子)
	ComboSequence ComboType = 2 // 順子
	ComboTriplet  ComboType = 3 // 刻子
)

type TileCombo struct {
	Type  ComboType
	Tiles []Tile
}

// Partition 代表一種手牌的合法拆解方式 (包含各組 combo)
type Partition struct {
	Combos []TileCombo
}

// FindAllPartitions 給定手牌 (長度必為 3N+2)，回傳所有可能合法組成 (包含 1 個 Pair 與多個 Triplets/Sequences) 的拆解方案。
// 若無法胡牌則回傳空陣列。
func FindAllPartitions(hand []Tile) []Partition {
	counts := make([]int, 34)
	var tilesByIdx [34]Tile // 用來還原 tile info
	total := 0

	for _, t := range hand {
		idx := t.ToIndex()
		if idx != -1 {
			counts[idx]++
			tilesByIdx[idx] = t
			total++
		}
	}

	if total%3 != 2 {
		return nil
	}

	var results []Partition

	var backtrack func(startIdx int, currentPartition Partition)
	backtrack = func(startIdx int, currentPartition Partition) {
		// 檢查是否已全部分解完畢
		allZero := true
		for i := 0; i < 34; i++ {
			if counts[i] > 0 {
				allZero = false
				startIdx = i // 找到第一個非 0 的作為起點
				break
			}
		}

		if allZero {
			// 如果已經分解完所有牌，這就是一組有效的 Partition，加入結果陣列中
			// 複製 slice 以免受到後續修改影響
			pCopy := Partition{Combos: make([]TileCombo, len(currentPartition.Combos))}
			copy(pCopy.Combos, currentPartition.Combos)
			results = append(results, pCopy)
			return
		}

		// 嘗試組成刻子 (Triplet)
		if counts[startIdx] >= 3 {
			counts[startIdx] -= 3
			newCombo := TileCombo{
				Type:  ComboTriplet,
				Tiles: []Tile{tilesByIdx[startIdx], tilesByIdx[startIdx], tilesByIdx[startIdx]},
			}
			currentPartition.Combos = append(currentPartition.Combos, newCombo)
			backtrack(startIdx, currentPartition)
			currentPartition.Combos = currentPartition.Combos[:len(currentPartition.Combos)-1]
			counts[startIdx] += 3
		}

		// 嘗試組成順子 (Sequence) - 僅限萬、筒、條 (idx 0~26)
		if startIdx < 27 && startIdx%9 <= 6 && counts[startIdx] > 0 && counts[startIdx+1] > 0 && counts[startIdx+2] > 0 {
			counts[startIdx]--
			counts[startIdx+1]--
			counts[startIdx+2]--

			// 構造順子的 Tiles (Value +0, +1, +2)
			newCombo := TileCombo{
				Type: ComboSequence,
				Tiles: []Tile{
					{Type: tilesByIdx[startIdx].Type, Value: tilesByIdx[startIdx].Value},
					{Type: tilesByIdx[startIdx].Type, Value: tilesByIdx[startIdx].Value + 1},
					{Type: tilesByIdx[startIdx].Type, Value: tilesByIdx[startIdx].Value + 2},
				},
			}
			currentPartition.Combos = append(currentPartition.Combos, newCombo)
			backtrack(startIdx, currentPartition)
			currentPartition.Combos = currentPartition.Combos[:len(currentPartition.Combos)-1]

			counts[startIdx]++
			counts[startIdx+1]++
			counts[startIdx+2]++
		}
	}

	// 1. 先抽取一個雀頭 (Pair)
	for i := 0; i < 34; i++ {
		if counts[i] >= 2 {
			counts[i] -= 2

			initialPartition := Partition{
				Combos: []TileCombo{
					{
						Type:  ComboPair,
						Tiles: []Tile{tilesByIdx[i], tilesByIdx[i]},
					},
				},
			}

			// 進入回溯尋找剩下牌的組合
			backtrack(0, initialPartition)

			counts[i] += 2
		}
	}

	return results
}

// CalculateScore 根據情境計算手牌能獲得的最大台數
func CalculateScore(ctx ScoringContext) ScoreResult {
	// 組合出完整的手牌 (包含胡的那張牌)
	fullHand := append([]Tile{}, ctx.ClosedHand...)
	fullHand = append(fullHand, ctx.WinningTile)

	partitions := FindAllPartitions(fullHand)
	if len(partitions) == 0 {
		return NewScoreResult() // 詐胡或未達成基本胡牌條件 (理論上不該發生，前置會有 CanHu 擋住)
	}

	bestScore := ScoreResult{TotalTai: -1}

	// 遍歷所有可能的拆解方式，取台數最高者
	for _, p := range partitions {
		score := evaluatePartition(ctx, p, fullHand)
		if score.TotalTai > bestScore.TotalTai {
			bestScore = score
		}
	}

	return bestScore
}

func evaluatePartition(ctx ScoringContext, p Partition, fullHand []Tile) ScoreResult {
	res := NewScoreResult()

	// 1. 各身分與狀態基本台
	if ctx.IsDealer {
		res.AddPattern("莊家", 1)
	}
	if ctx.IsSelfDrawn {
		res.AddPattern("自摸", 1)
	}

	// 門清 (沒有非暗槓的吃碰槓)
	isConcealed := true
	for _, m := range ctx.Melds {
		if m.Type != MeldTypeHiddenKong {
			isConcealed = false
			break
		}
	}
	if isConcealed {
		res.AddPattern("門清", 1)
	}

	// 2. 統計暗刻數量 (三/四/五暗刻)
	// 定義：暗牌區的刻子 + 暗槓。如果胡的那張牌剛好完成某個刻子，且不是自摸，該刻子算明刻。
	concealedTripletsCount := 0

	// 計算從 partition 來的刻子
	for _, combo := range p.Combos {
		if combo.Type == ComboTriplet {
			// 判斷這個刻子是否包含胡的那張牌，且不是自摸
			containsWinningTile := false
			for _, t := range combo.Tiles {
				if t.Type == ctx.WinningTile.Type && t.Value == ctx.WinningTile.Value {
					containsWinningTile = true
					break
				}
			}

			if containsWinningTile && !ctx.IsSelfDrawn {
				// 別人打的，算明刻
			} else {
				concealedTripletsCount++
			}
		}
	}

	// 加上暗槓數量
	for _, m := range ctx.Melds {
		if m.Type == MeldTypeHiddenKong {
			concealedTripletsCount++
		}
	}

	if concealedTripletsCount == 5 {
		res.AddPattern("五暗刻", 8)
	} else if concealedTripletsCount == 4 {
		res.AddPattern("四暗刻", 4)
	} else if concealedTripletsCount == 3 {
		res.AddPattern("三暗刻", 2)
	}

	// 3. 判斷平胡
	// 條件：無字、無花、無刻子 (只有順子+雀頭)、不含任何副露(除吃以外)、非自摸
	// 備註：嚴格的平胡甚至可能要求雀頭不能是字牌、聽牌型態為雙頭等，此處採寬鬆/基本判定
	if !ctx.IsSelfDrawn && len(ctx.Flowers) == 0 {
		hasOnlyChow := true
		for _, m := range ctx.Melds {
			if m.Type != MeldTypeChow {
				hasOnlyChow = false
				break
			}
		}

		if hasOnlyChow {
			hasTripletsOrHonors := false
			for _, combo := range p.Combos {
				if combo.Type == ComboTriplet {
					hasTripletsOrHonors = true
					break
				}
				for _, t := range combo.Tiles {
					if t.Type == Wind || t.Type == Dragon {
						hasTripletsOrHonors = true
						break
					}
				}
			}
			if !hasTripletsOrHonors {
				res.AddPattern("平胡", 2)
			}
		}
	}

	// 4. 花色判斷 (清一色 8 / 混一色 4 / 字一色 16)
	// 統計所有牌 (包含副露) 的萬、筒、條、字牌數量
	suitCount := make(map[TileType]int)

	allTiles := append([]Tile{}, fullHand...)
	for _, m := range ctx.Melds {
		allTiles = append(allTiles, m.Tiles...)
	}

	hasHonors := false
	for _, t := range allTiles {
		if t.Type == Wind || t.Type == Dragon {
			hasHonors = true
		} else {
			suitCount[t.Type]++
		}
	}

	// 計算出現了幾種基本花色
	kindsOfSuits := 0
	for _, count := range suitCount {
		if count > 0 {
			kindsOfSuits++
		}
	}

	if kindsOfSuits == 0 && hasHonors {
		res.AddPattern("字一色", 16)
	} else if kindsOfSuits == 1 && !hasHonors {
		res.AddPattern("清一色", 8)
	} else if kindsOfSuits == 1 && hasHonors {
		res.AddPattern("混一色", 4)
	}

	return res
}
