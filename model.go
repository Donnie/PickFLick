package main

import (
	"github.com/Donnie/PickFlick/bot"
	"github.com/Donnie/PickFlick/scraper"
)

// Global holds fundamental items
type Global struct {
	Bot    bot.Cl
	File   string
	Movies []scraper.Movie
}

// Input struct
type Input struct {
	UpdateID      *int64         `json:"update_id"`
	CallbackQuery *CallbackQuery `json:"callback_query"`
	Message       *Message       `json:"message"`
}

// Message struct
type Message struct {
	MessageID      *int64       `json:"message_id"`
	From           *From        `json:"from"`
	Chat           *Chat        `json:"chat"`
	Date           *int64       `json:"date"`
	ReplyToMessage *Message     `json:"reply_to_message"`
	ReplyMarkup    *ReplyMarkup `json:"reply_markup"`
	Text           *string      `json:"text"`
}

// From struct
type From struct {
	ID           *int64  `json:"id"`
	IsBot        *bool   `json:"is_bot"`
	FirstName    *string `json:"first_name"`
	LastName     *string `json:"last_name"`
	Username     *string `json:"username"`
	LanguageCode *string `json:"language_code"`
}

// Chat struct
type Chat struct {
	ID        *int64  `json:"id"`
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
	Username  *string `json:"username"`
	Type      *string `json:"type"`
}

// CallbackQuery struct
type CallbackQuery struct {
	ID           *string  `json:"id"`
	From         *From    `json:"from"`
	Message      *Message `json:"message"`
	ChatInstance *string  `json:"chat_instance"`
	Data         *string  `json:"data"`
}

// ReplyMarkup struct
type ReplyMarkup struct {
	InlineKeyboard *[][]InlineKeyboard `json:"inline_keyboard"`
}

// InlineKeyboard struct
type InlineKeyboard struct {
	Text         *string `json:"text"`
	CallbackData *string `json:"callback_data"`
}
