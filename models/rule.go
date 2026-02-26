package models

// ToIndex 將麻將牌轉換為 0-33 的索引值，方便進行胡牌演算法的計算
// 萬: 0-8 (1-9萬)
// 筒: 9-17 (1-9筒)
// 條: 18-26 (1-9條)
// 風: 27-30 (東南西北, 1-4)
// 元: 31-33 (中發白, 1-3)
// 若為花牌或其他非標準成組牌則回傳 -1
func (t Tile) ToIndex() int {
	switch t.Type {
	case Wan:
		return t.Value - 1
	case Tong:
		return 9 + t.Value - 1
	case Tiao:
		return 18 + t.Value - 1
	case Wind:
		return 27 + t.Value - 1
	case Dragon:
		return 31 + t.Value - 1
	}
	return -1
}

// CanHu 判斷當前手牌是否滿足基本胡牌型態
// 基本胡牌型：N 組「順子或刻子」 + 1 個「對子」(雀頭)
// 支援任何 3N + 2 張牌的狀況（如：14張、17張）
func CanHu(hand []Tile) bool {
	counts := make([]int, 34)
	total := 0

	for _, t := range hand {
		idx := t.ToIndex()
		if idx != -1 {
			counts[idx]++
			total++
		}
	}

	// 胡牌的總張數（排除花牌之後）必須是 3N + 2
	if total%3 != 2 {
		return false
	}

	// 嘗試將每一種牌當作「對子」(雀頭) 來判定
	for i := 0; i < 34; i++ {
		if counts[i] >= 2 {
			// 取出兩張當作對子
			counts[i] -= 2

			// 檢查剩餘的牌是否能全部分解為順子或刻子
			if checkCombos(counts) {
				return true
			}

			// 復原
			counts[i] += 2
		}
	}

	return false
}

// checkCombos 遞迴檢查陣列中的牌是否能完美分解為「刻子」(三張同牌) 或「順子」(三張連續)
func checkCombos(counts []int) bool {
	for i := 0; i < 34; i++ {
		if counts[i] == 0 {
			continue // 該牌沒有數量，檢查下一張
		}

		// 1. 嘗試組成「刻子」(Triplet)
		if counts[i] >= 3 {
			counts[i] -= 3
			if checkCombos(counts) {
				// 若成功分解，則層層回傳 true (不需要復原，因為已經確定胡牌了，但為了保持乾淨狀態還是加回去)
				counts[i] += 3
				return true
			}
			// 若此路不通，復原並嘗試其他組合 (如順子)
			counts[i] += 3
		}

		// 2. 嘗試組成「順子」(Sequence)
		// 規則：只有萬、筒、條（索引 < 27）可以組成順子
		// 加上不能跨花色：Value 只能是 1~7 (亦即 i%9 <= 6)
		if i < 27 && i%9 <= 6 && counts[i] > 0 && counts[i+1] > 0 && counts[i+2] > 0 {
			counts[i]--
			counts[i+1]--
			counts[i+2]--
			if checkCombos(counts) {
				counts[i]++
				counts[i+1]++
				counts[i+2]++
				return true
			}
			counts[i]++
			counts[i+1]++
			counts[i+2]++
		}

		// 如果走到這裡，代表這張牌既無法湊成刻子也無法湊成順子，因此此分解方式失敗
		return false
	}

	// 所有數字都等於 0，代表成功全部分解完畢！
	return true
}
