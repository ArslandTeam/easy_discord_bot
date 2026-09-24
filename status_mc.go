package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/andre-carbajal/go-mcstatus"
)

type ResponseServerInfo struct {
	IsOnline         bool
	MaxPlayers       int
	OnlinePlayers    int
	Description      string
	VersionMinecraft string
	Latency          int64
}

func pingMinecraftJavaServer() (ResponseServerInfo, error) {
	server, err := mcstatus.NewJavaServer(os.Getenv("MINECRAFT_ADDRESS"))
	if err != nil {
		return ResponseServerInfo{
			IsOnline: false,
		}, errors.New("Error create client")
	}

	status, err := server.Status()
	if err != nil {
		return ResponseServerInfo{
			IsOnline: false,
		}, errors.New("Error server unavailable")
	}

	resp, ok := status.(*mcstatus.JavaStatusResponse)

	if !ok {
		return ResponseServerInfo{
			IsOnline: false,
		}, errors.New("Error type response server")
	}

	return ResponseServerInfo{
		IsOnline:         true,
		MaxPlayers:       resp.Players.Max,
		OnlinePlayers:    resp.Players.Online,
		VersionMinecraft: resp.Version.Name,
		Description:      fmt.Sprint(resp.Description),
		Latency:          resp.GetLatency(),
	}, nil
}
