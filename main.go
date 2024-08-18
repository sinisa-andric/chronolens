package main

import (
	logDB "chronolens/log"
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

// TODO add native logs

func main() {

	registry := &logDB.Registry{
		Services: make(map[string]logDB.Service),
	}

	// Register services (example)
	serviceExample := logDB.Service{
		ID:    "serviceExample",
		Name:  "Service One",
		Type:  "Algorithmia type",
		Event: "Registered",
	}
	registry.Register(serviceExample)

	// database
	connStr := "user=postgres password=postgres dbname=postgres sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		zap.L().Info("opening connection to database failed")
	}
	defer db.Close()

	// Check if the connection is alive
	err = db.Ping()
	if err != nil {
		zap.L().Info("connection to database not alive")
	}

	log.Println("Successfully connected to PostgreSQL!")

	// Get service information (example)
	service, ok := registry.Get("serviceExample")
	if ok {
		fmt.Printf("Service: %v registered ", service.Name)
	} else {
		fmt.Println("Service unregistered succesfully ")
	}
}
