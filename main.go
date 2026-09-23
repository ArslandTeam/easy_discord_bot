package main

import (
	"context"
	"fmt"
	"html"
	"log"
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
		bot.WithEventListenerFunc(func(e *events.MessageCreate) {
		}),
	)
	if err != nil {
		panic(err)
	}
	if err = client.OpenGateway(context.TODO()); err != nil {
		panic(err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, html.EscapeString("status ok"))
	})

	go func() {

		s := &http.Server{
			Addr:           fmt.Sprintf("%s:%s", os.Getenv("ADDRES"), os.Getenv("PORT")),
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
				statusText := fmt.Sprintf("Online: %d/%d 🎮", status.OnlinePlayers, status.MaxPlayers)

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
}
