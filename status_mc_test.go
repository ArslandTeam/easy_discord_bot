package main

import (
	"fmt"
	"testing"
)

func TestPingMinecraftJavaServer(t *testing.T) {
	loadConfig()
	status, err := pingMinecraftJavaServer()

	if err != nil {
		t.Fatalf("%v", err)
	}

	fmt.Printf("Online %d/%d", status.OnlinePlayers, status.MaxPlayers)
}
