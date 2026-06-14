package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/andre-carbajal/go-mcstatus"
	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
)

var (
	minecraftServerIP = os.Getenv("MINECRAFT_IP")
	guildID           = snowflake.GetEnv("GUILD_ID")

	commands = []discord.ApplicationCommandCreate{
		discord.SlashCommandCreate{
			Name:        "ping",
			Description: "get status server Minecraft",
		},
	}
)

func main() {
	slog.Info("starting example...")
	slog.Info("disgo version", slog.String("version", disgo.Version))

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

	slog.Info("easy disocrd is now running. Press CTRL-C to exit.")
	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-s
}

func commandListener(event *events.ApplicationCommandInteractionCreate) {
	data := event.SlashCommandInteractionData()
	if data.CommandName() == "ping" {

		server, err := mcstatus.NewJavaServer(minecraftServerIP)
		if err != nil {
			slog.Info("error init address minecraft server")
			return
		}
		status, err := server.Status()
		if err != nil {
			_ = event.CreateMessage(discord.NewMessageCreate().
				WithContent("Server offline"),
			)
			return
		}

		if resp, ok := status.(*mcstatus.JavaStatusResponse); ok {
			status := fmt.Sprintf(
				"Онлайн: %d/%d",
				resp.Players.Online, resp.Players.Max,
			)

			err = event.CreateMessage(discord.NewMessageCreate().
				WithContent(status),
			)
			if err != nil {
				slog.Error("error on sending response", slog.Any("err", err))
			}
		}
	}
}
