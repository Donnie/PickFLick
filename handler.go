package main

import (
	"bytes"
	"encoding/json"

	"github.com/gin-gonic/gin"
	tg "github.com/go-telegram-bot-api/telegram-bot-api"
)

func (glob *Global) handleHook(c *gin.Context) {
	buf := new(bytes.Buffer)
	buf.ReadFrom(c.Request.Body)
	str := buf.String()

	var input Input

	err := json.Unmarshal([]byte(str), &input)
	check(err)

	msg := *input.Message.Text
	glob.sendMessage(*input.Message.Chat.ID, msg, input.Message.MessageID)

	c.JSON(200, nil)
}

func (glob *Global) sendMessage(chatID int64, text string, messageID *int64) {
	msg := tg.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.DisableWebPagePreview = true

	if messageID != nil {
		msg.ReplyToMessageID = int(*messageID)
	}
	glob.Bot.Send(msg)
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
