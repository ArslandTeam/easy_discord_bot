package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/andre-carbajal/go-mcstatus"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/gateway"
)

func updateDiscordStatus(client bot.Client) {
	go func() {
		checkAndExecute := func() {
			status, err := updateServerStatus()
			if err != nil || !status.Online {
				if err != nil {
					slog.Error("failed to fetch server status", slog.Any("err", err))
				}

				_ = client.SetPresence(context.Background(),
					gateway.WithOnlineStatus(discord.OnlineStatusDND),
					gateway.WithPlayingActivity("Сервер оффлайн"),
				)
				return
			}

			err = client.SetPresence(context.Background(),
				gateway.WithOnlineStatus(discord.OnlineStatusOnline),
				gateway.WithPlayingActivity(fmt.Sprintf("Онлайн: %d/%d", status.PlayersNow, status.PlayersMax)),
			)

			if err != nil {
				slog.Error("failed to update discord presence", slog.Any("err", err))
			}
		}

		checkAndExecute()
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			checkAndExecute()
		}
	}()
}

func updateServerStatus() (ServerStatus, error) {
	server, err := mcstatus.NewJavaServer(minecraftServerIP)
	if err != nil {
		slog.Error("error init address minecraft server", slog.Any("err", err))
		return ServerStatus{}, err
	}

	status, err := server.Status()

	statusMutex.Lock()
	if err != nil {
		cachedStatus = ServerStatus{
			Online: false,
		}
		statusMutex.Unlock()
		return cachedStatus, err
	}

	if resp, ok := status.(*mcstatus.JavaStatusResponse); ok {
		cachedStatus = ServerStatus{
			Online:     true,
			PlayersNow: resp.Players.Online,
			PlayersMax: resp.Players.Max,
		}
		statusMutex.Unlock()
		return cachedStatus, nil
	}

	cachedStatus = ServerStatus{
		Online: false,
	}
	statusMutex.Unlock()
	return cachedStatus, fmt.Errorf("invalid status response")
}
