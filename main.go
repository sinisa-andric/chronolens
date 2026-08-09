package main

import (
	"chronolens/db"
	logDB "chronolens/log"
	"chronolens/route"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

// TODO dodaj native logove

func main() {

	registry := &logDB.Registry{
		Services: make(map[string]logDB.Service),
	}

	// Registruj servise (primer)
	serviceExample := logDB.Service{
		ID:    "serviceExample",
		Name:  "Service One",
		Type:  "Algorithmia type",
		Event: "Registered",
	}
	registry.Register(serviceExample)

	// baza podataka
	connStr := "user=postgres password=postgres dbname=postgres sslmode=disable"
	database, err := sql.Open("postgres", connStr)
	if err != nil {
		zap.L().Info("opening connection to database failed")
	}
	defer database.Close()

	// proveri da li je konekcija aktivna
	err = database.Ping()
	if err != nil {
		zap.L().Info("connection to database not alive")
	}

	log.Println("Successfully connected to PostgreSQL!")

	err = db.EnsureReportsTable(database)
	if err != nil {
		zap.L().Info("creating reports table failed")
	}

	// preuzmi informacije o servisu
	service, ok := registry.Get("serviceExample")
	if ok {
		fmt.Printf("Service: %v registered ", service.Name)
	} else {
		fmt.Println("Service unregistered succesfully ")
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:3000", "http://127.0.0.1:5173"},
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}))
	router.POST("/report", route.ReportHandler(database))
	router.GET("/results", route.ResultsHandler(database))
	router.GET("/summary", route.SummaryHandler(database))

	log.Println("[Chronolens] Servis pokrenut na :9001")
	router.Run(":9001")
}

func init() {

}
