package main

import (
	"fmt"
	tele "gopkg.in/telebot.v3"
	"regexp"
	"time"
)

func IsCommand(message *tele.Message) bool {
	for _, entity := range message.Entities {
		if entity.Type == tele.EntityCommand {
			return true
		}
	}
	return false
}

func ReplaceForMarkdownV2(input string) string {
	if input == "" {
		return input
	}

	re := regexp.MustCompile("[_*\\[\\]()~`>#+=\\-|{}.!]")
	return re.ReplaceAllString(input, `\$0`)
}

func GetDuration(date time.Time) string {
	now := time.Now()

	// 计算时间差
	duration := date.Sub(now)
	diffDays := int(duration.Hours()) / 24
	diffHours := int(duration.Hours()) % 24
	diffMinutes := int(duration.Minutes()) % 60

	if diffDays != 0 {
		return fmt.Sprintf("%d天%d小时%d分钟", diffDays, diffHours, diffMinutes)
	} else if diffHours != 0 {
		return fmt.Sprintf("%d小时%d分钟", diffHours, diffMinutes)
	} else {
		return fmt.Sprintf("%d分钟", diffMinutes)
	}
}
