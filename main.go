package main

import (
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
	tele "gopkg.in/telebot.v3"
)

var bot *tele.Bot

func main() {
	InitSqlite()
	InitBot()
	InitCommandHandler()

	for command := range commandHandlers {
		commandHandler := commandHandlers[command]
		bot.Handle(commandHandler.command, commandHandler.handler)
	}
	bot.Handle(tele.OnText, TextHandler)

	log.Info("Telegram Bot 已启动")
	bot.Start()
}

func InitBot() {
	pref := tele.Settings{
		Token:  os.Getenv("TOKEN"),
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		log.Fatal(err)
		return
	}
	bot = b
}
