package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	_ "github.com/lib/pq"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/omit"
	"github.com/disgoorg/snowflake/v2"
)

// INFO позже вынести var ( commands ) в отдельный модуль
var (
	adminPerms = discord.PermissionAdministrator
	commands   = []discord.ApplicationCommandCreate{
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
		discord.SlashCommandCreate{
			Name:        "check",
			Description: "Check database and assign verify roles to linked users",
			DescriptionLocalizations: map[discord.Locale]string{
				discord.LocaleRussian: "проверить базу данных и выдать роли верификации привязанным пользователям",
			},
			DefaultMemberPermissions: omit.New(&adminPerms),
		},
	}
	db *sql.DB
)

type Cache struct {
	mu     sync.RWMutex
	status ResponseServerInfo
}

func (c *Cache) Update(status ResponseServerInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.status = status
}

func (c *Cache) Get() ResponseServerInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.status
}

var globalCache = &Cache{}

func main() {
	if _, err := os.Stat(".env"); os.IsNotExist(err) {
		initConfig()
		os.Exit(0)
	}

	loadConfig()

	var err error
	db, err = sql.Open("postgres", databaseUrl)
	if err != nil {
		log.Fatalf("Error connect Database: %v", err)
	}
	defer db.Close()

	client, err := disgo.New(discordToken,
		bot.WithGatewayConfigOpts(
			gateway.WithIntents(
				gateway.IntentGuilds,
				gateway.IntentGuildMessages,
				gateway.IntentDirectMessages,
				gateway.IntentGuildMembers,
			),
		),
		bot.WithEventListenerFunc(commandListener),
		bot.WithEventListenerFunc(guildMemberJoinListener),
	)

	if err != nil {
		panic(err)
	}

	if _, err = client.Rest.SetGuildCommands(client.ApplicationID, snowflake.GetEnv("GUILD_ID"), commands); err != nil {
		panic("error while registering commands: " + err.Error())
	}

	if err = client.OpenGateway(context.TODO()); err != nil {
		panic(err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		status := globalCache.Get()
		if err := json.NewEncoder(w).Encode(status); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("GET /verify", func(w http.ResponseWriter, r *http.Request) {
		authToken := r.Header.Get("X-Authentication-Token")
		if authToken != expectedToken {
			http.Error(w, "Unauthorized: Invalid or missing token", http.StatusUnauthorized)
			return
		}

		userId := r.URL.Query().Get("user_id")
		if userId == "" {
			http.Error(w, "Bad Request: Missing user_id parameter", http.StatusBadRequest)
			return
		}

		parsedUserId, err := snowflake.Parse(userId)
		if err != nil {
			http.Error(w, "Bad Request: Invalid user_id format", http.StatusBadRequest)
			return
		}

		existsUser, err := checkAndAssignRole(*client, guildID, parsedUserId)
		if err != nil {
			log.Printf("Error in checkAndAssignRole for user %s: %v", userId, err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if !existsUser {
			http.Error(w, "User not found in database", http.StatusNotFound)
			return
		}

		w.Write([]byte("OK"))
	})

	go func() {

		s := &http.Server{
			Addr:           fmt.Sprintf("%s:%s", address, port),
			Handler:        mux,
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			MaxHeaderBytes: 1 << 20,
		}

		log.Fatal(s.ListenAndServe())
	}()

	go func() {
		for {
			updatePresence(client)
			time.Sleep(pingInterval)
		}
	}()

	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGINT, syscall.SIGTERM)
	<-s
	client.Close(context.TODO())
}

func updatePresence(client *bot.Client) {
	status, err := pingMinecraftJavaServer()
	var statusText = "Server offline"
	var onlineStatus = discord.OnlineStatusDND

	if err != nil {
		log.Println("Ping background error:", err)
		globalCache.Update(ResponseServerInfo{IsOnline: false})
	} else {
		onlineStatus = discord.OnlineStatusOnline
		statusText = fmt.Sprintf("Online: %d/%d", status.OnlinePlayers, status.MaxPlayers)
		globalCache.Update(status)
	}

	errDiscord := client.SetPresence(context.Background(),
		gateway.WithOnlineStatus(onlineStatus),
		gateway.WithCustomActivity(statusText),
	)

	if errDiscord != nil {
		log.Println(errDiscord)
	}
}

func checkAndAssignRole(client bot.Client, guildsnowflakeID snowflake.ID, userID snowflake.ID) (bool, error) {
	var exists bool

	err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM users WHERE discordId = $1)`, userID.String()).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("%w", err)
	}

	if exists {
		err = client.Rest.AddMemberRole(guildsnowflakeID, userID, verifiedRoleID)
		if err != nil {
			return true, fmt.Errorf("%w", err)
		}
		return true, nil
	}

	return false, nil
}

func guildMemberJoinListener(event *events.GuildMemberJoin) {
	if event.GuildID != guildID {
		return
	}

	_, err := checkAndAssignRole(*event.Client(), event.GuildID, event.Member.User.ID)
	if err != nil {
		log.Printf("%s: %v", event.Member.User.ID, err)
	}
}

func commandListener(event *events.ApplicationCommandInteractionCreate) {
	data := event.SlashCommandInteractionData()
	if data.CommandName() == "ping" {
		updatePresence(event.Client())

		status := globalCache.Get()

		if !status.IsOnline {
			event.CreateMessage(discord.NewMessageCreate().WithContent("Server offline"))
		} else {
			statusText := fmt.Sprintf("Online: %d/%d", status.OnlinePlayers, status.MaxPlayers)
			event.CreateMessage(discord.NewMessageCreate().WithContent(statusText))
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

	if data.CommandName() == "check" {
		verifiedCount := 0

		if err := event.DeferCreateMessage(true); err != nil {
			event.Client().Logger.Error("error on deferring response", slog.Any("err", err))
			return
		}

		go func() {
			ctx := context.Background()
			rows, err := db.QueryContext(ctx, `SELECT "discordId" FROM users WHERE "discordId" IS NOT NULL`)
			if err != nil {
				event.Client().Logger.Error("error fetch users in DB", slog.Any("err", err))
				_, _ = event.Client().Rest.UpdateInteractionResponse(event.ApplicationID(), event.Token(),
					discord.NewMessageUpdate().WithContent("Ошибка при чтении базы данных."))
				return
			}
			defer rows.Close()

			for rows.Next() {
				var discordIDStr string
				if err := rows.Scan(&discordIDStr); err != nil {
					continue
				}

				userID, err := snowflake.Parse(discordIDStr)
				if err != nil {
					continue
				}

				err = event.Client().Rest.AddMemberRole(guildID, userID, verifiedRoleID)
				if err != nil {
					event.Client().Logger.Debug(fmt.Sprintf("Could not add role to %s (maybe left server?)", discordIDStr))
					continue
				}

				verifiedCount++
			}

			if err = rows.Err(); err != nil {
				event.Client().Logger.Error("error occurred during database rows iteration", slog.Any("err", err))
				_, _ = event.Client().Rest.UpdateInteractionResponse(event.ApplicationID(), event.Token(),
					discord.NewMessageUpdate().WithContent("В процессе чтения базы данных произошла ошибка. Часть пользователей могла быть не обработана."))
				return
			}

			_, _ = event.Client().Rest.UpdateInteractionResponse(event.ApplicationID(), event.Token(),
				discord.NewMessageUpdate().WithContent(fmt.Sprintf("Верифицировано %d пользователям из БД.", verifiedCount)))
		}()

	}
}
