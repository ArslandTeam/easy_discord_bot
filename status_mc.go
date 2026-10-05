package main

import (
	"context"
	"fmt"
	"time"

	"github.com/mcstatus-io/mcutil/v3"
)

// TODO переписать
type ResponseServerInfo struct {
	IsOnline         bool
	MaxPlayers       int64
	OnlinePlayers    int64
	Description      string
	VersionMinecraft string
	Latency          int64
}

func pingMinecraftJavaServer() (ResponseServerInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	response, err := mcutil.Status(ctx, minecraftAddress, minecraftPort)
	if err != nil {
		return ResponseServerInfo{IsOnline: false}, fmt.Errorf("Ping failed: %w", err)
	}

	return ResponseServerInfo{
		IsOnline:         true,
		MaxPlayers:       *response.Players.Max,
		OnlinePlayers:    *response.Players.Online,
		VersionMinecraft: response.Version.NameClean,
		Description:      response.MOTD.Clean,
		Latency:          response.Latency.Milliseconds(),
	}, nil
}
