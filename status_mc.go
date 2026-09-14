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

// TODO: вынести обновление статуса в отдельную функцию
func updateServerStatus(client bot.Client) {
	server, err := mcstatus.NewJavaServer(minecraftServerIP)
	if err != nil {
		slog.Error("error init address minecraft server", slog.Any("err", err))
		return
	}

	status, err := server.Status()

	statusMutex.Lock()
	if err != nil {
		cachedStatus = ServerStatus{
			Online: false,
		}
		statusMutex.Unlock()

		_ = client.SetPresence(context.Background(),
			gateway.WithOnlineStatus(discord.OnlineStatusDND),
			gateway.WithPlayingActivity("Сервер оффлайн"),
		)
		return
	}

	if resp, ok := status.(*mcstatus.JavaStatusResponse); ok {
		cachedStatus = ServerStatus{
			Online:     true,
			PlayersNow: resp.Players.Online,
			PlayersMax: resp.Players.Max,
		}
		statusMutex.Unlock()

		activityText := fmt.Sprintf("Онлайн: %d/%d", resp.Players.Online, resp.Players.Max)
		err := client.SetPresence(context.Background(),
			gateway.WithOnlineStatus(discord.OnlineStatusOnline),
			gateway.WithPlayingActivity(activityText),
		)
		if err != nil {
			slog.Error("failed to update discord presence", slog.Any("err", err))
		}
	} else {
		statusMutex.Unlock()
	}
}

func loopPingStatusServer(client bot.Client) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		updateServerStatus(client)
	}
}
