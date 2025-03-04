package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"tailscale.com/ipn/store/mem"
	"tailscale.com/tsnet"

	"github.com/joho/godotenv"
	"slices"
)

type ListFlag []string

func (i *ListFlag) String() string {
	return fmt.Sprintf("%v", *i)
}

func (i *ListFlag) Set(value string) error {
	*i = append(*i, value)
	return nil
}

var tags ListFlag

func main() {
	envfile := flag.String("env", ".env", "Environment file to load")
	flag.Var((&tags), "tag", "Tag for this client")
	flag.Parse()

	err := godotenv.Load(*envfile)
	if err != nil {
		log.Fatal("Error loading .env file", err)
	}

	authKey := os.Getenv("TS_AUTHKEY")
	controlURL := os.Getenv("TS_CONTROL_URL")

	log.Println("hostname: ", "tsls")
	log.Println("control Server URL: ", controlURL)
	log.Println("Auth Key: ", string([]rune(authKey)[:8])+"...")
	log.Println("Filter Tags: ", tags)

	// Create memory store
	store, err := mem.New(log.Printf, "")
	if err != nil {
		log.Fatalf("Failed to create store: %v", err)
	}

	// Create a new tsnet Server
	server := &tsnet.Server{
		Hostname:   "tsls",
		AuthKey:    os.Getenv("TS_AUTHKEY"),
		Store:      store,
		Ephemeral:  true,
		ControlURL: os.Getenv("TS_CONTROL_URL"),
	}

	// Start the server
	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start tsnet server: %v", err)
	}
	defer server.Close()

	// Get a local client
	client, err := server.LocalClient()
	if err != nil {
		log.Fatalf("Failed to get local client: %v", err)
	}

	// Get the status
	status, err := client.Status(context.Background())
	if err != nil {
		log.Fatalf("Failed to get status: %v", err)
	}

	// Wait for backend to start
	for status.BackendState != "Running" {
		log.Println("Waiting for backend to start... Current state: ", status.BackendState)
		time.Sleep(1 * time.Second)
		status, err = client.Status(context.Background())
		if err != nil {
			log.Fatalf("Failed to get status: %v", err)
		}
	}

	log.Println("Found Peers:")

	for id, peer := range status.Peer {
		peerTags := []string{}
		if peer.Tags != nil {
			peerTags = peer.Tags.AsSlice()
		}
		if len(tags) == 0 || hasMatchingTag(peerTags, tags) {
			log.Println("id:", id, "Name:", peer.HostName)
			for _, ip := range peer.TailscaleIPs {
				fmt.Printf("%s ", ip)
			}
		}
	}
}

func hasMatchingTag(peerTags []string, filterTags []string) bool {
	if peerTags == nil {
		return false
	}

	for _, ftag := range filterTags {
		if slices.Contains(peerTags, ftag) {
			return true
		}
	}

	return false
}
