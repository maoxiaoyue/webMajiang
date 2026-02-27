package controllers

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"sort"

	"golang.org/x/crypto/chacha20"

	"webmajiang/models"
	"webmajiang/service"
)

// NewDeck 初始化 144 張麻將牌
func NewDeck() []models.Tile {
	deck := make([]models.Tile, 0, 144)
	id := 0

	// 萬、筒、條 各 1-9，每種 4 張 → 3 × 9 × 4 = 108 張
	for _, tileType := range []models.TileType{models.Wan, models.Tong, models.Tiao} {
		for value := 1; value <= 9; value++ {
			for copy := 0; copy < 4; copy++ {
				deck = append(deck, models.Tile{
					ID:    id,
					Type:  tileType,
					Value: value,
				})
				id++
			}
		}
	}

	// 風牌: 東南西北，各 4 張 → 4 × 4 = 16 張
	for value := 1; value <= 4; value++ {
		for copy := 0; copy < 4; copy++ {
			deck = append(deck, models.Tile{
				ID:    id,
				Type:  models.Wind,
				Value: value,
			})
			id++
		}
	}

	// 元牌 (三元牌): 中發白，各 4 張 → 3 × 4 = 12 張
	for value := 1; value <= 3; value++ {
		for copy := 0; copy < 4; copy++ {
			deck = append(deck, models.Tile{
				ID:    id,
				Type:  models.Dragon,
				Value: value,
			})
			id++
		}
	}

	// 花牌: 梅蘭竹菊春夏秋冬，各 1 張 → 8 張
	for value := 1; value <= 8; value++ {
		deck = append(deck, models.Tile{
			ID:    id,
			Type:  models.Flower,
			Value: value,
		})
		id++
	}

	return deck
}

// chacha20Rand 使用 ChaCha20 產生密碼學安全的偽隨機數
// 以 crypto/rand 為種子，生成 ChaCha20 key + nonce
type chacha20Rand struct {
	cipher *chacha20.Cipher
	buf    [8]byte
}

// newChaCha20Rand 建立 ChaCha20 隨機數產生器 (crypto/rand 做種子)
func newChaCha20Rand() (*chacha20Rand, error) {
	// 使用 crypto/rand 產生 32-byte key + 12-byte nonce
	var key [32]byte
	var nonce [12]byte

	if _, err := rand.Read(key[:]); err != nil {
		return nil, fmt.Errorf("failed to generate ChaCha20 key: %w", err)
	}
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, fmt.Errorf("failed to generate ChaCha20 nonce: %w", err)
	}

	cipher, err := chacha20.NewUnauthenticatedCipher(key[:], nonce[:])
	if err != nil {
		return nil, fmt.Errorf("failed to create ChaCha20 cipher: %w", err)
	}

	return &chacha20Rand{cipher: cipher}, nil
}

// Intn 回傳 [0, n) 範圍內的隨機整數
func (r *chacha20Rand) Intn(n int) int {
	// 用 ChaCha20 keystream 加密全零 bytes 產生亂數
	for i := range r.buf {
		r.buf[i] = 0
	}
	r.cipher.XORKeyStream(r.buf[:], r.buf[:])
	return int(binary.LittleEndian.Uint64(r.buf[:]) % uint64(n))
}

// ShuffleChacha20 使用 crypto/rand 種子的 ChaCha20 洗牌
func ShuffleChacha20(deck []models.Tile) error {
	rng, err := newChaCha20Rand()
	if err != nil {
		return err
	}

	// Fisher-Yates shuffle
	for i := len(deck) - 1; i > 0; i-- {
		j := rng.Intn(i + 1)
		deck[i], deck[j] = deck[j], deck[i]
	}

	return nil
}

// SortHand 整理手牌（依類型和數值排序）
func SortHand(hand []models.Tile) {
	sort.Slice(hand, func(i, j int) bool {
		if hand[i].Type != hand[j].Type {
			return hand[i].Type < hand[j].Type
		}
		return hand[i].Value < hand[j].Value
	})
}

// DeckRedisKey 牌堆在 Redis 中的 key 格式
func DeckRedisKey(gameID string) string {
	return fmt.Sprintf("game:%s:deck", gameID)
}

// InitDeckToRedis 生成 144 張牌 → ChaCha20 洗牌 → LPUSH 到 Redis list
// 回傳 gameID 用來識別這場遊戲的牌堆
func InitDeckToRedis(ctx context.Context, gameID string) error {
	// 1. 生成牌堆
	deck := NewDeck()

	// 2. 使用 crypto/rand + ChaCha20 洗牌
	if err := ShuffleChacha20(deck); err != nil {
		return fmt.Errorf("shuffle failed: %w", err)
	}

	// 3. 序列化每張牌為 JSON，用 LPUSH 放入 Redis list
	redisKey := DeckRedisKey(gameID)

	// 先清除舊的牌堆（如果存在）
	if err := service.RedisClient.Del(ctx, redisKey).Err(); err != nil {
		return fmt.Errorf("failed to delete old deck: %w", err)
	}

	// 將所有牌序列化為 JSON 字串
	tileJSONs := make([]interface{}, len(deck))
	for i, tile := range deck {
		data, err := json.Marshal(tile)
		if err != nil {
			return fmt.Errorf("failed to marshal tile %d: %w", tile.ID, err)
		}
		tileJSONs[i] = string(data)
	}

	// LPUSH 一次推入所有牌（Redis 會按順序放入 list 頭部）
	// 最後推入的在最前面，所以摸牌時用 RPOP 即可按洗牌順序取出
	if err := service.RedisClient.LPush(ctx, redisKey, tileJSONs...).Err(); err != nil {
		return fmt.Errorf("LPUSH deck to redis failed: %w", err)
	}

	return nil
}

// DrawTile 從 Redis 牌堆摸一張牌 (RPOP)
func DrawTile(ctx context.Context, gameID string) (*models.Tile, error) {
	redisKey := DeckRedisKey(gameID)

	data, err := service.RedisClient.RPop(ctx, redisKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to draw tile: %w", err)
	}

	var tile models.Tile
	if err := json.Unmarshal([]byte(data), &tile); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tile: %w", err)
	}

	return &tile, nil
}

// GetDeckCount 查詢牌堆剩餘張數
func GetDeckCount(ctx context.Context, gameID string) (int64, error) {
	redisKey := DeckRedisKey(gameID)
	return service.RedisClient.LLen(ctx, redisKey).Result()
}

// PlayerHandKey 玩家手牌在 Redis 中的 key 格式
// playerID: 1-4 對應 player1-player4
func PlayerHandKey(gameID string, playerID int) string {
	return fmt.Sprintf("game:%s:player%d", gameID, playerID)
}

// DealTiles 發牌：從 Redis 牌堆 RPOP，按麻將規則輪流發給 4 位玩家
// 發牌順序：
//  1. 輪流摸 4 張 × 3 輪 = 每人 12 張
//  2. 每人再摸 1 張 = 每人 13 張
//  3. 莊家 (1-4 目前假設 1 號為開發預設) 再摸 1 張開門 = 莊家 14 張
//
// 每張牌用 RPOP 從牌堆取出，LPUSH 放入對應玩家的 Redis list
func DealTiles(ctx context.Context, gameID string, dealerPlayerID int) error {
	rdb := service.RedisClient
	deckKey := DeckRedisKey(gameID)

	// 先清除舊的玩家手牌
	for p := 1; p <= 4; p++ {
		playerKey := PlayerHandKey(gameID, p)
		if err := rdb.Del(ctx, playerKey).Err(); err != nil {
			return fmt.Errorf("failed to clear player%d hand: %w", p, err)
		}
	}

	// ----- 第一階段：輪流摸 4 張 × 3 輪 -----
	for round := 0; round < 3; round++ {
		for p := 1; p <= 4; p++ {
			playerKey := PlayerHandKey(gameID, p)
			for t := 0; t < 4; t++ {
				// RPOP 從牌堆尾端取一張牌
				data, err := rdb.RPop(ctx, deckKey).Result()
				if err != nil {
					return fmt.Errorf("round %d, player%d, tile %d: RPOP failed: %w", round+1, p, t+1, err)
				}
				// LPUSH 放入玩家手牌
				if err := rdb.LPush(ctx, playerKey, data).Err(); err != nil {
					return fmt.Errorf("round %d, player%d, tile %d: LPUSH failed: %w", round+1, p, t+1, err)
				}
			}
		}
	}

	// ----- 第二階段：每人再摸 1 張 -----
	for p := 1; p <= 4; p++ {
		playerKey := PlayerHandKey(gameID, p)
		data, err := rdb.RPop(ctx, deckKey).Result()
		if err != nil {
			return fmt.Errorf("final draw, player%d: RPOP failed: %w", p, err)
		}
		if err := rdb.LPush(ctx, playerKey, data).Err(); err != nil {
			return fmt.Errorf("final draw, player%d: LPUSH failed: %w", p, err)
		}
	}

	// ----- 第三階段：莊家再摸 1 張開門 -----
	dealerKey := PlayerHandKey(gameID, dealerPlayerID)
	data, err := rdb.RPop(ctx, deckKey).Result()
	if err != nil {
		return fmt.Errorf("dealer open tile: RPOP failed: %w", err)
	}
	if err := rdb.LPush(ctx, dealerKey, data).Err(); err != nil {
		return fmt.Errorf("dealer open tile: LPUSH failed: %w", err)
	}

	// ----- 第四階段：理牌（排序每位玩家的手牌）-----
	for p := 1; p <= 4; p++ {
		if err := SortPlayerHand(ctx, gameID, p); err != nil {
			return fmt.Errorf("sort player%d hand failed: %w", p, err)
		}
	}

	return nil
}

// SortPlayerHand 理牌：讀取玩家在 Redis 中的所有手牌，排序後重新寫回
func SortPlayerHand(ctx context.Context, gameID string, playerID int) error {
	rdb := service.RedisClient
	playerKey := PlayerHandKey(gameID, playerID)

	// 1. 讀取所有手牌 JSON
	tileJSONs, err := rdb.LRange(ctx, playerKey, 0, -1).Result()
	if err != nil {
		return fmt.Errorf("failed to read player%d hand: %w", playerID, err)
	}

	// 2. 反序列化為 Tile 切片
	tiles := make([]models.Tile, 0, len(tileJSONs))
	for _, tj := range tileJSONs {
		var tile models.Tile
		if err := json.Unmarshal([]byte(tj), &tile); err != nil {
			return fmt.Errorf("failed to unmarshal tile: %w", err)
		}
		tiles = append(tiles, tile)
	}

	// 3. 排序（依類型 → 數值）
	SortHand(tiles)

	// 4. 清除原有 list，重新寫入排序後的手牌
	if err := rdb.Del(ctx, playerKey).Err(); err != nil {
		return fmt.Errorf("failed to clear player%d hand for re-sort: %w", playerID, err)
	}

	sortedJSONs := make([]interface{}, len(tiles))
	for i, tile := range tiles {
		data, err := json.Marshal(tile)
		if err != nil {
			return fmt.Errorf("failed to marshal sorted tile: %w", err)
		}
		sortedJSONs[i] = string(data)
	}

	// RPUSH 保持排序順序（第一張在 index 0）
	if err := rdb.RPush(ctx, playerKey, sortedJSONs...).Err(); err != nil {
		return fmt.Errorf("failed to write sorted hand: %w", err)
	}

	return nil
}

// GetPlayerHand 取得玩家的手牌
func GetPlayerHand(ctx context.Context, gameID string, playerID int) ([]models.Tile, error) {
	playerKey := PlayerHandKey(gameID, playerID)

	tileJSONs, err := service.RedisClient.LRange(ctx, playerKey, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get player%d hand: %w", playerID, err)
	}

	tiles := make([]models.Tile, 0, len(tileJSONs))
	for _, tj := range tileJSONs {
		var tile models.Tile
		if err := json.Unmarshal([]byte(tj), &tile); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tile: %w", err)
		}
		tiles = append(tiles, tile)
	}

	return tiles, nil
}

// RemoveTileFromPlayerHand 從玩家手牌中移除特定的一張牌
func RemoveTileFromPlayerHand(ctx context.Context, gameID string, playerID int, targetTile models.Tile) error {
	rdb := service.RedisClient
	playerKey := PlayerHandKey(gameID, playerID)

	// 1. 讀取所有手牌
	tileJSONs, err := rdb.LRange(ctx, playerKey, 0, -1).Result()
	if err != nil {
		return fmt.Errorf("failed to read player%d hand: %w", playerID, err)
	}

	// 2. 找到並移除指定的牌 (只移除一張)
	removed := false
	remainingTiles := make([]models.Tile, 0, len(tileJSONs)-1)

	for _, tj := range tileJSONs {
		var tile models.Tile
		if err := json.Unmarshal([]byte(tj), &tile); err != nil {
			return fmt.Errorf("failed to unmarshal tile: %w", err)
		}

		// 比對 Type 和 Value 即可，因為 ID 是唯一對應特定的實體牌
		// 我們比對 ID 也行，確保移除的那張就是他在畫面上點擊的
		if !removed && tile.ID == targetTile.ID {
			removed = true
			continue // 跳過這張牌，不加入 remainingTiles
		}
		remainingTiles = append(remainingTiles, tile)
	}

	if !removed {
		return fmt.Errorf("tile not found in player hand")
	}

	// 3. 重新寫回 Redis (已移除指定的牌)
	// 先清除原有 list
	if err := rdb.Del(ctx, playerKey).Err(); err != nil {
		return fmt.Errorf("failed to clear player%d hand for removal: %w", playerID, err)
	}

	// 寫入剩餘的牌
	if len(remainingTiles) > 0 {
		sortedJSONs := make([]interface{}, len(remainingTiles))
		for i, tile := range remainingTiles {
			data, err := json.Marshal(tile)
			if err != nil {
				return fmt.Errorf("failed to marshal remaining tile: %w", err)
			}
			sortedJSONs[i] = string(data)
		}

		if err := rdb.RPush(ctx, playerKey, sortedJSONs...).Err(); err != nil {
			return fmt.Errorf("failed to write remaining hand: %w", err)
		}
	}

	return nil
}

// GetAllPlayersHands 取得所有玩家的手牌（含牌名）
func GetAllPlayersHands(ctx context.Context, gameID string) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	for p := 1; p <= 4; p++ {
		tiles, err := GetPlayerHand(ctx, gameID, p)
		if err != nil {
			return nil, err
		}

		// 產生可讀的牌名列表
		tileNames := make([]string, len(tiles))
		for i, t := range tiles {
			tileNames[i] = t.String()
		}

		playerKey := fmt.Sprintf("player%d", p)
		result[playerKey] = map[string]interface{}{
			"count":      len(tiles),
			"tiles":      tiles,
			"tile_names": tileNames,
		}
	}

	return result, nil
}
