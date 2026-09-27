package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Tnze/go-mc/bot"
	"github.com/Tnze/go-mc/chat"
)

// TODO переписать
type ResponseServerInfo struct {
	IsOnline         bool
	MaxPlayers       int
	OnlinePlayers    int
	Description      string
	VersionMinecraft string
	Latency          int64
}

type mcStatusResp struct {
	Version struct {
		Name string `json:"name"`
	} `json:"version"`
	Players struct {
		Max    int `json:"max"`
		Online int `json:"online"`
	} `json:"players"`
	Description chat.Message `json:"description"`
}

func pingMinecraftJavaServer() (ResponseServerInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	startTime := time.Now()

	respBytes, delay, err := bot.PingAndListContext(ctx, minecraftAddress)
	if err != nil {
		return ResponseServerInfo{IsOnline: false}, fmt.Errorf("Ping failed: %w", err)
	}

	var status mcStatusResp
	if err := json.Unmarshal(respBytes, &status); err != nil {
		return ResponseServerInfo{IsOnline: false}, fmt.Errorf("Failed to parse json response: %w", err)
	}

	latency := delay.Milliseconds()
	if latency == 0 {
		latency = time.Since(startTime).Milliseconds()
	}

	return ResponseServerInfo{
		IsOnline:         true,
		MaxPlayers:       status.Players.Max,
		OnlinePlayers:    status.Players.Online,
		VersionMinecraft: status.Version.Name,
		Description:      status.Description.ClearString(),
		Latency:          latency,
	}, nil
}
