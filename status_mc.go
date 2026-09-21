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

func updateDiscordStatus(ctx context.Context, client bot.Client) {
	go func() {
		update := func() {
			status, err := updateServerStatus()
			if err != nil {
				slog.Error("failed to fetch server status", slog.Any("err", err))
				status.Online = false
			}

			statusMutex.Lock()
			cachedStatus = status
			statusMutex.Unlock()

			var statusText string
			var onlineStatus discord.OnlineStatus

			if status.Online {
				statusText = fmt.Sprintf("Онлайн: %d/%d", status.PlayersNow, status.PlayersMax)
				onlineStatus = discord.OnlineStatusOnline
			} else {
				statusText = "Сервер оффлайн"
				onlineStatus = discord.OnlineStatusDND
			}

			err = client.SetPresence(ctx,
				gateway.WithOnlineStatus(onlineStatus),
				gateway.WithPlayingActivity(statusText),
			)
			if err != nil {
				slog.Error("failed to update discord presence", slog.Any("err", err))
			}
		}

		update()

		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				update()
			case <-ctx.Done():
				slog.Info("stopping discord status updater")
				return
			}
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
	if err != nil {
		slog.Error("failed to get minecraft server status", slog.Any("err", err))
		return ServerStatus{}, fmt.Errorf("get server status: %w", err)
	}

	resp, ok := status.(*mcstatus.JavaStatusResponse)
	if !ok {
		return ServerStatus{
			Online: false,
		}, fmt.Errorf("unexpected status response type: %T", status)
	}

	return ServerStatus{
		Online:     true,
		PlayersNow: resp.Players.Online,
		PlayersMax: resp.Players.Max,
	}, nil
}
