package main

import (
	"context"
	"fmt"
	"html"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/snowflake/v2"
)

// INFO позже вынести var ( commands ) в отдельный модуль
var (
	commands = []discord.ApplicationCommandCreate{
		discord.SlashCommandCreate{
			Name:        "ping",
			Description: "get status server minecraft",
			DescriptionLocalizations: map[discord.Locale]string{
				discord.LocaleRussian: "получить актуальный статус сервера Minecraft",
			},
		},
		discord.SlashCommandCreate{
			Name:        "infoserver",
			Description: "get full info server",
			DescriptionLocalizations: map[discord.Locale]string{
				discord.LocaleRussian: "получить полную ифнормацию о сервере",
			},
		},
	}
)

func main() {
	client, err := disgo.New(os.Getenv("TOKEN"),
		bot.WithGatewayConfigOpts(
			gateway.WithIntents(
				gateway.IntentGuilds,
				gateway.IntentGuildMessages,
				gateway.IntentDirectMessages,
			),
		),
		bot.WithEventListenerFunc(commandListener),
	)

	if _, err = client.Rest.SetGuildCommands(client.ApplicationID, snowflake.GetEnv("GUILD_ID"), commands); err != nil {
		panic("error while registering commands: " + err.Error())
	}

	if err = client.OpenGateway(context.TODO()); err != nil {
		panic(err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, html.EscapeString("status ok"))
	})

	go func() {

		s := &http.Server{
			Addr:           fmt.Sprintf("%s:%s", os.Getenv("ADDRESS"), os.Getenv("PORT")),
			Handler:        handler,
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			MaxHeaderBytes: 1 << 20,
		}

		log.Fatal(s.ListenAndServe())
	}()

	go func(client *bot.Client) {
		for {
			status, err := pingMinecraftJavaServer()
			if err != nil {
				log.Println(err)
				errDiscord := client.SetPresence(context.Background(),
					gateway.WithOnlineStatus(discord.OnlineStatusDND),
					gateway.WithCustomActivity("Server offline"),
				)
				log.Println(errDiscord)

			} else {
				statusText := fmt.Sprintf("Online: %d/%d", status.OnlinePlayers, status.MaxPlayers)

				errDiscord := client.SetPresence(context.Background(),
					gateway.WithOnlineStatus(discord.OnlineStatusOnline),
					gateway.WithCustomActivity(statusText),
				)
				log.Println(errDiscord)

			}

			time.Sleep(400 * time.Second)
		}
	}(client)

	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGINT, syscall.SIGTERM)
	<-s
	client.Close(context.TODO())
}

func commandListener(event *events.ApplicationCommandInteractionCreate) {
	data := event.SlashCommandInteractionData()
	if data.CommandName() == "ping" {
		status, err := pingMinecraftJavaServer()

		if err != nil {
			log.Println(err)
			err = event.CreateMessage(discord.NewMessageCreate().WithContent("Server offline"))
		} else {
			statusText := fmt.Sprintf("Online: %d/%d", status.OnlinePlayers, status.MaxPlayers)
			err = event.CreateMessage(discord.NewMessageCreate().WithContent(statusText))
		}

		if err != nil {
			event.Client().Logger.Error("error on sending response", slog.Any("err", err))
		}
	}

	// TODO вынести настройки embed в отедельный json файл и там их редактивировать
	if data.CommandName() == "infoserver" {
		status, err := pingMinecraftJavaServer()

		var embed discord.Embed

		if err != nil {
			log.Println(err)
			embed = discord.NewEmbed().WithTitle("Status server").
				WithDescription("**Server offline**").
				WithColor(0xFF0000)
		} else {
			embed = discord.NewEmbed().WithTitle("Status server").
				WithColor(0x00FF00).
				AddField("Online", fmt.Sprintf("%d/%d", status.OnlinePlayers, status.MaxPlayers), false).
				AddField("Version", fmt.Sprintf("%s", status.VersionMinecraft), true).
				AddField("Ping server", fmt.Sprintf("%d ms", status.Latency), true).
				AddField("MOTD", status.Description, false)
		}

		err = event.CreateMessage(discord.NewMessageCreate().WithEmbeds(embed))
		if err != nil {
			event.Client().Logger.Error("error on sending response", slog.Any("err", err))
		}
	}
}
