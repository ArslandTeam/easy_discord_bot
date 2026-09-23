package main

import (
	"errors"
	"log"
	"os"

	"github.com/andre-carbajal/go-mcstatus"
)

type ResponseServerStatus struct {
	IsOnline      bool
	MaxPlayers    int
	OnlinePlayers int
}

func pingMinecraftJavaServer() (ResponseServerStatus, error) {
	server, err := mcstatus.NewJavaServer(os.Getenv("MINECRAFT_IP"))
	if err != nil {
		log.Fatal(err)
	}

	status, err := server.Status()
	if err != nil {
		log.Fatal(err)
	}

	resp, ok := status.(*mcstatus.JavaStatusResponse)

	if !ok {
		return ResponseServerStatus{
			IsOnline: false,
		}, errors.New("Error type response server")
	}

	return ResponseServerStatus{
		IsOnline:      true,
		MaxPlayers:    resp.Players.Max,
		OnlinePlayers: resp.Players.Online,
	}, nil
}
