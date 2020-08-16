package bot

import (
	"bytes"
	"io/ioutil"
	"net/http"

	tg "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/pkg/errors"
)

// Button struct
type Button struct {
	Label string
	Value string
}

// Cl is Client
type Cl struct {
	Bot     *tg.BotAPI
	Session *Session
}

// Session is a chat flow
type Session struct {
	Buttons    *[]Button
	CallBackID *string
	ChatID     *int64
	ImageLink  *string
	IsEdit     *bool
	ReplyToID  *int
	SentMsgID  *int
	Text       *string
	Toast      *string
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

// Send the message
func (cl *Cl) Send() (*tg.Message, error) {
	if cl.Session.ChatID != nil && cl.Session.Text != nil {
		if cl.Session.Toast != nil && cl.Session.CallBackID != nil {
			cl.ConfirmCallback(*cl.Session.CallBackID, *cl.Session.Toast)
		}

		if cl.Session.ImageLink != nil && *cl.Session.ImageLink != "" {
			m, err := cl.SendPhoto(*cl.Session.ImageLink, *cl.Session.ChatID, *cl.Session.Text, cl.Session.Buttons)
			if err != nil {
				return nil, err
			}
			return m, nil
		}
		if cl.Session.SentMsgID != nil && cl.Session.IsEdit != nil && *cl.Session.IsEdit {
			m, err := cl.SendEdit(*cl.Session.ChatID, *cl.Session.Text, *cl.Session.SentMsgID, cl.Session.Buttons)
			if err != nil {
				return nil, err
			}
			return m, nil
		}
		m, err := cl.SendNew(*cl.Session.ChatID, *cl.Session.Text, cl.Session.Buttons, cl.Session.ReplyToID)
		if err != nil {
			return nil, err
		}
		return m, nil
	}
	err := errors.Wrap(nil, "Chat ID or Text is missing")
	return nil, err
}

// SendNew sends a new Telegram message
func (cl *Cl) SendNew(chatID int64, text string, option *[]Button, replyToID *int) (*tg.Message, error) {
	msg := tg.NewMessage(chatID, text)

	if option != nil {
		markup := tg.NewInlineKeyboardMarkup(makeButtons(*option))
		msg.ReplyMarkup = &markup
	}
	if replyToID != nil {
		msg.ReplyToMessageID = *replyToID
	}

	msg.ParseMode = "Markdown"
	msg.DisableWebPagePreview = true

	m, err := cl.Bot.Send(msg)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// SendEdit edits an already sent message
func (cl *Cl) SendEdit(chatID int64, text string, sentMsgID int, option *[]Button) (*tg.Message, error) {
	msg := tg.NewEditMessageText(chatID, sentMsgID, text)

	if option != nil {
		markup := tg.NewInlineKeyboardMarkup(makeButtons(*option))
		msg.ReplyMarkup = &markup
	}

	msg.ParseMode = "Markdown"
	msg.DisableWebPagePreview = true

	m, err := cl.Bot.Send(msg)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// ConfirmCallback sends a callback toast
func (cl *Cl) ConfirmCallback(callID, response string) {
	cl.Bot.AnswerCallbackQuery(tg.NewCallback(callID, response))
}

// SendPhoto sends a new Telegram photo
func (cl *Cl) SendPhoto(image string, chatID int64, caption string, options *[]Button) (*tg.Message, error) {
	resp, err := http.Get(image)
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	file := tg.FileReader{Reader: bytes.NewReader(data), Size: int64(len(data))}
	defer resp.Body.Close()

	msg := tg.NewPhotoUpload(chatID, file)
	msg.Caption = caption
	msg.ParseMode = "Markdown"

	if options != nil {
		markup := tg.NewInlineKeyboardMarkup(makeButtons(*options))
		msg.ReplyMarkup = &markup
	}
	m, err := cl.Bot.Send(msg)
	if err != nil {
		return nil, err
	}
	return &m, nil
}
