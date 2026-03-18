package service

import (
	"database/sql"
	"fmt"
	"math/rand"
	"sync"

	_ "github.com/lib/pq"
)

// ConversationCache 對話字典快取（啟動時從 PostgreSQL 載入，之後全部從記憶體讀取）
type ConversationCache struct {
	mu   sync.RWMutex
	data map[string][]string // key = "type:role", value = list of content
}

var convCache = &ConversationCache{
	data: make(map[string][]string),
}

// InitConversations 從 PostgreSQL 載入對話字典到記憶體
func InitConversations(dsn string) error {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("connect to postgres: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}

	rows, err := db.Query("SELECT type, role, content FROM conversation_dict")
	if err != nil {
		return fmt.Errorf("query conversation_dict: %w", err)
	}
	defer rows.Close()

	convCache.mu.Lock()
	defer convCache.mu.Unlock()

	count := 0
	for rows.Next() {
		var typ, role, content string
		if err := rows.Scan(&typ, &role, &content); err != nil {
			continue
		}
		key := typ + ":" + role
		convCache.data[key] = append(convCache.data[key], content)
		count++
	}

	fmt.Printf("[Conversation] Loaded %d entries from conversation_dict\n", count)
	return nil
}

// GetRandomConversation 取得隨機對話內容
// typ: "draw", "discard", "pong", "hu" 等
// role: "bot", "player"
func GetRandomConversation(typ, role string) string {
	convCache.mu.RLock()
	defer convCache.mu.RUnlock()

	key := typ + ":" + role
	items := convCache.data[key]
	if len(items) == 0 {
		return ""
	}
	return items[rand.Intn(len(items))]
}
