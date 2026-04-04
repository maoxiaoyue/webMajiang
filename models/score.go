package models

// ScoreResult 儲存結算結果與所有達成的牌型名稱、總台數
type ScoreResult struct {
	TotalTai int            `json:"total_tai"`
	Patterns map[string]int `json:"patterns"` // ex: {"自摸": 1, "平胡": 2}
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
	GameType    GameType // 13 或 16 張
	ClosedHand  []Tile   // 玩家手中的暗牌 (不包含剛胡的那張)
	Melds       []Meld   // 玩家打出的明牌/暗槓
	WinningTile Tile     // 使玩家胡牌的那張牌 (自摸或別人打的)
	Flowers     []Tile   // 抽到的花牌

	// 狀態旗標
	IsSelfDrawn bool // 是否為自摸
	IsDealer    bool // 是否為莊家
	IsFirstDraw bool // 是否為第一次摸牌 (天胡/地胡判斷用)

	// 特殊狀態
	IsRobbingKong bool // 搶槓
	IsAfterKong   bool // 槓上開花
	IsLastTile    bool // 海底撈月 (最後一張牌)

	// 風位資訊
	PrevailingWind WindPosition // 圈風 (1=東, 2=南, 3=西, 4=北)
	SeatWind       WindPosition // 門風/自風

	// 連莊
	ConsecutiveDealer int // 連莊次數 (0=無連莊, 1=連一, ...)

	// 座位編號 (花牌正花判斷用，1-4)
	SeatID int
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

// FindAllPartitions 給定手牌 (長度必為 3N+2)，回傳所有可能合法組成
func FindAllPartitions(hand []Tile) []Partition {
	counts := make([]int, 34)
	var tilesByIdx [34]Tile
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
		allZero := true
		for i := 0; i < 34; i++ {
			if counts[i] > 0 {
				allZero = false
				startIdx = i
				break
			}
		}

		if allZero {
			pCopy := Partition{Combos: make([]TileCombo, len(currentPartition.Combos))}
			copy(pCopy.Combos, currentPartition.Combos)
			results = append(results, pCopy)
			return
		}

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

		if startIdx < 27 && startIdx%9 <= 6 && counts[startIdx] > 0 && counts[startIdx+1] > 0 && counts[startIdx+2] > 0 {
			counts[startIdx]--
			counts[startIdx+1]--
			counts[startIdx+2]--

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

	for i := 0; i < 34; i++ {
		if counts[i] >= 2 {
			counts[i] -= 2
			initialPartition := Partition{
				Combos: []TileCombo{
					{Type: ComboPair, Tiles: []Tile{tilesByIdx[i], tilesByIdx[i]}},
				},
			}
			backtrack(0, initialPartition)
			counts[i] += 2
		}
	}

	return results
}

// CalculateScore 根據情境計算手牌能獲得的最大台數
func CalculateScore(ctx ScoringContext) ScoreResult {
	// 13 張麻將使用獨立的計分系統
	if ctx.GameType == GameType13 {
		return CalculateScore13(ctx)
	}

	// 16 張麻將（原有邏輯）
	fullHand := append([]Tile{}, ctx.ClosedHand...)
	fullHand = append(fullHand, ctx.WinningTile)

	partitions := FindAllPartitions(fullHand)
	if len(partitions) == 0 {
		return NewScoreResult()
	}

	bestScore := ScoreResult{TotalTai: -1}
	for _, p := range partitions {
		score := evaluatePartition(ctx, p, fullHand)
		if score.TotalTai > bestScore.TotalTai {
			bestScore = score
		}
	}

	return bestScore
}

// collectAllTiles 收集手牌 + 副露的所有牌
func collectAllTiles(fullHand []Tile, melds []Meld) []Tile {
	allTiles := append([]Tile{}, fullHand...)
	for _, m := range melds {
		allTiles = append(allTiles, m.Tiles...)
	}
	return allTiles
}

// collectAllTriplets 從 partition 和副露收集所有刻子/槓子
func collectAllTriplets(p Partition, melds []Meld) []TileCombo {
	var triplets []TileCombo
	for _, combo := range p.Combos {
		if combo.Type == ComboTriplet {
			triplets = append(triplets, combo)
		}
	}
	for _, m := range melds {
		if m.Type == MeldTypePong || m.Type == MeldTypeKong || m.Type == MeldTypeHiddenKong || m.Type == MeldTypeAddKong {
			triplets = append(triplets, TileCombo{Type: ComboTriplet, Tiles: m.Tiles})
		}
	}
	return triplets
}

// isConcealed 判斷是否門清
func isConcealed(melds []Meld) bool {
	for _, m := range melds {
		if m.Type != MeldTypeHiddenKong {
			return false
		}
	}
	return true
}

func evaluatePartition(ctx ScoringContext, p Partition, fullHand []Tile) ScoreResult {
	res := NewScoreResult()
	is16 := ctx.GameType == GameType16

	allTiles := collectAllTiles(fullHand, ctx.Melds)
	concealed := isConcealed(ctx.Melds)

	// ========================================
	// 特殊役滿（最高優先，直接回傳）
	// ========================================

	// 天胡 (16台) — 莊家第一次摸牌即胡 (僅16張)
	if is16 && ctx.IsDealer && ctx.IsFirstDraw && ctx.IsSelfDrawn {
		res.AddPattern("天胡", 16)
		return res
	}

	// 地胡 (16台) — 非莊家第一次摸牌即胡 (僅16張)
	if is16 && !ctx.IsDealer && ctx.IsFirstDraw && ctx.IsSelfDrawn {
		res.AddPattern("地胡", 16)
		return res
	}

	// ========================================
	// 1. 基本台：莊家、自摸、門清
	// ========================================

	// 莊家 (1台)
	if ctx.IsDealer {
		res.AddPattern("莊家", 1)
	}

	// 連莊 (每連一次加 1 台，僅16張)
	if is16 && ctx.ConsecutiveDealer > 0 {
		res.AddPattern("連莊", ctx.ConsecutiveDealer)
	}

	// 自摸 (1台)
	if ctx.IsSelfDrawn {
		res.AddPattern("自摸", 1)
	}

	// 門清 (1台)
	if concealed {
		res.AddPattern("門清", 1)
	}

	// 門清一摸三 (僅16張): 門清 + 自摸 → 額外加 3 台 (三家都要付)
	// 注意：門清(1) + 自摸(1) 已加，此處再加 3 台
	if is16 && concealed && ctx.IsSelfDrawn {
		res.AddPattern("門清一摸三", 3)
	}

	// 全求人 (2台): 手上全部吃碰槓出去，只剩一張單吊，由別人打的
	if !ctx.IsSelfDrawn && !concealed {
		closedCount := len(ctx.ClosedHand) // 不含 WinningTile
		if closedCount == 0 {
			// 手上 0 張暗牌，全靠副露 + 最後一張別人打的
			res.AddPattern("全求人", 2)
		}
	}

	// ========================================
	// 2. 特殊條件台
	// ========================================

	// 搶槓 (1台)
	if ctx.IsRobbingKong {
		res.AddPattern("搶槓", 1)
	}

	// 槓上開花 (1台): 槓牌後從嶺上摸牌自摸
	if ctx.IsAfterKong && ctx.IsSelfDrawn {
		res.AddPattern("槓上開花", 1)
	}

	// 海底撈月 (1台): 最後一張牌自摸
	if ctx.IsLastTile && ctx.IsSelfDrawn {
		res.AddPattern("海底撈月", 1)
	}

	// ========================================
	// 3. 花牌計算 (僅16張)
	// ========================================
	if is16 && len(ctx.Flowers) > 0 {
		evalFlowers(ctx, &res)
	}

	// ========================================
	// 4. 三元牌：中/發/白 各刻子 1 台
	// ========================================
	allTriplets := collectAllTriplets(p, ctx.Melds)
	evalDragons(allTriplets, &res)

	// ========================================
	// 5. 圈風刻 / 門風刻
	// ========================================
	evalWindTriplets(allTriplets, ctx.PrevailingWind, ctx.SeatWind, &res)

	// ========================================
	// 6. 暗刻統計
	// ========================================
	evalConcealedTriplets(p, ctx, &res)

	// ========================================
	// 7. 對對胡 (碰碰胡) — 全刻子無順子
	// ========================================
	evalAllTriplets(p, ctx.Melds, &res)

	// ========================================
	// 8. 平胡
	// ========================================
	evalPingHu(p, ctx, &res)

	// ========================================
	// 9. 一條龍 (同花色 1-9 順子)
	// ========================================
	evalStraight(p, ctx.Melds, &res)

	// ========================================
	// 10. 花色判斷 (清一色/混一色/字一色)
	// ========================================
	evalFlush(allTiles, &res)

	// ========================================
	// 11. 大三元/小三元/大四喜/小四喜
	// ========================================
	evalSpecialHands(allTriplets, p, &res)

	return res
}

// ========================================
// 各牌型判斷子函式
// ========================================

// evalFlowers 花牌台數計算 (僅16張)
// 正花: 座位1=花1(梅/春), 座位2=花2(蘭/夏), 座位3=花3(竹/秋), 座位4=花4(菊/冬)
// 每朵正花 1 台，全花 (8 朵) 8 台，七搶一 8 台
func evalFlowers(ctx ScoringContext, res *ScoreResult) {
	if len(ctx.Flowers) == 0 {
		return
	}

	// 七搶一 (8台): 玩家有 7 朵花，搶到第 8 朵
	if len(ctx.Flowers) >= 7 && ctx.IsRobbingKong {
		res.AddPattern("七搶一", 8)
		return
	}

	// 八仙過海 (8台): 集滿 8 朵花牌
	if len(ctx.Flowers) >= 8 {
		res.AddPattern("八仙過海", 8)
		return
	}

	flowerTai := 0
	for _, f := range ctx.Flowers {
		// 正花判斷: 花牌 Value 1-4 (梅蘭竹菊) 對應座位 1-4
		//           花牌 Value 5-8 (春夏秋冬) 對應座位 1-4 (value-4)
		matchSeat := 0
		if f.Value >= 1 && f.Value <= 4 {
			matchSeat = f.Value
		} else if f.Value >= 5 && f.Value <= 8 {
			matchSeat = f.Value - 4
		}
		if matchSeat == ctx.SeatID {
			flowerTai++ // 正花 +1 台
		}
	}
	if flowerTai > 0 {
		res.AddPattern("正花", flowerTai)
	}
}

// evalDragons 三元牌刻子: 中(1台) / 發(1台) / 白(1台)
func evalDragons(allTriplets []TileCombo, res *ScoreResult) {
	for _, trip := range allTriplets {
		if len(trip.Tiles) > 0 && trip.Tiles[0].Type == Dragon {
			switch trip.Tiles[0].Value {
			case 1:
				res.AddPattern("中", 1)
			case 2:
				res.AddPattern("發", 1)
			case 3:
				res.AddPattern("白", 1)
			}
		}
	}
}

// evalWindTriplets 圈風刻 / 門風刻
func evalWindTriplets(allTriplets []TileCombo, prevailing, seat WindPosition, res *ScoreResult) {
	for _, trip := range allTriplets {
		if len(trip.Tiles) > 0 && trip.Tiles[0].Type == Wind {
			windVal := WindPosition(trip.Tiles[0].Value)
			if windVal == prevailing {
				res.AddPattern("圈風刻", 1)
			}
			if windVal == seat {
				res.AddPattern("門風刻", 1)
			}
		}
	}
}

// evalConcealedTriplets 暗刻統計 (三暗刻 2台 / 四暗刻 5台 / 五暗刻 8台)
func evalConcealedTriplets(p Partition, ctx ScoringContext, res *ScoreResult) {
	count := 0

	for _, combo := range p.Combos {
		if combo.Type == ComboTriplet {
			containsWinningTile := false
			for _, t := range combo.Tiles {
				if t.Type == ctx.WinningTile.Type && t.Value == ctx.WinningTile.Value {
					containsWinningTile = true
					break
				}
			}
			if containsWinningTile && !ctx.IsSelfDrawn {
				// 別人打的完成刻子 → 明刻
			} else {
				count++
			}
		}
	}

	for _, m := range ctx.Melds {
		if m.Type == MeldTypeHiddenKong {
			count++
		}
	}

	if count >= 5 {
		res.AddPattern("五暗刻", 8)
	} else if count >= 4 {
		res.AddPattern("四暗刻", 5)
	} else if count >= 3 {
		res.AddPattern("三暗刻", 2)
	}
}

// evalAllTriplets 對對胡 (碰碰胡): 4 組刻子 + 1 雀頭，無順子
func evalAllTriplets(p Partition, melds []Meld, res *ScoreResult) {
	seqCount := 0
	tripCount := 0
	for _, combo := range p.Combos {
		if combo.Type == ComboSequence {
			seqCount++
		}
		if combo.Type == ComboTriplet {
			tripCount++
		}
	}
	for _, m := range melds {
		if m.Type == MeldTypeChow {
			seqCount++
		} else if m.Type == MeldTypePong || m.Type == MeldTypeKong || m.Type == MeldTypeHiddenKong || m.Type == MeldTypeAddKong {
			tripCount++
		}
	}
	if seqCount == 0 && tripCount >= 4 {
		res.AddPattern("對對胡", 4)
	}
}

// evalPingHu 平胡: 無字、無花、無刻子、只有順子+雀頭、非自摸
func evalPingHu(p Partition, ctx ScoringContext, res *ScoreResult) {
	if ctx.IsSelfDrawn || len(ctx.Flowers) > 0 {
		return
	}

	// 副露只能是吃
	for _, m := range ctx.Melds {
		if m.Type != MeldTypeChow {
			return
		}
	}

	for _, combo := range p.Combos {
		if combo.Type == ComboTriplet {
			return
		}
		for _, t := range combo.Tiles {
			if t.Type == Wind || t.Type == Dragon {
				return
			}
		}
	}

	res.AddPattern("平胡", 2)
}

// evalStraight 一條龍: 同花色 123 + 456 + 789
func evalStraight(p Partition, melds []Meld, res *ScoreResult) {
	// 收集所有順子
	type seqKey struct {
		suit  TileType
		start int // 起始值
	}
	seqs := make(map[seqKey]bool)

	for _, combo := range p.Combos {
		if combo.Type == ComboSequence && len(combo.Tiles) > 0 {
			seqs[seqKey{combo.Tiles[0].Type, combo.Tiles[0].Value}] = true
		}
	}
	for _, m := range melds {
		if m.Type == MeldTypeChow && len(m.Tiles) > 0 {
			// 找最小值
			minVal := m.Tiles[0].Value
			for _, t := range m.Tiles[1:] {
				if t.Value < minVal {
					minVal = t.Value
				}
			}
			seqs[seqKey{m.Tiles[0].Type, minVal}] = true
		}
	}

	// 檢查同花色是否有 1, 4, 7 起始的順子
	for _, suit := range []TileType{Wan, Tong, Tiao} {
		if seqs[seqKey{suit, 1}] && seqs[seqKey{suit, 4}] && seqs[seqKey{suit, 7}] {
			res.AddPattern("一條龍", 2)
			return
		}
	}
}

// evalFlush 花色判斷
func evalFlush(allTiles []Tile, res *ScoreResult) {
	suitCount := make(map[TileType]int)
	hasHonors := false

	for _, t := range allTiles {
		if t.Type == Wind || t.Type == Dragon {
			hasHonors = true
		} else if t.Type != Flower {
			suitCount[t.Type]++
		}
	}

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
}

// evalSpecialHands 大三元/小三元/大四喜/小四喜
func evalSpecialHands(allTriplets []TileCombo, p Partition, res *ScoreResult) {
	// 統計三元牌刻子和對子
	dragonTrips := 0
	dragonPairs := 0
	for _, trip := range allTriplets {
		if len(trip.Tiles) > 0 && trip.Tiles[0].Type == Dragon {
			dragonTrips++
		}
	}
	for _, combo := range p.Combos {
		if combo.Type == ComboPair && len(combo.Tiles) > 0 && combo.Tiles[0].Type == Dragon {
			dragonPairs++
		}
	}

	// 大三元 (8台): 三組三元牌刻子
	if dragonTrips >= 3 {
		res.AddPattern("大三元", 8)
	} else if dragonTrips == 2 && dragonPairs >= 1 {
		// 小三元 (4台): 兩組三元牌刻子 + 一組三元牌對子
		res.AddPattern("小三元", 4)
	}

	// 統計風牌刻子和對子
	windTrips := 0
	windPairs := 0
	for _, trip := range allTriplets {
		if len(trip.Tiles) > 0 && trip.Tiles[0].Type == Wind {
			windTrips++
		}
	}
	for _, combo := range p.Combos {
		if combo.Type == ComboPair && len(combo.Tiles) > 0 && combo.Tiles[0].Type == Wind {
			windPairs++
		}
	}

	// 大四喜 (16台): 四組風牌刻子
	if windTrips >= 4 {
		res.AddPattern("大四喜", 16)
	} else if windTrips == 3 && windPairs >= 1 {
		// 小四喜 (8台): 三組風牌刻子 + 一組風牌對子
		res.AddPattern("小四喜", 8)
	}
}
