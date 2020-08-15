package main

import (
	"bytes"
	"encoding/json"
	"fmt"
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

	toasts := []string{"Okay!", "Cool!", "Alright!", "Fine!", "Hmmm!"}
	glob.Bot.ConfirmCallback(*callID, toasts[randInt(0, 4)])

	context, actionable := glob.detectContext(*chatID, *text)
	if actionable {
		glob.handleAction(*chatID, messageID, context, *text)
	}
	output, buttons, edit := glob.genResponse(context, *text, *chatID)
	if edit {
		glob.Bot.SendEdit(*chatID, *messageID, output, buttons)
	} else {
		glob.Bot.SendNew(*chatID, nil, output, buttons)
	}
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
	if text == "show-result" && step == "3" {
		context = "show-result"
		return
	}
	if text == "end" && step == "3" {
		context = "end"
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
			file.UpdateColsCSV(text, 2, strconv.FormatInt(chatID, 10), 0, glob.File)
		}
	case "room-found":
		glob.handleScrape()
	case "start-choice":
		file.UpdateColsCSV("2-1", 1, strconv.FormatInt(chatID, 10), 0, glob.File)
	case "discard", "like":
		movieStep, _ := strconv.Atoi(strings.Split(text, "-")[1])
		file.UpdateColsCSV("2-"+strconv.Itoa(movieStep+1), 1, strconv.FormatInt(chatID, 10), 0, glob.File)

		choice := glob.getChoice(chatID)
		switch context {
		case "discard":
			choice[movieStep-1] = 0
		case "like":
			choice[movieStep-1] = 1
		}
		choiceStr, _ := json.Marshal(choice)
		file.UpdateColsCSV(string(choiceStr), 3, strconv.FormatInt(chatID, 10), 0, glob.File)
	case "choice-made":
		file.UpdateColsCSV("3", 1, strconv.FormatInt(chatID, 10), 0, glob.File)
	}
}

func (glob *Global) genResponse(context, text string, chatID int64) (response string, options *[]bot.Button, edit bool) {
	room := glob.getRoom(chatID)

	switch context {
	case "start":
		// first clean all past records
		glob.init(chatID)
		response = "Create a room or enter an existing room?"
		options = &[]bot.Button{
			bot.Button{Label: "Create", Value: "create-room"},
			bot.Button{Label: "Enter", Value: "enter-room"},
		}
		edit = true
	case "create-room":
		if room == "" {
			response = "We could not create a room for you. Try again?"
			options = &[]bot.Button{
				bot.Button{Label: "Try again!", Value: "create-room"},
				bot.Button{Label: "Enter", Value: "enter-room"},
			}
		} else {
			response = "Here is your room number: ```" + room + "```.\nNow share it with your friends."
			options = &[]bot.Button{
				bot.Button{Label: "Done", Value: "room-found"},
			}
		}
		edit = true
	case "enter-room":
		response = "Okay tell me the room number?"
		edit = true
	case "join-room":
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
		response = "Now I would show you top ten movies this week in Berlin."
		options = &[]bot.Button{
			bot.Button{Label: "Meh!", Value: "exit"},
			bot.Button{Label: "Cool!", Value: "start-choice"},
		}
		edit = true
	case "exit":
		response = "All clear! Have fun manually deciding movies 😂"
		options = &[]bot.Button{
			bot.Button{Label: "Start Again", Value: "/start"},
		}
	case "start-choice":
		response = "First movie:\n\n" + glob.Movies[0].Title +
			"\n\n" + glob.Movies[0].Description
		options = &[]bot.Button{
			bot.Button{Label: "👎", Value: "discard-1"},
			bot.Button{Label: "👍", Value: "like-1"},
		}
		edit = true
	case "discard", "like":
		step := glob.getStep(chatID)
		movieStep := strings.Split(step, "-")[1]
		movieNum, _ := strconv.Atoi(movieStep)
		switch context {
		case "discard":
			response = "Okay! Next:\n\n" + movieStep + ". " + glob.Movies[movieNum-1].Title +
				"\n\n" + glob.Movies[movieNum].Description
		case "like":
			response = "Let's find more! Next!:\n\n" + movieStep + ". " + glob.Movies[movieNum-1].Title +
				"\n\n" + glob.Movies[movieNum].Description
		}
		options = &[]bot.Button{
			bot.Button{Label: "👎", Value: "discard-" + movieStep},
			bot.Button{Label: "👍", Value: "like-" + movieStep},
		}
		edit = true
	case "choice-made":
		response = "Great you are done choosing!"
		options = &[]bot.Button{
			bot.Button{Label: "Results?", Value: "show-result"},
			bot.Button{Label: "Choose Again", Value: "start-choice"},
		}
		edit = true
	case "show-result":
		mergedChoice := mergeChoices(glob.getChoices(room))
		movieList := glob.getMovieList(mergedChoice)

		if len(movieList) > 0 {
			response = "So you have chosen:\n\n"
			for i, movie := range movieList {
				response = response + fmt.Sprintf("%d. [%s](%s)\n", i+1, movie.Title, movie.Link)
			}
		} else {
			response = "Sorry! You do not have any common options.\nRecommended number of choices is six."
		}
		response = response + "\n\nYou can try results again when your friends finish."

		options = &[]bot.Button{
			bot.Button{Label: "Results?", Value: "show-result"},
			bot.Button{Label: "Choose Again", Value: "start-choice"},
			bot.Button{Label: "Exit", Value: "end"},
		}
		edit = true
	case "end":
		glob.init(chatID)
		response = "Create a room or enter an existing room?"
		options = &[]bot.Button{
			bot.Button{Label: "Create", Value: "create-room"},
			bot.Button{Label: "Enter", Value: "enter-room"},
		}
	default:
		response = "I didn't get you"
	}
	return
}

func (glob *Global) init(chatID int64) {
	// clear previous chats
	file.UpdateLinesCSV(nil, glob.File, strconv.FormatInt(chatID, 10), 0)
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

func (glob *Global) getChoices(roomID string) (choices [][]int) {
	mem, err := file.ReadCSV(glob.File)
	if err != nil {
		return
	}
	for _, line := range mem {
		choice := []int{}
		if roomID == line[2] {
			json.Unmarshal([]byte(line[3]), &choice)
			choices = append(choices, choice)
		}
	}
	return
}

func mergeChoices(choices [][]int) (merged []int) {
	merged = []int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	for i := range choices[0] {
		crossSec := getCrossSection(choices, i)
		if !bothValues(crossSec, 0, 1) {
			merged[i] = crossSec[0]
		}
	}
	return
}

func getCrossSection(matrix [][]int, col int) (crossSec []int) {
	for i := range matrix {
		crossSec = append(crossSec, matrix[i][col])
	}
	return
}

func bothValues(array []int, value1, value2 int) (bo bool) {
	bo = strings.Contains(fmt.Sprintf("%v", array), fmt.Sprintf("%d", value1)) &&
		strings.Contains(fmt.Sprintf("%v", array), fmt.Sprintf("%d", value2))
	return
}

func (glob *Global) getMovieList(choice []int) (movies []scraper.Movie) {
	for i, ch := range choice {
		if ch == 1 {
			movies = append(movies, glob.Movies[i])
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

func randInt(min, max int) int {
	rand.Seed(time.Now().Unix())
	return min + rand.Intn(max-min+1)
}
