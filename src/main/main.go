package main

import (
	"github.com/Dimonchik0036/vk-api"
	r "gopkg.in/gorethink/gorethink.v3"
	"log"
	"strings"
)

const (
	dbUrl = "krasovsky.me"
)

var (
	client  *vkapi.Client
	session *r.Session
)

func main() {
	var err error
	client, err = vkapi.NewClientFromToken("b6f810fd39fbc0d2f562384f58e78ac8ffbae986f97267e1f116474396e61ce8afd260a9f96a4bf97ac4b")
	if err != nil {
		log.Panic(err)
	}

	client.Log(true)

	if err := client.InitLongPoll(0, 2); err != nil {
		log.Panic(err)
	}

	updates, _, err := client.GetLPUpdatesChan(100, vkapi.LPConfig{25, vkapi.LPModeAttachments})
	if err != nil {
		log.Panic(err)
	}

	go InitConnectionPool()

	for update := range updates {
		if update.Message == nil {
			continue
		}

		if update.IsNewMessage() {
			if strings.HasPrefix(update.Message.Text, "/start") {
				go StartCommand(update)
			}

			if strings.HasPrefix(update.Message.Text, "/help") {
				go HelpCommand(update)
			}

			if strings.HasPrefix(update.Message.Text, "/save") {
				go SaveCommand(update)
			}

			if strings.HasPrefix(update.Message.Text, "/full") {
				go FullCommand(update)
			}

			if strings.HasPrefix(update.Message.Text, "/today") {
				go TodayCommand(update)
			}

			if strings.HasPrefix(update.Message.Text, "/tomorrow") {
				go TomorrowCommand(update)
			}

			if strings.HasPrefix(update.Message.Text, "/get") {
				go GetCommand(update)
			}

			if strings.HasPrefix(update.Message.Text, "/status") {
				go StatusCommand(update)
			}

			if strings.HasPrefix(update.Message.Text, "/update") {
				go UpdateCommand(update)
			}

			if strings.HasPrefix(update.Message.Text, "/delete") {
				go DeleteCommand(update)
			}
		}
	}
}
