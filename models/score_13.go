package models

// CalculateScore13 計算 13 張麻將的番數（國際標準 + 港式起胡 3 番）
func CalculateScore13(ctx ScoringContext) ScoreResult {
	fullHand := append([]Tile{}, ctx.ClosedHand...)
	fullHand = append(fullHand, ctx.WinningTile)

	// 先檢查十三么（不需要 partition）
	if isThirteenOrphans(fullHand) {
		res := NewScoreResult()
		res.AddPattern("十三么", 88)
		return res
	}

	// 檢查七對子（不需要 partition）
	if pairs := isSevenPairs(fullHand); pairs {
		res := NewScoreResult()
		res.AddPattern("七對子", 24)
		// 七對子可複合門清
		if isConcealed(ctx.Melds) {
			res.AddPattern("門清", 1)
		}
		// 七對子可複合斷么九
		if isTanyao13(fullHand, ctx.Melds) {
			res.AddPattern("斷么九", 1)
		}
		// 七對子可複合清一色
		evalFlush13(collectAllTiles(fullHand, ctx.Melds), &res)
		return res
	}

	// 標準 partition 解法
	partitions := FindAllPartitions(fullHand)
	if len(partitions) == 0 {
		return NewScoreResult()
	}

	bestScore := ScoreResult{TotalTai: -1}
	for _, p := range partitions {
		score := evaluatePartition13(ctx, p, fullHand)
		if score.TotalTai > bestScore.TotalTai {
			bestScore = score
		}
	}

	return bestScore
}

func evaluatePartition13(ctx ScoringContext, p Partition, fullHand []Tile) ScoreResult {
	res := NewScoreResult()

	allTiles := collectAllTiles(fullHand, ctx.Melds)
	concealed := isConcealed(ctx.Melds)
	allTriplets := collectAllTriplets(p, ctx.Melds)

	// ========================================
	// 役滿級（直接回傳）
	// ========================================

	// 天胡 (20番) — 莊家第一手自摸
	if ctx.IsDealer && ctx.IsFirstDraw && ctx.IsSelfDrawn {
		res.AddPattern("天胡", 20)
		return res
	}

	// 九連寶燈 (20番) — 門清同花色 1112345678999 + 任一同花
	if concealed && isNineGates(fullHand) {
		res.AddPattern("九連寶燈", 20)
		return res
	}

	// 大四喜 (40番) — 東南西北四組刻子
	windTrips, windPairs := countWindSets(allTriplets, p)
	if windTrips >= 4 {
		res.AddPattern("大四喜", 40)
		return res
	}

	// 四槓 (40番) — 四組槓子
	kongCount := countKongs(ctx.Melds)
	if kongCount >= 4 {
		res.AddPattern("四槓", 40)
		return res
	}

	// 小四喜 (20番) — 三組風刻 + 一組風對
	if windTrips == 3 && windPairs >= 1 {
		res.AddPattern("小四喜", 20)
		return res
	}

	// 大三元 (10番) — 中發白三組刻子
	dragonTrips, _ := countDragonSets(allTriplets, p)
	if dragonTrips >= 3 {
		res.AddPattern("大三元", 10)
		return res
	}

	// 地胡 (10番) — 非莊家第一手自摸
	if !ctx.IsDealer && ctx.IsFirstDraw && ctx.IsSelfDrawn {
		res.AddPattern("地胡", 10)
		return res
	}

	// 清一色 (10番)
	if isFullFlush(allTiles) {
		res.AddPattern("清一色", 10)
		return res
	}

	// 四暗刻 (10番)
	concealedTripCount := countConcealedTriplets13(p, ctx)
	if concealedTripCount >= 4 {
		res.AddPattern("四暗刻", 10)
		return res
	}

	// 十八羅漢 (10番) — 四組槓（已在上面處理過）

	// ========================================
	// 高番牌型（可累計）
	// ========================================

	// 小三元 (5番)
	_, dragonPairs := countDragonSets(allTriplets, p)
	if dragonTrips == 2 && dragonPairs >= 1 {
		res.AddPattern("小三元", 5)
	}

	// 清四碰/對對胡 (4番) — 全刻子無順子
	if isAllTriplets13(p, ctx.Melds) {
		res.AddPattern("對對胡", 4)
	}

	// 全求人 (3番) — 全副露 + 別人打的胡
	if !ctx.IsSelfDrawn && !concealed && len(ctx.ClosedHand) == 0 {
		res.AddPattern("全求人", 3)
	}

	// 碰碰胡 (3番) — 不含槓的全碰型
	// 注意：如果已計「對對胡」則不重複計「碰碰胡」
	if _, ok := res.Patterns["對對胡"]; !ok {
		if isPongPongHu(p, ctx.Melds) {
			res.AddPattern("碰碰胡", 3)
		}
	}

	// 一條龍 (3番) — 同花色 123+456+789
	if hasStraight(p, ctx.Melds) {
		res.AddPattern("一條龍", 3)
	}

	// 三相逢 (3番) — 三花色同數順子 (如萬123+筒123+條123)
	if hasTripleSameSequence(p, ctx.Melds) {
		res.AddPattern("三相逢", 3)
	}

	// 混帶么 (2番) — 每組都含么九或字牌
	if isMixedTerminals(p, ctx.Melds) {
		res.AddPattern("混帶么", 2)
	}

	// ========================================
	// 1 番牌型
	// ========================================

	// 平胡 (1番) — 四組順子 + 雀頭，無字牌刻子
	if isPingHu13(p, ctx) {
		res.AddPattern("平胡", 1)
	}

	// 斷么九 (1番) — 沒有 1、9、字牌
	if isTanyao13(allTiles, ctx.Melds) {
		res.AddPattern("斷么九", 1)
	}

	// 中發白刻子 各 1 番
	evalDragons13(allTriplets, &res)

	// 獨聽 (1番) — 只聽一張牌（單騎、邊張、嵌張）
	if isUniqueWait(ctx, p) {
		res.AddPattern("獨聽", 1)
	}

	// 門清 (1番)
	if concealed {
		res.AddPattern("門清", 1)
	}

	// 自摸 (1番)
	if ctx.IsSelfDrawn {
		res.AddPattern("自摸", 1)
	}

	// 海底撈月 (1番) — 最後一張牌
	if ctx.IsLastTile && ctx.IsSelfDrawn {
		res.AddPattern("海底撈月", 1)
	}

	// 槓上自摸 (1番)
	if ctx.IsAfterKong && ctx.IsSelfDrawn {
		res.AddPattern("槓上自摸", 1)
	}

	// 三暗刻 (1番)
	if concealedTripCount >= 3 && concealedTripCount < 4 {
		res.AddPattern("三暗刻", 1)
	}

	return res
}

// ========================================
// 13 張專用判斷函式
// ========================================

// isThirteenOrphans 十三么: 1萬9萬1筒9筒1條9條東南西北中發白 各一 + 其中一張成對
func isThirteenOrphans(hand []Tile) bool {
	if len(hand) != 14 {
		return false
	}

	required := map[int]bool{
		0: true, 8: true, // 1萬, 9萬
		9: true, 17: true, // 1筒, 9筒
		18: true, 26: true, // 1條, 9條
		27: true, 28: true, 29: true, 30: true, // 東南西北
		31: true, 32: true, 33: true, // 中發白
	}

	counts := make(map[int]int)
	for _, t := range hand {
		idx := t.ToIndex()
		if idx == -1 || !required[idx] {
			return false
		}
		counts[idx]++
	}

	if len(counts) != 13 {
		return false
	}

	hasPair := false
	for _, c := range counts {
		if c == 2 {
			hasPair = true
		} else if c != 1 {
			return false
		}
	}
	return hasPair
}

// isSevenPairs 七對子: 7 組對子
func isSevenPairs(hand []Tile) bool {
	if len(hand) != 14 {
		return false
	}
	counts := make(map[int]int)
	for _, t := range hand {
		idx := t.ToIndex()
		if idx == -1 {
			return false
		}
		counts[idx]++
	}
	if len(counts) != 7 {
		return false
	}
	for _, c := range counts {
		if c != 2 {
			return false
		}
	}
	return true
}

// isNineGates 九連寶燈: 門清同花色 1112345678999 + 任一同花
func isNineGates(hand []Tile) bool {
	if len(hand) != 14 {
		return false
	}

	// 確認全部同花色 (數牌)
	suit := hand[0].Type
	if suit != Wan && suit != Tong && suit != Tiao {
		return false
	}
	for _, t := range hand {
		if t.Type != suit {
			return false
		}
	}

	counts := make([]int, 10) // index 1-9
	for _, t := range hand {
		if t.Value < 1 || t.Value > 9 {
			return false
		}
		counts[t.Value]++
	}

	// 1112345678999 = 1:3, 2:1, 3:1, 4:1, 5:1, 6:1, 7:1, 8:1, 9:3 = 14
	// 但需要多一張任意同花，所以基本型是 3+1+1+1+1+1+1+1+3 = 14
	// 實際上是 1112345678999 + 任一 = 15 張...
	// 13 張麻將的九連寶燈是 14 張 (手牌13 + 胡牌1)
	// 基本型 1112345678999 = 14 張，其中某個值多一張
	base := [10]int{0, 3, 1, 1, 1, 1, 1, 1, 1, 3}
	for i := 1; i <= 9; i++ {
		counts[i] -= base[i]
	}
	extraCount := 0
	for i := 1; i <= 9; i++ {
		if counts[i] < 0 {
			return false
		}
		extraCount += counts[i]
	}
	return extraCount == 0
}

// countWindSets 統計風牌刻子和對子數
func countWindSets(allTriplets []TileCombo, p Partition) (trips, pairs int) {
	for _, trip := range allTriplets {
		if len(trip.Tiles) > 0 && trip.Tiles[0].Type == Wind {
			trips++
		}
	}
	for _, combo := range p.Combos {
		if combo.Type == ComboPair && len(combo.Tiles) > 0 && combo.Tiles[0].Type == Wind {
			pairs++
		}
	}
	return
}

// countDragonSets 統計三元牌刻子和對子數
func countDragonSets(allTriplets []TileCombo, p Partition) (trips, pairs int) {
	for _, trip := range allTriplets {
		if len(trip.Tiles) > 0 && trip.Tiles[0].Type == Dragon {
			trips++
		}
	}
	for _, combo := range p.Combos {
		if combo.Type == ComboPair && len(combo.Tiles) > 0 && combo.Tiles[0].Type == Dragon {
			pairs++
		}
	}
	return
}

// countKongs 統計槓子數量
func countKongs(melds []Meld) int {
	count := 0
	for _, m := range melds {
		if m.Type == MeldTypeKong || m.Type == MeldTypeHiddenKong || m.Type == MeldTypeAddKong {
			count++
		}
	}
	return count
}

// countConcealedTriplets13 統計暗刻數
func countConcealedTriplets13(p Partition, ctx ScoringContext) int {
	count := 0
	for _, combo := range p.Combos {
		if combo.Type == ComboTriplet {
			containsWin := false
			for _, t := range combo.Tiles {
				if t.Type == ctx.WinningTile.Type && t.Value == ctx.WinningTile.Value {
					containsWin = true
					break
				}
			}
			if containsWin && !ctx.IsSelfDrawn {
				continue // 放槍完成的刻子 = 明刻
			}
			count++
		}
	}
	for _, m := range ctx.Melds {
		if m.Type == MeldTypeHiddenKong {
			count++
		}
	}
	return count
}

// isAllTriplets13 對對胡（含槓）
func isAllTriplets13(p Partition, melds []Meld) bool {
	for _, combo := range p.Combos {
		if combo.Type == ComboSequence {
			return false
		}
	}
	for _, m := range melds {
		if m.Type == MeldTypeChow {
			return false
		}
	}
	return true
}

// isPongPongHu 碰碰胡（純碰，不含槓）
func isPongPongHu(p Partition, melds []Meld) bool {
	for _, combo := range p.Combos {
		if combo.Type == ComboSequence {
			return false
		}
	}
	for _, m := range melds {
		if m.Type == MeldTypeChow {
			return false
		}
		if m.Type == MeldTypeKong || m.Type == MeldTypeHiddenKong || m.Type == MeldTypeAddKong {
			return false
		}
	}
	return true
}

// isFullFlush 清一色: 只有一種花色數牌，無字牌
func isFullFlush(allTiles []Tile) bool {
	suit := TileType(-1)
	for _, t := range allTiles {
		if t.Type == Flower {
			continue
		}
		if t.Type == Wind || t.Type == Dragon {
			return false
		}
		if suit == TileType(-1) {
			suit = t.Type
		} else if t.Type != suit {
			return false
		}
	}
	return suit != TileType(-1)
}

// isTanyao13 斷么九: 沒有 1、9、字牌
func isTanyao13(allTiles []Tile, melds []Meld) bool {
	for _, t := range allTiles {
		if t.Type == Flower {
			continue
		}
		if t.Type == Wind || t.Type == Dragon {
			return false
		}
		if t.Value == 1 || t.Value == 9 {
			return false
		}
	}
	for _, m := range melds {
		for _, t := range m.Tiles {
			if t.Type == Wind || t.Type == Dragon {
				return false
			}
			if t.Value == 1 || t.Value == 9 {
				return false
			}
		}
	}
	return true
}

// isPingHu13 平胡: 四組順子 + 雀頭（非字牌雀頭）
func isPingHu13(p Partition, ctx ScoringContext) bool {
	for _, combo := range p.Combos {
		if combo.Type == ComboTriplet {
			return false
		}
		if combo.Type == ComboPair {
			if len(combo.Tiles) > 0 {
				t := combo.Tiles[0]
				if t.Type == Wind || t.Type == Dragon {
					return false
				}
			}
		}
	}
	for _, m := range ctx.Melds {
		if m.Type != MeldTypeChow {
			return false
		}
	}
	return true
}

// evalDragons13 中發白刻子各 1 番
func evalDragons13(allTriplets []TileCombo, res *ScoreResult) {
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

// hasStraight 一條龍: 同花色 123+456+789
func hasStraight(p Partition, melds []Meld) bool {
	type seqKey struct {
		suit  TileType
		start int
	}
	seqs := make(map[seqKey]bool)

	for _, combo := range p.Combos {
		if combo.Type == ComboSequence && len(combo.Tiles) > 0 {
			seqs[seqKey{combo.Tiles[0].Type, combo.Tiles[0].Value}] = true
		}
	}
	for _, m := range melds {
		if m.Type == MeldTypeChow && len(m.Tiles) > 0 {
			minVal := m.Tiles[0].Value
			for _, t := range m.Tiles[1:] {
				if t.Value < minVal {
					minVal = t.Value
				}
			}
			seqs[seqKey{m.Tiles[0].Type, minVal}] = true
		}
	}

	for _, suit := range []TileType{Wan, Tong, Tiao} {
		if seqs[seqKey{suit, 1}] && seqs[seqKey{suit, 4}] && seqs[seqKey{suit, 7}] {
			return true
		}
	}
	return false
}

// hasTripleSameSequence 三相逢: 三花色同數順子 (例: 萬123+筒123+條123)
func hasTripleSameSequence(p Partition, melds []Meld) bool {
	type seqKey struct {
		suit  TileType
		start int
	}
	seqs := make(map[seqKey]bool)

	for _, combo := range p.Combos {
		if combo.Type == ComboSequence && len(combo.Tiles) > 0 {
			seqs[seqKey{combo.Tiles[0].Type, combo.Tiles[0].Value}] = true
		}
	}
	for _, m := range melds {
		if m.Type == MeldTypeChow && len(m.Tiles) > 0 {
			minVal := m.Tiles[0].Value
			for _, t := range m.Tiles[1:] {
				if t.Value < minVal {
					minVal = t.Value
				}
			}
			seqs[seqKey{m.Tiles[0].Type, minVal}] = true
		}
	}

	for start := 1; start <= 7; start++ {
		if seqs[seqKey{Wan, start}] && seqs[seqKey{Tong, start}] && seqs[seqKey{Tiao, start}] {
			return true
		}
	}
	return false
}

// isMixedTerminals 混帶么: 每組（順子/刻子/雀頭）都包含么九牌或字牌
func isMixedTerminals(p Partition, melds []Meld) bool {
	hasTerminalOrHonor := func(tiles []Tile) bool {
		for _, t := range tiles {
			if t.Type == Wind || t.Type == Dragon {
				return true
			}
			if t.Value == 1 || t.Value == 9 {
				return true
			}
		}
		return false
	}

	for _, combo := range p.Combos {
		if !hasTerminalOrHonor(combo.Tiles) {
			return false
		}
	}
	for _, m := range melds {
		if !hasTerminalOrHonor(m.Tiles) {
			return false
		}
	}
	return true
}

// isUniqueWait 獨聽: 只聽一張牌（單騎、邊張、嵌張）
func isUniqueWait(ctx ScoringContext, p Partition) bool {
	wt := ctx.WinningTile

	for _, combo := range p.Combos {
		// 單騎（雀頭等待）
		if combo.Type == ComboPair {
			if combo.Tiles[0].Type == wt.Type && combo.Tiles[0].Value == wt.Value {
				return true
			}
		}

		// 嵌張（順子中間張等待）
		if combo.Type == ComboSequence && len(combo.Tiles) >= 3 {
			mid := combo.Tiles[1]
			if mid.Type == wt.Type && mid.Value == wt.Value {
				return true
			}
		}

		// 邊張（1-2-3 等 3 或 7-8-9 等 7）
		if combo.Type == ComboSequence && len(combo.Tiles) >= 3 {
			if combo.Tiles[0].Value == 1 && wt.Type == combo.Tiles[0].Type && wt.Value == 3 {
				return true
			}
			if combo.Tiles[0].Value == 7 && wt.Type == combo.Tiles[0].Type && wt.Value == 7 {
				return true
			}
		}
	}
	return false
}

// evalFlush13 花色判斷（13張用，只檢查清一色）
func evalFlush13(allTiles []Tile, res *ScoreResult) {
	if isFullFlush(allTiles) {
		res.AddPattern("清一色", 10)
	}
}
