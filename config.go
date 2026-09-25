package main

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

var (
	token            string
	guildID          string
	address          string
	port             string
	minecraftAddress string
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
	guildID = os.Getenv("GUILD_ID")
	address = os.Getenv("ADDRESS")
	port = os.Getenv("PORT")
	minecraftAddress = os.Getenv("MINECRAFT_ADDRESS")

	intervalSec, err := strconv.Atoi(os.Getenv("PING_INTERVAL"))
	if err != nil || intervalSec <= 0 {
		log.Fatalln("PING_INTERVAL not set or invalid")
	} else {
		pingInterval = time.Duration(intervalSec) * time.Second
	}
}
