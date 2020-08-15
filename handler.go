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
	output, buttons, _ := glob.genResponse(context, *text, *chatID)

	glob.Bot.SendNew(*chatID, replyID, output, buttons)
}

func (glob *Global) handleCallback(call CallbackQuery) {
	text := call.Data
	callID := call.ID
	chatID := call.Message.Chat.ID
	messageID := call.Message.MessageID

	glob.Bot.ConfirmCallback(*callID, "Okay!")

	context, actionable := glob.detectContext(*chatID, *text)
	if actionable {
		glob.handleAction(*chatID, messageID, context, *text)
	}
	output, buttons, _ := glob.genResponse(context, *text, *chatID)

	glob.Bot.SendEdit(*chatID, *messageID, output, buttons)
}

func (glob *Global) detectContext(chatID int64, text string) (context string, actionable bool) {
	step := glob.getStep(chatID)
	if text == "/start" {
		context = "start"
		return
	}
	if text == "create-room" {
		context = text
		actionable = true
		return
	}
	if text == "enter-room" {
		context = text
		actionable = true
		return
	}
	if len(text) == 3 && step == "1" {
		context = "join-room"
		actionable = true
		return
	}
	if text == "room-found" {
		context = text
		actionable = true
		return
	}
	if text == "exit" {
		context = text
		actionable = true
		return
	}
	if text == "start-choice" {
		context = text
		actionable = true
		return
	}
	if strings.Contains(text, "discard") && step != "2-10" {
		context = "discard"
		actionable = true
		return
	}
	if strings.Contains(text, "like") && step != "2-10" {
		context = "like"
		actionable = true
		return
	}
	if step == "2-10" {
		context = "choice-made"
		actionable = true
		return
	}
	return
}

func (glob *Global) handleAction(chatID int64, messageID *int64, context, text string) {
	switch context {
	case "create-room":
		glob.init(chatID)
		file.WriteLineCSV([]string{
			strconv.FormatInt(chatID, 10),
			"1",
			genRoomNum(),
			"[0,0,0,0,0,0,0,0,0,0]",
		}, glob.File)
	case "enter-room":
		glob.init(chatID)
		// register step 1
		file.WriteLineCSV([]string{
			strconv.FormatInt(chatID, 10),
			"1",
			"",
			"[0,0,0,0,0,0,0,0,0,0]",
		}, glob.File)
	case "join-room":
		if glob.isRoom(text) {
			file.UpdateColCSV(text, 2, strconv.FormatInt(chatID, 10), 0, glob.File)
		}
	case "room-found":
		glob.handleScrape()
	case "start-choice":
		file.UpdateColCSV("2-1", 1, strconv.FormatInt(chatID, 10), 0, glob.File)
	case "discard", "like":
		movieStep, _ := strconv.Atoi(strings.Split(text, "-")[1])
		file.UpdateColCSV("2-"+strconv.Itoa(movieStep+1), 1, strconv.FormatInt(chatID, 10), 0, glob.File)

		choice := glob.getChoice(chatID)
		switch context {
		case "discard":
			choice[movieStep-1] = 0
		case "like":
			choice[movieStep-1] = 1
		}
		choiceStr, _ := json.Marshal(choice)
		file.UpdateColCSV(string(choiceStr), 3, strconv.FormatInt(chatID, 10), 0, glob.File)
	case "choice-made":
		file.UpdateColCSV("3", 1, strconv.FormatInt(chatID, 10), 0, glob.File)
	}
}

func (glob *Global) genResponse(context, text string, chatID int64) (response string, options *[]bot.Button, edit bool) {
	switch context {
	case "start":
		// first clean all past records
		glob.init(chatID)
		response = "Create a room or enter an existing room?"
		options = &[]bot.Button{
			bot.Button{Label: "Create", Value: "create-room"},
			bot.Button{Label: "Enter", Value: "enter-room"},
		}
	case "create-room":
		room := glob.getRoom(chatID)
		if room == "" {
			response = "We could not create a room for you. Try again?"
			options = &[]bot.Button{
				bot.Button{Label: "Try again!", Value: "create-room"},
				bot.Button{Label: "Enter", Value: "enter-room"},
			}
			edit = true
		} else {
			response = "Here is your room number: ```" + room + "```.\nNow share it with your friends."
			options = &[]bot.Button{
				bot.Button{Label: "Done", Value: "room-found"},
			}
		}
	case "enter-room":
		response = "Okay tell me the room number?"
	case "join-room":
		room := glob.getRoom(chatID)
		if room == "" {
			response = "We could not find a room by that number"
			options = &[]bot.Button{
				bot.Button{Label: "Create", Value: "create-room"},
				bot.Button{Label: "Enter", Value: "enter-room"},
			}
			edit = true
		} else {
			response = "Room found!"
			options = &[]bot.Button{
				bot.Button{Label: "Continue", Value: "room-found"},
			}
		}
	case "room-found":
		response = "Now I would show you a few movies. And you would need to say if you want to watch it or not. Alright?"
		options = &[]bot.Button{
			bot.Button{Label: "Cool!", Value: "start-choice"},
			bot.Button{Label: "Meh!", Value: "exit"},
		}
	case "exit":
		response = "All clear! Have fun manually deciding movies 😂"
		options = &[]bot.Button{
			bot.Button{Label: "Start Again", Value: "/start"},
		}
	case "start-choice":
		response = "First movie:\n\n" + glob.Movies[0].Title +
			"\n\n" + glob.Movies[0].Description
		options = &[]bot.Button{
			bot.Button{Label: "Discard", Value: "discard-1"},
			bot.Button{Label: "Like", Value: "like-1"},
		}
	case "discard", "like":
		step := glob.getStep(chatID)
		movieStep := strings.Split(step, "-")[1]
		movieNum, _ := strconv.Atoi(movieStep)
		switch context {
		case "discard":
			response = "Okay! Next:\n\n" + movieStep + ". " + glob.Movies[movieNum].Title +
				"\n\n" + glob.Movies[movieNum].Description
		case "like":
			response = "Let's find more! Next!:\n\n" + movieStep + ". " + glob.Movies[movieNum].Title +
				"\n\n" + glob.Movies[movieNum].Description
		}
		options = &[]bot.Button{
			bot.Button{Label: "Discard", Value: "discard-" + movieStep},
			bot.Button{Label: "Like", Value: "like-" + movieStep},
		}
	case "choice-made":
		response = "Great you are done choosing!"
	default:
		response = "I didn't get you"
	}
	return
}

func (glob *Global) init(chatID int64) {
	// clear previous chats
	file.UpdateLineCSV(nil, glob.File, strconv.FormatInt(chatID, 10), 0)
}

func (glob *Global) isRoom(room string) bool {
	mem, err := file.ReadCSV(glob.File)
	if err != nil {
		return false
	}
	for _, line := range mem {
		if len(line) == 4 && room == line[2] {
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
			break
		}
	}
	return
}

func (glob *Global) getStep(chatID int64) (step string) {
	mem, err := file.ReadCSV(glob.File)
	if err != nil {
		return ""
	}
	for _, line := range mem {
		lineChatID, _ := strconv.ParseInt(line[0], 10, 64)
		if chatID == lineChatID {
			step = line[1]
			break
		}
	}
	return
}

func (glob *Global) getChoice(chatID int64) (choice []int) {
	mem, err := file.ReadCSV(glob.File)
	if err != nil {
		return
	}
	for _, line := range mem {
		lineChatID, _ := strconv.ParseInt(line[0], 10, 64)
		if chatID == lineChatID {
			json.Unmarshal([]byte(line[3]), &choice)
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
