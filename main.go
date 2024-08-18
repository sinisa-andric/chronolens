package main

import (
	log "chronolens/log"
	"fmt"
)

// TODO add native logs

func main() {

	registry := &log.Registry{
		Services: make(map[string]log.Service),
	}

	// Register services (example)
	serviceExample := log.Service{
		ID:    "serviceExample",
		Name:  "Service One",
		Type:  "Algorithmia type",
		Event: "Registered",
	}
	registry.Register(serviceExample)

	// TODO add database

	// Get service information (example)
	service, ok := registry.Get("serviceExample")
	if ok {
		fmt.Printf("Service: %v registered ", service.Name)
	} else {
		fmt.Println("Service unregistered succesfully ")
	}
}
