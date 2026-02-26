package models

// GetBestDiscard 根據手牌評估給出最適合丟棄的牌
// 基礎 AI 邏輯：尋找孤立無援的牌，優先丟棄風牌、元牌等字牌，再來是邊張(1, 9)，最不傾向丟棄中張(2-8)
func GetBestDiscard(hand []Tile) Tile {
	if len(hand) == 0 {
		return Tile{} // 不應該發生的極端狀況
	}

	// 1. 將手牌轉換為容易計算相鄰牌的陣列 (Index: 0-33)
	counts := make([]int, 34)
	for _, t := range hand {
		idx := t.ToIndex()
		if idx != -1 {
			counts[idx]++
		}
	}

	bestDiscard := hand[0]
	lowestScore := 9999

	// 2. 針對每一張手牌進行評分，分數越低代表對目前的牌型越沒幫助，越應該丟棄
	// 因為同樣的牌評分會一樣，我們可以用一個 map 記錄評估過的 index 避免重複算
	evaluated := make(map[int]bool)

	for _, t := range hand {
		idx := t.ToIndex()
		if idx == -1 {
			// 如果是花牌或是未知的牌，優先丟掉 (一般來說花牌拿到會直接補花，不會在手牌中)
			return t
		}

		if evaluated[idx] {
			continue
		}
		evaluated[idx] = true

		score := evaluateTile(idx, counts)
		if score < lowestScore {
			lowestScore = score
			bestDiscard = t
		}
	}

	return bestDiscard
}

// evaluateTile 評估某張牌在當前手牌陣列中的價值 (分數越低越該丟)
func evaluateTile(idx int, counts []int) int {
	score := 0

	// 基本價值評分
	if counts[idx] >= 3 {
		// 已經是刻子，非常有貢獻，不要隨便拆
		score += 100
	} else if counts[idx] == 2 {
		// 是對子 (雀頭候选或碰牌候选)
		score += 50
	}

	// 根據牌型分類計算相鄰關聯性
	if idx >= 27 {
		// 字牌 (風牌 27-30, 元牌 31-33)
		// 字牌沒有順子，所以只要不是對子或刻子，它就是純孤張
		if counts[idx] == 1 {
			score += 0 // 孤立的字牌最有可能是要被丟棄的
		}
	} else {
		// 數牌 (萬 0-8, 筒 9-17, 條 18-26)
		// 基礎權重：中張(排在3-7)比較容易湊順子，邊張(1,2,8,9)比較難
		val := (idx % 9) + 1
		if val == 1 || val == 9 {
			score += 5 // 邊張
		} else if val == 2 || val == 8 {
			score += 10 // 次邊張
		} else {
			score += 15 // 中張
		}

		// 檢查周圍是否有相鄰的牌 (湊搭子)
		// 同花色的合法檢查範圍
		minIdx := (idx / 9) * 9
		maxIdx := minIdx + 8

		// 檢查左邊的牌 (例如 4萬 看 3萬, 2萬)
		if idx-1 >= minIdx && counts[idx-1] > 0 {
			score += 20 // 雙頭搭子候选 (-1)
			if idx-2 >= minIdx && counts[idx-2] > 0 {
				score += 30 // 已經形成順子 (-2, -1, 0)
			}
		}

		// 檢查右邊的牌 (例如 4萬 看 5萬, 6萬)
		if idx+1 <= maxIdx && counts[idx+1] > 0 {
			score += 20 // 雙頭搭子候选 (+1)
			if idx+2 <= maxIdx && counts[idx+2] > 0 {
				score += 30 // 已經形成順子 (0, +1, +2)
			}
		}

		// 檢查坎張/夾張 (例如 4萬 看 2萬或6萬)
		if idx-2 >= minIdx && counts[idx-2] > 0 && counts[idx-1] == 0 {
			score += 10 // 夾張搭子
		}
		if idx+2 <= maxIdx && counts[idx+2] > 0 && counts[idx+1] == 0 {
			score += 10 // 夾張搭子
		}
	}

	return score
}
