package main

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/joho/godotenv"
)

var (
	token            string
	guildID          snowflake.ID
	verifiedRoleID   snowflake.ID
	address          string
	port             string
	databaseUrl      string
	minecraftAddress string
	minecraftPort    uint16
	pingInterval     time.Duration
)

//go:embed .env.example
var defaultEnvContent string

func initConfig() {
	err := os.WriteFile(".env", []byte(defaultEnvContent), 0644)
	if err != nil {
		log.Fatalf("Error create .env file: %v", err)
	}

	fmt.Println("Check .env file")
}

// TODO добавить обработку если переменные пусты
func loadConfig() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	token = os.Getenv("TOKEN")
	guildID, err = snowflake.Parse(os.Getenv("GUILD_ID"))
	if err != nil {
		log.Printf("GUILD_ID key error: %v", err)
		return
	}
	verifiedRoleID, err = snowflake.Parse(os.Getenv("VERIFIED_ROLE_ID"))
	if err != nil {
		log.Printf("GUILD_ID key error: %v", err)
		return
	}

	address = os.Getenv("ADDRESS")
	port = os.Getenv("PORT")
	databaseUrl = os.Getenv("DATABASE_URL")
	minecraftAddress = os.Getenv("MINECRAFT_ADDRESS")

	val, err := strconv.ParseUint(os.Getenv("MINECRAFT_PORT"), 10, 16)
	if err != nil {
		log.Fatal("MINECRAFT_PORT key error: %w", err)
	}
	minecraftPort = uint16(val)

	intervalSec, err := strconv.Atoi(os.Getenv("PING_INTERVAL"))
	if err != nil || intervalSec <= 0 {
		log.Fatalln("PING_INTERVAL not set or invalid")
	}

	pingInterval = time.Duration(intervalSec) * time.Second

}
