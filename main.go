package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
)

type ServerStatus struct {
	Online     bool `json:"online"`
	PlayersNow int  `json:"players_online"`
	PlayersMax int  `json:"players_max"`
}

var (
	minecraftServerIP = os.Getenv("MINECRAFT_IP")
	guildID           = snowflake.GetEnv("GUILD_ID")

	statusMutex  sync.RWMutex
	cachedStatus ServerStatus

	commands = []discord.ApplicationCommandCreate{
		discord.SlashCommandCreate{
			Name:        "ping",
			Description: "get status server Minecraft",
			DescriptionLocalizations: map[discord.Locale]string{
				discord.LocaleRussian: "получить статус сервера Minecraft",
			},
		},
		discord.SlashCommandCreate{
			Name:        "infoserver",
			Description: "get info server Minecraft",
		},
	}
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client, err := disgo.New(os.Getenv("DISCORD_TOKEN"),
		bot.WithDefaultGateway(),
		bot.WithEventListenerFunc(commandListener),
	)
	if err != nil {
		slog.Error("error while building disgo instance", slog.Any("err", err))
		return
	}

	defer client.Close(context.TODO())

	if _, err = client.Rest.SetGuildCommands(client.ApplicationID, guildID, commands); err != nil {
		slog.Error("error while registering commands", slog.Any("err", err))
	}

	if err = client.OpenGateway(context.TODO()); err != nil {
		slog.Error("error while connecting to gateway", slog.Any("err", err))
	}

	go updateDiscordStatus(ctx, *client)

	http.HandleFunc("/", httpStatusHandler)

	go func() {
		if err := http.ListenAndServe(":5645", nil); err != nil {
			slog.Error("HTTP server failed", slog.Any("err", err))
		}
	}()

	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-s
	slog.Info("Shutting down bot...")
}

func httpStatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	statusMutex.RLock()
	currentStatus := cachedStatus
	statusMutex.RUnlock()

	json.NewEncoder(w).Encode(currentStatus)
}

func commandListener(event *events.ApplicationCommandInteractionCreate) {
	data := event.SlashCommandInteractionData()

	if data.CommandName() == "ping" {
		err := event.DeferCreateMessage(false)
		if err != nil {
			slog.Error("error on deferring message", slog.Any("err", err))
			return
		}

		status, err := updateServerStatus()
		if err != nil {
			slog.Error("failed to fetch live server status for ping command", slog.Any("err", err))
			status.Online = false
		}

		statusMutex.Lock()
		cachedStatus = status
		statusMutex.Unlock()

		var content string
		if status.Online {
			content = fmt.Sprintf("Онлайн: %d/%d", status.PlayersNow, status.PlayersMax)
		} else {
			content = "Сервер оффлайн"
		}

		_, err = event.Client().Rest.UpdateInteractionResponse(event.ApplicationID(), event.Token(),
			discord.NewMessageUpdate().WithContent(content),
		)
		if err != nil {
			slog.Error("error on updating ping response", slog.Any("err", err))
		}
	}
	if data.CommandName() == "infoServer" {
		context.TODO()
	}
}
