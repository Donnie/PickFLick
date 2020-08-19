package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Donnie/PickFlick/bot"
	"github.com/Donnie/PickFlick/file"
	"github.com/Donnie/PickFlick/scraper"
	"github.com/gin-gonic/gin"
)

var roomIDlen = 3
var defLim = 10
var chcInc = 10
var defObj = "movies"
var defColl = "Berlin"

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
	if glob.Context.Text == "/start" || glob.Context.Fresh {
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
	if len(glob.Context.Text) == roomIDlen && glob.Context.Step == "room" {
		glob.Context.Meaning = "join-room"
		return
	}
	if strings.Contains(glob.Context.Text, "dislike") {
		glob.Context.Meaning = "dislike"
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
	if glob.Context.Text == "show-result" && glob.Context.Step == "result" {
		glob.Context.Meaning = "show-result"
		return
	}
	if glob.Context.Text == "end" && glob.Context.Step == "result" {
		glob.Context.Meaning = "end"
		return
	}
}

func (glob *Global) handleAction() {
	switch glob.Context.Meaning {
	case "create-room":
		glob.Context.Step = "room"
		glob.Context.RoomID = genRoomNum()
		glob.Context.Limit = defLim

	case "enter-room":
		// register step 1
		glob.Context.Step = "room"
		glob.Context.Limit = defLim

	case "join-room":
		if glob.isRoom(glob.Context.Text) {
			glob.Context.RoomID = glob.Context.Text
		}

	case "start-choice":
		glob.Context.Step = "choice-" + strconv.Itoa(glob.Context.Limit-chcInc+1)

	case "new-list":
		glob.Context.Step = "choice-" + strconv.Itoa(glob.Context.Limit+1)
		glob.Context.Limit = glob.Context.Limit + chcInc

	case "prev-list":
		glob.Context.Step = "choice-" + strconv.Itoa(glob.Context.Limit-(chcInc*2)+1)
		glob.Context.Limit = glob.Context.Limit - chcInc

	case "dislike", "like":
		lastStep, _ := strconv.Atoi(strings.Split(glob.Context.Text, "-")[1])

		switch glob.Context.Meaning {
		case "like":
			glob.addChoice(lastStep)
		case "dislike":
			glob.removeChoice(lastStep)
		}

		if glob.Context.Step == "choice-"+strconv.Itoa(glob.Context.Limit) {
			glob.Context.Step = "result"
			glob.Context.Meaning = "choice-made"
		} else {
			glob.Context.Step = "choice-" + strconv.Itoa(lastStep+1)
		}

	case "choice-made":
		glob.Context.Step = "result"

	case "end":
		glob.Context.Step = "room"
		glob.Context.RoomID = ""
		glob.Context.Limit = defLim
		glob.Context.Choice = nil
	}
}

func (glob *Global) handleResponse() {
	switch glob.Context.Meaning {
	case "start":
		glob.Response.Text = "Hello there, I am PickFlick! I can help you and your friends " +
			"decide on a special evening by taking you through the latest " + defObj + " currently in " + defColl + ".\n\n" +
			"Do not worry I would also send you the links at the end. 😎"
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
		glob.Response.Text = fmt.Sprintf("Now I would show you %d %s currently in %s.", defLim, defObj, defColl) +
			" You have to like or dislike. You could also skip to the end anytime. Alright?"
		glob.Response.Options = &[]bot.Button{
			bot.Button{Label: "Meh!", Value: "exit"},
			bot.Button{Label: "Cool!", Value: "start-choice"},
		}
		glob.Response.IsEdit = true
	case "exit":
		glob.Response.Text = fmt.Sprintf("All clear! Have fun manually deciding %s 😂", defObj)
		glob.Response.Options = &[]bot.Button{
			bot.Button{Label: "Start Again", Value: "/start"},
		}
	case "new-list", "prev-list", "start-choice", "dislike", "like":
		movieNum, _ := strconv.Atoi(strings.Split(glob.Context.Step, "-")[1])
		glob.Response.Text = fmt.Sprintf(
			"%d. [%s](%s)\n\n%s\n",
			movieNum,
			glob.Movies[movieNum-1].Title,
			glob.Movies[movieNum-1].Link,
			glob.Movies[movieNum-1].Description,
		)
		glob.Response.Options = &[]bot.Button{
			bot.Button{Label: "👎", Value: fmt.Sprintf("dislike-%d", movieNum)},
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
		glob.Response.Text = fmt.Sprintf("Do you want to see a new list of %s or try again with previous ones?", defObj)
		options := []bot.Button{
			bot.Button{Label: "New list", Value: "new-list"},
			bot.Button{Label: "Same list", Value: "start-choice"},
		}
		if glob.Context.Limit > defLim {
			options = append(options, bot.Button{Label: "Previous list", Value: "prev-list"})
		}
		glob.Response.Options = &options
	case "end":
		glob.Response.Text = "Click here to /start again"
	case "help":
		glob.Response.Text = "*Help*:\n\n" +
			fmt.Sprintf("This bot shows you top %d %s from %s.\n", defLim, defObj, defColl) +
			fmt.Sprintf("Try to choose at least %d %s so that the probability of a match is higher with your friends.\n", ((defLim/2)+1), defObj) +
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
		if unit == 0 {
			continue
		}
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
	found := false
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
			found = true
			break
		}
	}
	if !found {
		glob.Context.Fresh = true
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
			"room",
			"",
			"nil",
			strconv.Itoa(defLim),
		}, glob.File)
	}
}

func (glob *Global) poll() {
	for range time.Tick(time.Hour) {
		go glob.handleScrape()
	}
}
