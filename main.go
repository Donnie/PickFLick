package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/Donnie/PickFlick/bot"
	"github.com/Donnie/PickFlick/file"
	"github.com/gin-gonic/gin"
	tg "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/joho/godotenv"
)

func init() {
	if _, err := os.Stat(".env.local"); os.IsNotExist(err) {
		godotenv.Load(".env")
	} else {
		godotenv.Load(".env.local")
	}

	logFile, exists := os.LookupEnv("LOG")
	if !exists {
		fmt.Println("Add LOG to .env file")
		os.Exit(1)
	}

	file.CreatePath(logFile)
	f, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Error opening log file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)

	log.Println("Running for " + os.Getenv("ENV"))
}

func main() {
	teleToken, exists := os.LookupEnv("TELEGRAM_TOKEN")
	if !exists {
		log.Println("Add TELEGRAM_TOKEN to .env file")
		os.Exit(1)
	}
	dbFile, exists := os.LookupEnv("DBFILE")
	if !exists {
		log.Println("Add DBFILE to .env file")
		os.Exit(1)
	}

	tgbot, err := tg.NewBotAPI(teleToken)
	if err != nil {
		log.Panic(err)
	}
	tgbot.Debug, _ = strconv.ParseBool(os.Getenv("DEBUG"))

	glob := Global{
		Bot: bot.Cl{
			Bot: tgbot,
		},
		File: dbFile,
	}

	go glob.poll()

	r := gin.Default()
	r.GET("/scrape", func(c *gin.Context) {
		glob.handleScrape()
		c.JSON(200, nil)
	})
	r.POST("/hook", glob.handleHook)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, nil)
	})
	r.Run()
}
