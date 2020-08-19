package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"sort"
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

	glob.retrieve()
	glob.handleContext()
	glob.handleAction()
	glob.handleResponse()
	glob.persist()
	glob.Bot.Send()

	c.JSON(200, nil)
}

func (glob *Global) handleMessage(msg Message) {
	replyToID := int(*msg.MessageID)

	glob.Bot.Session.ChatID = msg.Chat.ID
	glob.Bot.Session.ReplyToID = &replyToID
	glob.Context.ChatID = *msg.Chat.ID
	glob.Context.Text = *msg.Text
}

func (glob *Global) handleCallback(call CallbackQuery) {
	sentMsgID := int(*call.Message.MessageID)

	glob.Bot.Session.CallBackID = call.ID
	glob.Bot.Session.ChatID = call.Message.Chat.ID
	glob.Bot.Session.SentMsgID = &sentMsgID
	glob.Context.ChatID = *call.Message.Chat.ID
	glob.Context.Text = *call.Data

	toasts := []string{"Okay!", "Cool!", "Alright!", "Fine!", "Hmmm!", "Nice"}
	glob.Bot.Session.Toast = &toasts[randInt(0, 4)]
}

func (glob *Global) handleContext() {
	if glob.Context.Text == "/start" {
		glob.Context.Meaning = "start"
		return
	}
	if glob.Context.Text == "/about" {
		glob.Context.Meaning = "about"
		return
	}
	if glob.Context.Text == "/help" {
		glob.Context.Meaning = "help"
		return
	}
	if glob.Context.Text == "create-room" ||
		glob.Context.Text == "choice-room" ||
		glob.Context.Text == "choice-more" ||
		glob.Context.Text == "enter-room" ||
		glob.Context.Text == "exit" ||
		glob.Context.Text == "new-list" ||
		glob.Context.Text == "prev-list" ||
		glob.Context.Text == "room-found" ||
		glob.Context.Text == "start-choice" {
		glob.Context.Meaning = glob.Context.Text
		return
	}
	if len(glob.Context.Text) == 3 && glob.Context.Step == "1" {
		glob.Context.Meaning = "join-room"
		return
	}
	if strings.Contains(glob.Context.Text, "discard") {
		glob.Context.Meaning = "discard"
		return
	}
	if strings.Contains(glob.Context.Text, "like") {
		glob.Context.Meaning = "like"
		return
	}
	if glob.Context.Text == "choice-made" {
		glob.Context.Meaning = "choice-made"
		return
	}
	if glob.Context.Text == "show-result" && glob.Context.Step == "3" {
		glob.Context.Meaning = "show-result"
		return
	}
	if glob.Context.Text == "end" && glob.Context.Step == "3" {
		glob.Context.Meaning = "end"
		return
	}
}

func (glob *Global) handleAction() {
	switch glob.Context.Meaning {
	case "create-room":
		glob.Context.Step = "1"
		glob.Context.RoomID = genRoomNum()
		glob.Context.Limit = 11

	case "enter-room":
		// register step 1
		glob.Context.Step = "1"
		glob.Context.Limit = 11

	case "join-room":
		if glob.isRoom(glob.Context.Text) {
			glob.Context.RoomID = glob.Context.Text
		}

	case "start-choice":
		glob.Context.Step = "2-" + strconv.Itoa(glob.Context.Limit-10)

	case "new-list":
		glob.Context.Step = "2-" + strconv.Itoa(glob.Context.Limit)
		glob.Context.Limit = glob.Context.Limit + 10

	case "prev-list":
		glob.Context.Step = "2-" + strconv.Itoa(glob.Context.Limit-20)
		glob.Context.Limit = glob.Context.Limit - 10

	case "discard", "like":
		lastStep, _ := strconv.Atoi(strings.Split(glob.Context.Text, "-")[1])
		glob.Context.Step = "2-" + strconv.Itoa(lastStep+1)

		switch glob.Context.Meaning {
		case "like":
			glob.addChoice(lastStep)
		case "discard":
			glob.removeChoice(lastStep)
		}

		if glob.Context.Step == "2-"+strconv.Itoa(glob.Context.Limit) {
			glob.Context.Step = "3"
			glob.Context.Meaning = "choice-made"
		}

	case "choice-made":
		glob.Context.Step = "3"
	case "end":
		glob.Context.Step = "1"
		glob.Context.RoomID = ""
		glob.Context.Limit = 11
		glob.Context.Choice = nil
	}
}

func (glob *Global) handleResponse() {
	switch glob.Context.Meaning {
	case "start":
		glob.Response.Text = "Hello there, I am PickFlick! I can help you and your " +
			"friends decide on a movie evening by taking you through the latest shows currently in town.\n\n" +
			"Do not worry I would also send you the ticket links to buy"
		glob.Response.Options = &[]bot.Button{
			bot.Button{Label: "Meh!", Value: "exit"},
			bot.Button{Label: "Continue!", Value: "choice-room"},
		}
	case "choice-room":
		glob.Response.Text = "A room is required to find a common choice between multiple friends.\n\nCreate a room or enter an existing room?"
		glob.Response.Options = &[]bot.Button{
			bot.Button{Label: "Create", Value: "create-room"},
			bot.Button{Label: "Enter", Value: "enter-room"},
		}
		glob.Response.IsEdit = true
	case "create-room":
		if glob.Context.RoomID == "" {
			glob.Response.Text = "We could not create a room for you. Try again?"
			glob.Response.Options = &[]bot.Button{
				bot.Button{Label: "Try again!", Value: "create-room"},
				bot.Button{Label: "Enter", Value: "enter-room"},
			}
		} else {
			glob.Response.Text = "Here is your room number: ```" + glob.Context.RoomID + "```.\nNow share it with your friends."
			glob.Response.Options = &[]bot.Button{
				bot.Button{Label: "Done", Value: "room-found"},
			}
		}
		glob.Response.IsEdit = true
	case "enter-room":
		glob.Response.Text = "Okay tell me the room number? You need to ask your friends if you do not already have one."
		glob.Response.IsEdit = true
	case "join-room":
		if glob.Context.RoomID == "" {
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
		glob.Response.Text = "Now I would show you 10 movies currently running in Berlin. You have to like or dislike. You could also skip to the end anytime. Alright?"
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
	case "new-list", "prev-list", "start-choice", "discard", "like":
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
			bot.Button{Label: "Skip", Value: "choice-made"},
		}
		glob.Response.IsEdit = true
		glob.Response.Image = &glob.Movies[movieNum-1].Poster
	case "choice-made":
		glob.Response.Text = "Great you are done choosing!"
		glob.Response.Options = &[]bot.Button{
			bot.Button{Label: "Results?", Value: "show-result"},
			bot.Button{Label: "Choose Again", Value: "start-choice"},
		}
	case "show-result":
		movieList := glob.movieChoices()
		if len(movieList) > 0 {
			glob.Response.Text = "So you have together chosen:\n\n"
			for i, mc := range movieList {
				glob.Response.Text = glob.Response.Text + fmt.Sprintf("%d. [%s](%s) **%d**%%\n", i+1, mc.Movie.Title, mc.Movie.Link, mc.Percent)
			}
		} else {
			glob.Response.Text = "Sorry! You do not have any common choices.\nRecommended number of choices is six."
		}
		glob.Response.Text = glob.Response.Text + "\n\nYou can try results again if your friends still need to finish.\n" +
			"Or click More to get more options"

		glob.Response.Options = &[]bot.Button{
			bot.Button{Label: "Results?", Value: "show-result"},
			bot.Button{Label: "More", Value: "choice-more"},
			bot.Button{Label: "Exit", Value: "end"},
		}
		glob.Response.IsEdit = true
	case "choice-more":
		glob.Response.Text = "Do you want to see a new list of movies or try again with previous ones?"
		options := []bot.Button{
			bot.Button{Label: "New list", Value: "new-list"},
			bot.Button{Label: "Same list", Value: "start-choice"},
		}
		if glob.Context.Limit > 11 {
			options = append(options, bot.Button{Label: "Previous list", Value: "prev-list"})
		}
		glob.Response.Options = &options
	case "end":
		glob.Response.Text = "Click here to /start again"
	case "help":
		glob.Response.Text = "*Help*:\n\n" +
			"This bot shows you top ten movies playing in Berlin currently.\n" +
			"Try to choose at least six movies so that the probability of a match is higher with your friends.\n" +
			"Once complete it would show you the results. But you would be able to try again.\n" +
			"The results are saved as long as you do not exit the bot.\n\n" +
			"/start Use this command to restart the bot anytime during usage."
	case "about":
		glob.Response.Text = "*PickFlick*:\n\n" +
			"Open Source on [GitHub](https://github.com/Donnie/PickFlick)\n" +
			"Hosted on Vultr.com in New Jersey, USA\n" +
			"No personally identifiable information is stored or used by this bot."
	default:
		glob.Response.Text = "I didn't get you"
	}

	glob.Bot.Session.Buttons = glob.Response.Options
	glob.Bot.Session.ImageLink = glob.Response.Image
	glob.Bot.Session.IsEdit = &glob.Response.IsEdit
	glob.Bot.Session.Text = &glob.Response.Text
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

func (glob *Global) addChoice(current int) {
	glob.Context.Choice = append(glob.Context.Choice, current)
	glob.Context.Choice = deDupe(glob.Context.Choice)
}

func (glob *Global) removeChoice(current int) {
	glob.Context.Choice = remove(glob.Context.Choice, current)
}

func (glob *Global) getChoices() (choices [][]int) {
	mem, err := file.ReadCSV(glob.File)
	if err != nil {
		return
	}
	for _, line := range mem {
		choice := []int{}
		if glob.Context.RoomID == line[2] {
			json.Unmarshal([]byte(line[3]), &choice)
			choices = append(choices, choice)
		}
	}
	return
}

func (glob *Global) movieChoices() (mc []MovieChoice) {
	choices := glob.getChoices()

	choiceMap := make([]int, len(glob.Movies))

	for _, choice := range choices {
		for _, unit := range choice {
			choiceMap[unit]++
		}
	}

	for i, unit := range choiceMap {
		percent := int(float64(unit) / float64(len(choices)) * float64(100))
		if percent > 50 {
			mc = append(mc, MovieChoice{
				Movie:   glob.Movies[i-1],
				Percent: percent,
			})
		}
	}

	sort.Slice(mc, func(i, j int) bool {
		return mc[i].Percent > mc[j].Percent
	})
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

func (glob *Global) retrieve() {
	mem, err := file.ReadCSV(glob.File)
	if err != nil {
		return
	}
	for _, line := range mem {
		lineChatID, _ := strconv.ParseInt(line[0], 10, 64)
		if glob.Context.ChatID == lineChatID {
			glob.Context.Step = line[1]
			glob.Context.RoomID = line[2]
			json.Unmarshal([]byte(line[3]), &glob.Context.Choice)
			glob.Context.Limit, _ = strconv.Atoi(line[4])
			break
		}
	}
}

func (glob *Global) persist() {
	chatID := strconv.FormatInt(glob.Context.ChatID, 10)
	choiceStr, _ := json.Marshal(glob.Context.Choice)

	done, _ := file.UpdateLinesCSV([]string{
		chatID,
		glob.Context.Step,
		glob.Context.RoomID,
		string(choiceStr),
		strconv.Itoa(glob.Context.Limit),
	}, glob.File, chatID, 0)

	if !done {
		file.WriteLineCSV([]string{
			chatID,
			"1",
			"",
			"nil",
			"11",
		}, glob.File)
	}
}

func (glob *Global) poll() {
	for range time.Tick(time.Hour) {
		go glob.handleScrape()
	}
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

func deDupe(slice []int) (list []int) {
	keys := make(map[int]bool)
	for _, entry := range slice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return
}

func remove(s []int, i int) (o []int) {
	for _, e := range s {
		if e != i {
			o = append(o, e)
		}
	}
	return
}
