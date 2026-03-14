package controllers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"webmajiang/models"

	hypcontext "github.com/maoxiaoyue/hypgo/pkg/context"
)

// lobbyPlayerInfo describes a player sent from gamelobby.
type lobbyPlayerInfo struct {
	SeatID   int    `json:"seat_id"`
	PlayerID string `json:"player_id"`
	Name     string `json:"name"`
	IsBot    bool   `json:"is_bot"`
}

// createGameFromLobbyRequest is the request from gamelobby internal API.
type createGameFromLobbyRequest struct {
	GameType int               `json:"game_type"` // 13 or 16
	Players  []lobbyPlayerInfo `json:"players"`
}

// CreateGameFromLobbyHandler handles POST /api/internal/create-game
// Called by gamelobby to create a game with specific players.
func CreateGameFromLobbyHandler(c *hypcontext.Context) {
	var req createGameFromLobbyRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":   "invalid request",
			"message": err.Error(),
		})
		return
	}

	// Validate game type
	gameType := models.GameType(req.GameType)
	if gameType != models.GameType13 && gameType != models.GameType16 {
		gameType = models.GameType16
	}

	// Validate players (need exactly 4)
	if len(req.Players) == 0 || len(req.Players) > 4 {
		c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":   "invalid players",
			"message": "need 1-4 players",
		})
		return
	}

	// Build player map
	players := make(map[int]models.Player)
	for _, p := range req.Players {
		seatID := p.SeatID
		if seatID < 1 || seatID > 4 {
			continue
		}
		players[seatID] = models.Player{
			ID:    seatID,
			Name:  p.Name,
			IsBot: p.IsBot,
			Hand:  []models.Tile{},
		}
	}

	// Fill remaining seats with AI
	for i := 1; i <= 4; i++ {
		if _, ok := players[i]; !ok {
			players[i] = models.Player{
				ID:    i,
				Name:  fmt.Sprintf("AI 電腦%d", i),
				IsBot: true,
				Hand:  []models.Tile{},
			}
		}
	}

	gameID := fmt.Sprintf("majiang_%d", time.Now().UnixNano())
	ctx := context.Background()

	state, err := StartNewGameWithPlayers(ctx, gameID, gameType, players)
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error":   "failed to start game",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"game_id": gameID,
		"token":   gameID, // game ID as token for now; can be JWT later
		"stage":   state.Stage,
	})
}
