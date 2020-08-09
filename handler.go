package main

import (
	"bytes"
	"encoding/json"

	"github.com/Donnie/PickFlick/bot"
	"github.com/gin-gonic/gin"
)

func (glob *Global) handleHook(c *gin.Context) {
	buf := new(bytes.Buffer)
	buf.ReadFrom(c.Request.Body)
	str := buf.String()

	var input Input

	err := json.Unmarshal([]byte(str), &input)
	check(err)

	if input.Message != nil && input.Message.Text != nil {
		glob.handleMessage(*input.Message)
	}

	if input.CallbackQuery != nil && input.CallbackQuery.Data != nil {
		glob.handleCallback(*input.CallbackQuery)
	}

	c.JSON(200, nil)
}

func (glob *Global) handleMessage(msg Message) {
	text := msg.Text
	chatID := msg.Chat.ID
	replyID := msg.MessageID

	glob.Bot.SendNew(*chatID, replyID, *text, &[]bot.Button{
		bot.Button{Label: "yes", Value: "yes"},
		bot.Button{Label: "no", Value: "no"},
	})
}

func (glob *Global) handleCallback(call CallbackQuery) {
	text := call.Data
	callID := call.ID
	chatID := call.Message.Chat.ID
	messageID := call.Message.MessageID

	glob.Bot.ConfirmCallback(*callID)

	glob.Bot.SendEdit(*chatID, *messageID, *text, &[]bot.Button{
		bot.Button{Label: "yes", Value: "yes"},
		bot.Button{Label: "no", Value: "no"},
	})
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
