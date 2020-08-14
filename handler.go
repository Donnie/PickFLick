package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/Donnie/PickFlick/bot"
	"github.com/Donnie/PickFlick/file"
	"github.com/Donnie/PickFlick/scraper"
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

	context, actionable := glob.detectContext(*chatID, *text)
	if actionable {
		glob.handleAction(*chatID, replyID, context, *text)
	}
	output, buttons := glob.genResponse(context, *text, *chatID)

	glob.Bot.SendNew(*chatID, replyID, output, buttons)
}

func (glob *Global) handleCallback(call CallbackQuery) {
	text := call.Data
	callID := call.ID
	chatID := call.Message.Chat.ID
	messageID := call.Message.MessageID

	glob.Bot.ConfirmCallback(*callID)

	context, actionable := glob.detectContext(*chatID, *text)
	if actionable {
		glob.handleAction(*chatID, messageID, context, *text)
	}
	output, buttons := glob.genResponse(context, *text, *chatID)

	glob.Bot.SendEdit(*chatID, *messageID, output, buttons)
}

func (glob *Global) detectContext(chatID int64, text string) (context string, actionable bool) {
	step := glob.getStep(chatID)
	if strings.Contains(text, "/start") {
		context = "start"
	}
	if strings.Contains(text, "create-room") {
		context = text
		actionable = true
	}
	if strings.Contains(text, "enter-room") {
		context = text
		actionable = true
	}
	if len(text) == 3 && step == 1 {
		context = "join-room"
		actionable = true
	}
	return
}

func (glob *Global) genResponse(context, text string, chatID int64) (response string, options *[]bot.Button) {
	switch context {
	case "start":
		// first clean all past records
		file.UpdateLineCSV(nil, glob.File, strconv.FormatInt(chatID, 10), 0)

		response = "Create a room or enter an existing room?"
		options = &[]bot.Button{
			bot.Button{Label: "Create", Value: "create-room"},
			bot.Button{Label: "Enter", Value: "enter-room"},
		}
	case "create-room":
		room := glob.getRoom(chatID)
		if room == "" {
			response = "We could not create a room for you"
		} else {
			response = "Here is your room number: ```" + room + "```.\nNow share it with your friends."
		}
	case "enter-room":
		response = "Okay tell me the room number?"
	case "join-room":
		room := glob.getRoom(chatID)
		if room == "" {
			response = "We could not find a room by that number"
		} else {
			response = "Room found!"
		}
	default:
		response = "I still don't understand you"
	}
	return
}

func (glob *Global) handleAction(chatID int64, messageID *int64, context, text string) {
	switch context {
	case "create-room":
		file.WriteLineCSV([]string{
			strconv.FormatInt(chatID, 10),
			"1",
			genRoomNum(),
		}, glob.File)
	case "enter-room":
		// register step 1
		file.WriteLineCSV([]string{
			strconv.FormatInt(chatID, 10),
			"1",
			"",
		}, glob.File)
	case "join-room":
		if glob.isRoom(text) {
			file.UpdateLineCSV([]string{
				strconv.FormatInt(chatID, 10),
				"1",
				text,
			}, glob.File, strconv.FormatInt(chatID, 10), 0)
		}
	}
}

func (glob *Global) isRoom(room string) bool {
	mem, err := file.ReadCSV(glob.File)
	if err != nil {
		return false
	}
	for _, line := range mem {
		if room == line[2] {
			return true
		}
	}
	return false
}

func (glob *Global) getRoom(chatID int64) (room string) {
	mem, err := file.ReadCSV(glob.File)
	if err != nil {
		return ""
	}
	for _, line := range mem {
		lineChatID, _ := strconv.ParseInt(line[0], 10, 64)
		if chatID == lineChatID {
			room = line[2]
			// get the last room id
		}
	}
	return
}

func (glob *Global) getStep(chatID int64) (step int) {
	mem, err := file.ReadCSV(glob.File)
	if err != nil {
		return 0
	}
	for _, line := range mem {
		lineChatID, _ := strconv.ParseInt(line[0], 10, 64)
		if chatID == lineChatID {
			step, _ = strconv.Atoi(line[1])
			break
		}
	}
	return
}

func (glob *Global) handleScrape() {
	layout := "2006-01-02.json"
	filename := time.Now().Format(layout)

	file, err := os.Open(filename)
	if err != nil {
		scraper.Save(filename)
		file, err = os.Open(filename)
		check(err)
	}
	defer file.Close()

	var movies []scraper.Movie
	jsonBytes, _ := ioutil.ReadAll(file)
	json.Unmarshal(jsonBytes, &movies)

	glob.Movies = movies
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func genRoomNum() string {
	n := 3
	b := make([]byte, n)
	var src = rand.NewSource(time.Now().UnixNano())
	const letterBytes = "abcdefghijkmnopqrstuvwxyz023456789"
	const (
		letterIdxBits = 6                    // 6 bits to represent a letter index
		letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
		letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
	)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return *(*string)(unsafe.Pointer(&b))
}
