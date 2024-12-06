package main

import (
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	tele "gopkg.in/telebot.v3"
)

var bot *tele.Bot

func main() {
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true, TimestampFormat: DateTimeFormat})

	InitSqlite()
	InitBot()
	InitCommandHandler()
	InitChacha20()
	InitXrayStats()
	InitXrayLog()

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
		Token:     os.Getenv("TOKEN"),
		Poller:    &tele.LongPoller{Timeout: 10 * time.Second},
		ParseMode: tele.ModeMarkdownV2,
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		log.Fatal(err)
		return
	}
	bot = b
}
