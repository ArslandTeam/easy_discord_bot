package main

import (
	"log/slog"
	"time"

	"github.com/andre-carbajal/go-mcstatus"
)

func updateServerStatus() {
	server, err := mcstatus.NewJavaServer(minecraftServerIP)
	if err != nil {
		slog.Error("error init address minecraft server", slog.Any("err", err))
		return
	}

	status, err := server.Status()

	statusMutex.Lock()
	defer statusMutex.Unlock()

	if err != nil {
		cachedStatus = ServerStatus{
			Online: false,
		}
		return
	}

	if resp, ok := status.(*mcstatus.JavaStatusResponse); ok {
		cachedStatus = ServerStatus{
			Online:     true,
			PlayersNow: resp.Players.Online,
			PlayersMax: resp.Players.Max,
		}
	}
}

func loopPingStatusServer() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		updateServerStatus()
	}
}
