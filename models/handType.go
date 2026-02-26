package models

// IsAllTriplets 判斷手牌是否為「對對胡」 (碰碰胡)
// 四組刻子加一組對子
func IsAllTriplets(hand []Tile) bool {
	if !CanHu(hand) {
		return false // 必須先滿足基本胡牌條件
	}

	counts := make([]int, 34)
	for _, t := range hand {
		idx := t.ToIndex()
		if idx != -1 {
			counts[idx]++
		}
	}

	pairs := 0
	triplets := 0

	for _, count := range counts {
		if count >= 3 {
			t := count / 3
			triplets += t
			if count%3 == 2 {
				pairs++
			}
		} else if count == 2 {
			pairs++
		} else if count == 1 {
			// 對對胡手牌中不應該存在單張無法組合的牌
			// (因為前提已過 CanHu，如果有單張必定是順子的一部分，順子就違反對對胡了)
			return false
		}
	}

	// 對對胡條件：除了雀頭(1個對子)以外，其餘全是刻子(或槓子)
	return pairs == 1 && triplets*3+2 == len(hand)
}

// IsBigFourWinds 判斷手牌是否為「大四喜」
// 包含四組風牌的刻子(或槓子)，以及一組對子
func IsBigFourWinds(hand []Tile) bool {
	counts := make([]int, 34)
	for _, t := range hand {
		idx := t.ToIndex()
		if idx != -1 {
			counts[idx]++
		}
	}

	windIdx := []int{27, 28, 29, 30} // 東、南、西、北
	for _, idx := range windIdx {
		if counts[idx] < 3 {
			// 只要有任何一個風牌不足 3 張，就不可能大四喜
			return false
		}
	}

	// 確保能胡牌（有雀頭並且剩下的有搭配好）
	return CanHu(hand)
}

// IsSmallFourWinds 判斷手牌是否為「小四喜」
// 包含三組風牌的刻子(或槓子)，以及一組風牌的對子
func IsSmallFourWinds(hand []Tile) bool {
	counts := make([]int, 34)
	for _, t := range hand {
		idx := t.ToIndex()
		if idx != -1 {
			counts[idx]++
		}
	}

	windIdx := []int{27, 28, 29, 30} // 東、南、西、北
	windTriplets := 0
	windPairs := 0

	for _, idx := range windIdx {
		if counts[idx] >= 3 {
			windTriplets++
		} else if counts[idx] == 2 {
			windPairs++
		}
	}

	// 小四喜條件：3 組風牌刻子 + 1 組風牌對子（雀頭）
	// 大四喜也算包含小四喜的刻子條件，若要嚴格排除大四喜，需要要求風刻為 3 且風對為 1
	return windTriplets == 3 && windPairs == 1 && CanHu(hand)
}

// IsBigThreeDragons 判斷手牌是否為「大三元」
// 包含三組元牌(中、發、白)的刻子(或槓子)
func IsBigThreeDragons(hand []Tile) bool {
	counts := make([]int, 34)
	for _, t := range hand {
		idx := t.ToIndex()
		if idx != -1 {
			counts[idx]++
		}
	}

	dragonIdx := []int{31, 32, 33} // 中、發、白
	for _, idx := range dragonIdx {
		if counts[idx] < 3 {
			return false
		}
	}

	return CanHu(hand)
}

// IsSmallThreeDragons 判斷手牌是否為「小三元」
// 包含兩組元牌(中、發、白)的刻子(或槓子)，以及一組元牌的對子
func IsSmallThreeDragons(hand []Tile) bool {
	counts := make([]int, 34)
	for _, t := range hand {
		idx := t.ToIndex()
		if idx != -1 {
			counts[idx]++
		}
	}

	dragonIdx := []int{31, 32, 33} // 中、發、白
	dragonTriplets := 0
	dragonPairs := 0

	for _, idx := range dragonIdx {
		if counts[idx] >= 3 {
			dragonTriplets++
		} else if counts[idx] == 2 {
			dragonPairs++
		}
	}

	return dragonTriplets == 2 && dragonPairs == 1 && CanHu(hand)
}
