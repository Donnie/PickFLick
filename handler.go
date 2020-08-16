package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
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

	var input Input

	err := json.Unmarshal(buf.Bytes(), &input)
	check(err)

	glob.Context = new(Context)
	glob.Response = new(Response)
	glob.Bot.Session = new(bot.Session)

	if input.Message != nil && input.Message.Text != nil {
		glob.handleMessage(*input.Message)
	} else if input.CallbackQuery != nil && input.CallbackQuery.Data != nil {
		glob.handleCallback(*input.CallbackQuery)
	}

	c.JSON(200, nil)
}

func (glob *Global) handleMessage(msg Message) {
	replyToID := int(*msg.MessageID)

	glob.Bot.Session.ChatID = msg.Chat.ID
	glob.Bot.Session.ReplyToID = &replyToID
	glob.Context.ChatID = *msg.Chat.ID
	glob.Context.Text = *msg.Text

	glob.handleContext()
	glob.handleAction()
	glob.handleResponse()
	glob.Bot.Send()
}

func (glob *Global) handleCallback(call CallbackQuery) {
	sentMsgID := int(*call.Message.MessageID)

	glob.Bot.Session.CallBackID = call.ID
	glob.Bot.Session.ChatID = call.Message.Chat.ID
	glob.Bot.Session.SentMsgID = &sentMsgID
	glob.Context.ChatID = *call.Message.Chat.ID
	glob.Context.Text = *call.Data

	toasts := []string{"Okay!", "Cool!", "Alright!", "Fine!", "Hmmm!"}
	glob.Bot.Session.Toast = &toasts[randInt(0, 4)]

	glob.handleContext()
	glob.handleAction()
	glob.handleResponse()
	glob.Bot.Send()
}

func (glob *Global) handleContext() {
	glob.Context.Step = glob.getStep(glob.Context.ChatID)
	if glob.Context.Text == "/start" {
		glob.Context.Meaning = "start"
		return
	}
	if glob.Context.Text == "/about" {
		glob.Context.Meaning = "about"
		return
	}
	if glob.Context.Text == "create-room" ||
		glob.Context.Text == "enter-room" ||
		glob.Context.Text == "exit" ||
		glob.Context.Text == "room-found" ||
		glob.Context.Text == "start-choice" {
		glob.Context.Meaning = glob.Context.Text
		glob.Context.Actionable = true
		return
	}
	if len(glob.Context.Text) == 3 && glob.Context.Step == "1" {
		glob.Context.Meaning = "join-room"
		glob.Context.Actionable = true
		return
	}
	if strings.Contains(glob.Context.Text, "discard") &&
		glob.Context.Step != "2-10" {
		glob.Context.Meaning = "discard"
		glob.Context.Actionable = true
		return
	}
	if strings.Contains(glob.Context.Text, "like") &&
		glob.Context.Step != "2-10" {
		glob.Context.Meaning = "like"
		glob.Context.Actionable = true
		return
	}
	if glob.Context.Step == "2-10" || glob.Context.Text == "choice-made" {
		glob.Context.Meaning = "choice-made"
		glob.Context.Actionable = true
		return
	}
	if glob.Context.Text == "show-result" && glob.Context.Step == "3" {
		glob.Context.Meaning = "show-result"
		return
	}
	if glob.Context.Text == "end" && glob.Context.Step == "3" {
		glob.Context.Meaning = "end"
		glob.Context.Actionable = true
		return
	}
}

func (glob *Global) handleAction() {
	if glob.Context.Actionable {
		switch glob.Context.Meaning {

		case "create-room":
			glob.init(glob.Context.ChatID)
			glob.Context.Step = "1"
			file.WriteLineCSV([]string{
				strconv.FormatInt(glob.Context.ChatID, 10),
				glob.Context.Step,
				genRoomNum(),
				"[0,0,0,0,0,0,0,0,0,0]",
			}, glob.File)

		case "enter-room":
			glob.init(glob.Context.ChatID)
			// register step 1
			glob.Context.Step = "1"
			file.WriteLineCSV([]string{
				strconv.FormatInt(glob.Context.ChatID, 10),
				glob.Context.Step,
				"",
				"[0,0,0,0,0,0,0,0,0,0]",
			}, glob.File)

		case "join-room":
			if glob.isRoom(glob.Context.Text) {
				file.UpdateColsCSV(glob.Context.Text, 2, strconv.FormatInt(glob.Context.ChatID, 10), 0, glob.File)
			}

		case "room-found":
			glob.handleScrape()

		case "start-choice":
			glob.Context.Step = "2-1"
			file.UpdateColsCSV(glob.Context.Step, 1, strconv.FormatInt(glob.Context.ChatID, 10), 0, glob.File)

		case "discard", "like":
			lastStep, _ := strconv.Atoi(strings.Split(glob.Context.Text, "-")[1])
			glob.Context.Step = "2-" + strconv.Itoa(lastStep+1)
			file.UpdateColsCSV(glob.Context.Step, 1, strconv.FormatInt(glob.Context.ChatID, 10), 0, glob.File)

			choice := glob.getChoice(glob.Context.ChatID)
			switch glob.Context.Meaning {
			case "discard":
				choice[lastStep-1] = 0
			case "like":
				choice[lastStep-1] = 1
			}
			choiceStr, _ := json.Marshal(choice)
			file.UpdateColsCSV(string(choiceStr), 3, strconv.FormatInt(glob.Context.ChatID, 10), 0, glob.File)

		case "choice-made":
			glob.Context.Step = "3"
			file.UpdateColsCSV(glob.Context.Step, 1, strconv.FormatInt(glob.Context.ChatID, 10), 0, glob.File)
		}
	}
}

func (glob *Global) handleResponse() {
	room := glob.getRoom(glob.Context.ChatID)

	switch glob.Context.Meaning {
	case "start":
		// first clean all past records
		glob.init(glob.Context.ChatID)
		glob.Response.Text = "A room is required to find a common choice between multiple friends.\n\nCreate a room or enter an existing room?"
		glob.Response.Options = &[]bot.Button{
			bot.Button{Label: "Create", Value: "create-room"},
			bot.Button{Label: "Enter", Value: "enter-room"},
		}
		glob.Response.IsEdit = true
	case "create-room":
		if room == "" {
			glob.Response.Text = "We could not create a room for you. Try again?"
			glob.Response.Options = &[]bot.Button{
				bot.Button{Label: "Try again!", Value: "create-room"},
				bot.Button{Label: "Enter", Value: "enter-room"},
			}
		} else {
			glob.Response.Text = "Here is your room number: ```" + room + "```.\nNow share it with your friends."
			glob.Response.Options = &[]bot.Button{
				bot.Button{Label: "Done", Value: "room-found"},
			}
		}
		glob.Response.IsEdit = true
	case "enter-room":
		glob.Response.Text = "Okay tell me the room number? You need to ask your friends if you do not already have one."
		glob.Response.IsEdit = true
	case "join-room":
		if room == "" {
			glob.Response.Text = "We could not find a room by that number"
			glob.Response.Options = &[]bot.Button{
				bot.Button{Label: "Create", Value: "create-room"},
				bot.Button{Label: "Enter", Value: "enter-room"},
			}
			glob.Response.IsEdit = true
		} else {
			glob.Response.Text = "Room found!"
			glob.Response.Options = &[]bot.Button{
				bot.Button{Label: "Continue", Value: "room-found"},
			}
		}
	case "room-found":
		glob.Response.Text = "Now I would show you top 10 movies this week in Berlin. You have to like or dislike. You could also stop it anytime. Alright?"
		glob.Response.Options = &[]bot.Button{
			bot.Button{Label: "Meh!", Value: "exit"},
			bot.Button{Label: "Cool!", Value: "start-choice"},
		}
		glob.Response.IsEdit = true
	case "exit":
		glob.Response.Text = "All clear! Have fun manually deciding movies 😂"
		glob.Response.Options = &[]bot.Button{
			bot.Button{Label: "Start Again", Value: "/start"},
		}
	case "start-choice", "discard", "like":
		movieNum, _ := strconv.Atoi(strings.Split(glob.Context.Step, "-")[1])
		glob.Response.Text = fmt.Sprintf(
			"%d. [%s](%s)\n\n%s\n",
			movieNum,
			glob.Movies[movieNum-1].Title,
			glob.Movies[movieNum-1].Link,
			glob.Movies[movieNum-1].Description,
		)
		glob.Response.Options = &[]bot.Button{
			bot.Button{Label: "👎", Value: fmt.Sprintf("discard-%d", movieNum)},
			bot.Button{Label: "👍", Value: fmt.Sprintf("like-%d", movieNum)},
			bot.Button{Label: "Stop", Value: "choice-made"},
		}
		glob.Response.IsEdit = true
		glob.Response.Image = glob.Movies[movieNum-1].Poster
	case "choice-made":
		glob.Response.Text = "Great you are done choosing!"
		glob.Response.Options = &[]bot.Button{
			bot.Button{Label: "Results?", Value: "show-result"},
			bot.Button{Label: "Choose Again", Value: "start-choice"},
		}
	case "show-result":
		mergedChoice := mergeChoices(glob.getChoices(room))
		movieList := glob.getMovieList(mergedChoice)

		if len(movieList) > 0 {
			glob.Response.Text = "So your room has chosen:\n\n"
			for i, movie := range movieList {
				glob.Response.Text = glob.Response.Text + fmt.Sprintf("%d. [%s](%s)\n", i+1, movie.Title, movie.Link)
			}
		} else {
			glob.Response.Text = "Sorry! You do not have any common options.\nRecommended number of choices is six."
		}
		glob.Response.Text = glob.Response.Text + "\n\nYou can try results again when your friends finish."

		glob.Response.Options = &[]bot.Button{
			bot.Button{Label: "Results?", Value: "show-result"},
			bot.Button{Label: "Choose Again", Value: "start-choice"},
			bot.Button{Label: "Exit", Value: "end"},
		}
		glob.Response.IsEdit = true
	case "end":
		glob.init(glob.Context.ChatID)
		glob.Response.Text = "Create a room or enter an existing room?"
		glob.Response.Options = &[]bot.Button{
			bot.Button{Label: "Create", Value: "create-room"},
			bot.Button{Label: "Enter", Value: "enter-room"},
		}
	case "about":
		glob.Response.Text = "*PickFlick*:\n\n" +
			"Open Source on [GitHub](https://github.com/Donnie/PickFlick)\n" +
			"Hosted on Vultr.com in New Jersey, USA\n" +
			"No personally identifiable information is stored or used by this bot."
	default:
		glob.Response.Text = "I didn't get you"
	}

	glob.Bot.Session.Buttons = glob.Response.Options
	glob.Bot.Session.ImageLink = &glob.Response.Image
	glob.Bot.Session.IsEdit = &glob.Response.IsEdit
	glob.Bot.Session.Text = &glob.Response.Text
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
	layout := "db/2006-01-02.json"
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
		log.Panic(e)
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
