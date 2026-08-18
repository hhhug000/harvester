package main

import (
	"log"

	"github.com/hhhug000/harvester/crawler"
	"github.com/hhhug000/harvester/server"
)

func main() {
	crawlerEngine := crawler.NewEngine("./output")

	srv, err := server.NewServer("./templates", crawlerEngine)
	if err != nil {
		log.Fatalf("Failed to initialize server: %v", err)
	}

	log.Println("Server listening on port 8080")
	log.Fatal(srv.Start(":8080"))
}
