package bot

import (
	"bytes"
	"io/ioutil"
	"net/http"

	tg "github.com/go-telegram-bot-api/telegram-bot-api"
)

// Button struct
type Button struct {
	Label string
	Value string
}

// Cl is Client
type Cl struct {
	Bot *tg.BotAPI
}

func makeButtons(keyb []Button) (keyboard []tg.InlineKeyboardButton) {
	for _, key := range keyb {
		val := key.Value
		keyboard = append(keyboard, tg.InlineKeyboardButton{
			Text:         key.Label,
			CallbackData: &val,
		})
	}
	return
}

// SendNew sends a new Telegram message
func (cl *Cl) SendNew(chatID int64, replyToID *int64, text string, buttons *[]Button) (m tg.Message) {
	msg := tg.NewMessage(chatID, text)

	if buttons != nil {
		markup := tg.NewInlineKeyboardMarkup(makeButtons(*buttons))
		msg.ReplyMarkup = &markup
	}
	if replyToID != nil {
		msg.ReplyToMessageID = int(*replyToID)
	}

	msg.ParseMode = "Markdown"
	msg.DisableWebPagePreview = true

	m, _ = cl.Bot.Send(msg)
	return
}

// SendEdit edits an already sent message
func (cl *Cl) SendEdit(chatID, messageID int64, text string, buttons *[]Button) (m tg.Message) {
	msg := tg.NewEditMessageText(chatID, int(messageID), text)

	if buttons != nil {
		markup := tg.NewInlineKeyboardMarkup(makeButtons(*buttons))
		msg.ReplyMarkup = &markup
	}

	msg.ParseMode = "Markdown"
	msg.DisableWebPagePreview = true

	m, _ = cl.Bot.Send(msg)
	return
}

// ConfirmCallback sends a callback toast
func (cl *Cl) ConfirmCallback(callID, response string) {
	cl.Bot.AnswerCallbackQuery(tg.NewCallback(callID, response))
}

// SendPhoto sends a new Telegram photo
func (cl *Cl) SendPhoto(chatID int64, photolink string, text string, buttons *[]Button) (m tg.Message) {
	resp, _ := http.Get(photolink)
	data, _ := ioutil.ReadAll(resp.Body)
	file := tg.FileReader{Name: "Name", Reader: bytes.NewReader(data), Size: int64(len(data))}
	defer resp.Body.Close()

	msg := tg.NewPhotoUpload(chatID, file)
	msg.Caption = text
	msg.ParseMode = "Markdown"

	if buttons != nil {
		markup := tg.NewInlineKeyboardMarkup(makeButtons(*buttons))
		msg.ReplyMarkup = &markup
	}

	m, _ = cl.Bot.Send(msg)
	return
}
