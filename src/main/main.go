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

	//client.Log(true)

	if err := client.InitLongPoll(0, 2); err != nil {
		log.Panic(err)
	}

	updates, _, err := client.GetLPUpdatesChan(100, vkapi.LPConfig{25, vkapi.LPModeAttachments})
	if err != nil {
		log.Panic(err)
	}

	go InitConnectionPool()

	for update := range updates {
		if update.Message == nil || !update.IsNewMessage() || update.Message.Outbox() {
			continue
		}

		command := strings.ToLower(update.Message.Text)

		if strings.HasPrefix(command, "start") {
			go StartCommand(update)
			continue
		}

		if strings.HasPrefix(command, "help") {
			go HelpCommand(update)
			continue
		}

		if strings.HasPrefix(command, "save") {
			go SaveCommand(update)
			continue
		}

		if strings.HasPrefix(command, "full") {
			go FullCommand(update)
			continue
		}

		if strings.HasPrefix(command, "today") {
			go TodayCommand(update)
			continue
		}

		if strings.HasPrefix(command, "tomorrow") {
			go TomorrowCommand(update)
			continue
		}

		if strings.HasPrefix(command, "get") {
			go GetCommand(update)
			continue
		}

		if strings.HasPrefix(command, "status") {
			go StatusCommand(update)
			continue
		}

		if strings.HasPrefix(command, "update") {
			go UpdateCommand(update)
			continue
		}

		if strings.HasPrefix(command, "delete") {
			go DeleteCommand(update)
			continue
		}

		go AnythingCommand(update)
	}
}
