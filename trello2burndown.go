package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"os"
	"time"
)

type Trello struct {
	AppKey              string
	ApiToken            string
	BoardId             string
	Domain              string
	BeginOfSprint       time.Time
	LengthOfSprint      int
	BeginOfSprintString string `json:"BeginOfSprint"`
	ListTitles          map[string]string
	Endpoints           map[string]string
	Matrix              map[string]int
	DoneCards           []Card
	DoingCards          []Card
	OpenCards           []Card
}

func NewTrello() *Trello {
	var trello Trello
	trello.ListTitles = map[string]string{
		"open":    "Offen",
		"doing":   "In Arbeit",
		"done":    "Erledigt",
		"backlog": "Backlog",
	}
	trello.Endpoints = map[string]string{
		"getLists":   "1/boards/%s/lists",
		"getCards":   "1/lists/%s/cards",
		"getLabel":   "1/labels/%s",
		"getActions": "1/cards/%s/actions",
	}
	trello.Domain = "https://api.trello.com/"
	trello.configFromFile("./config.json")
	beginOfSprint := fmt.Sprintf("%sT00:00:00Z", trello.BeginOfSprintString)
	trello.BeginOfSprint, _ = time.Parse(time.RFC3339, beginOfSprint)
	lists := trello.getLists()
	trello.DoneCards = trello.getCards(lists[trello.ListTitles["done"]])
	trello.OpenCards = trello.getCards(lists[trello.ListTitles["open"]])
	trello.DoingCards = trello.getCards(lists[trello.ListTitles["doing"]])
	return &trello
}

type Card struct {
	Id     string  `json:"id"`
	Name   string  `json:"name"`
	ListId string  `json:"idList"`
	Url    string  `json:"url"`
	Labels []Label `json:"labels"`
}

type List struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type Label struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type ActionData struct {
	ListBefore List `json:"listBefore"`
	ListAfter  List `json:"listAfter"`
}

type Action struct {
	Id         string `json:"id"`
	Type       string `json:"type"`
	DateString string `json:"date"`
	Time       time.Time
	Data       ActionData `json:"data"`
}

func (trello *Trello) configFromFile(filename string) {
	file, e := ioutil.ReadFile(filename)
	if e != nil {
		fmt.Printf("Cannot read from config file: %v\n", e)
		os.Exit(1)
	}
	json.Unmarshal(file, &trello)
}

// Return url object for trello api domain.
func (trello Trello) buildQuery(endpoint string) *url.URL {
	var trelloApi, _ = url.Parse(trello.Domain)
	trelloApi.Path = endpoint
	var q = trelloApi.Query()
	q.Add("key", trello.AppKey)
	q.Add("token", trello.ApiToken)
	trelloApi.RawQuery = q.Encode()

	return trelloApi
}

// Query endpoint and return response content.
func executeQuery(url *url.URL, params map[string]string) (response []byte) {
	query := url.Query()
	for key, value := range params {
		query.Add(key, value)
	}
	url.RawQuery = query.Encode()
	resp, err := http.Get(url.String())
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	response, err = ioutil.ReadAll(resp.Body)

	return
}

// Get Lists on trello board, needed for list ids as params in other queries.
func (trello Trello) getLists() (listMap map[string]string) {
	query := trello.buildQuery(fmt.Sprintf(trello.Endpoints["getLists"], trello.BoardId))
	params := map[string]string{
		"fields": "name,idList,url,labels",
	}
	content := executeQuery(query, params)

	lists := make([]List, 0)
	json.Unmarshal(content, &lists)
	listMap = make(map[string]string, 0)
	for _, list := range lists {
		listMap[list.Name] = list.Id
	}
	return listMap
}

// Get Cards on a certain list
func (trello Trello) getCards(listId string) (cardList []Card) {
	query := trello.buildQuery(fmt.Sprintf(trello.Endpoints["getCards"], listId))
	params := map[string]string{
		"fields": "labels,id,name,idList",
	}
	content := executeQuery(query, params)
	cardList = make([]Card, 0)
	json.Unmarshal(content, &cardList)
	return cardList
}

// Get actions for a certain card
func (trello Trello) getLatestDoneAction(card Card) (latestDoneAction Action, err error) {
	query := trello.buildQuery(fmt.Sprintf(trello.Endpoints["getActions"], card.Id))
	params := map[string]string{
		"filter": "updateCard:idList",
	}
	content := executeQuery(query, params)
	actionList := getActionList(content)
	isDone := false
	for _, action := range actionList {
		if action.Data.ListAfter.Name == trello.ListTitles["done"] {
			latestDoneAction = action
			isDone = true
			break
		}
	}
	if isDone == false {
		err = errors.New("Action is not yet done.")
	}
	return

}

// Get label information for a certain label id
func (trello Trello) getLabel(labelId string) {
	query := trello.buildQuery(fmt.Sprintf(trello.Endpoints["getLabel"], labelId))
	params := map[string]string{
		"fields": "name",
	}
	content := executeQuery(query, params)
	cardList := make([]Card, 0)
	json.Unmarshal(content, &cardList)
}

func (trello Trello) getDayOfWork(time time.Time) (dayOfWork int) {
	deltaHours := math.Ceil((time.Sub(trello.BeginOfSprint).Hours()))
	dayOfWork = int(deltaHours) / 24
	if math.Mod(deltaHours, 24) > 0 {
		dayOfWork = dayOfWork + 1
	}
	weeks := dayOfWork / 7
	dayOfWork = dayOfWork - weeks*2
	return
}

func (card Card) evaluateCard(matrix map[string]int) (storyPoints int) {
	for _, label := range card.Labels {
		if val, ok := matrix[label.Name]; ok {
			storyPoints = storyPoints + val
		}
	}
	return storyPoints
}

func evaluateList(cardList []Card, matrix map[string]int) (storyPoints int) {
	for _, card := range cardList {
		storyPoints = storyPoints + card.evaluateCard(matrix)
	}
	return storyPoints
}

func getActionList(content []byte) []Action {
	actionList := make([]Action, 0)
	json.Unmarshal(content, &actionList)
	for idx, action := range actionList {
		actionTime, _ := time.Parse(
			time.RFC3339Nano,
			action.DateString)
		actionList[idx].Time = actionTime
	}
	return actionList
}

func (trello Trello) calculateStoryPoints() (totalStoryPoints int) {
	totalStoryPoints = evaluateList(trello.DoneCards, trello.Matrix) +
		evaluateList(trello.OpenCards, trello.Matrix) +
		evaluateList(trello.DoingCards, trello.Matrix)
	return
}

func (burndown Burndown) calcIdealSpeed() float64 {
	return float64(burndown.totalStoryPoints) / float64(burndown.LengthOfSprint)
}

func (burndown Burndown) calcActualSpeed(trello *Trello) float64 {
	donePoints := float64(evaluateList(trello.DoneCards, trello.Matrix))
	actualSpeed := float64(donePoints) / float64(trello.getCurrentDayOfWork())
	return actualSpeed
}

func (burndown Burndown) calcIdealRemaining() (idealRemaining []float64) {
	lengthOfSprint := int(burndown.LengthOfSprint)
	for day := 1; day <= lengthOfSprint; day++ {
		idealRemaining = append(idealRemaining, (burndown.totalStoryPoints - float64(day)*burndown.idealSpeed))
	}
	return idealRemaining
}

func (burndown Burndown) calcActualRemaining(trello *Trello) (actualRemaining []int) {
	for idx := 0; idx < trello.LengthOfSprint; idx++ {
		actualRemaining = append(actualRemaining, int(burndown.totalStoryPoints))
	}
	for _, card := range trello.DoneCards {
		storyPoints := card.evaluateCard(trello.Matrix)
		doneAction, _ := trello.getLatestDoneAction(card)
		dayOfWork := trello.getDayOfWork(doneAction.Time)
		for idx := dayOfWork; idx < len(actualRemaining); idx++ {
			actualRemaining[idx] -= storyPoints
		}
	}
	return
}

func (trello Trello) getCurrentDayOfWork() int {
	return trello.getDayOfWork(time.Now())
}

type Burndown struct {
	LengthOfSprint   float64
	totalStoryPoints float64
	idealRemaining   []float64
	actualRemaining  []int
	idealSpeed       float64
	actualSpeed      float64
}

func NewBurndown(trello *Trello) *Burndown {
	var burndown Burndown
	burndown.LengthOfSprint = float64(trello.LengthOfSprint)
	burndown.totalStoryPoints = float64(trello.calculateStoryPoints())
	burndown.idealSpeed = burndown.calcIdealSpeed()
	burndown.actualSpeed = burndown.calcActualSpeed(trello)
	burndown.idealRemaining = burndown.calcIdealRemaining()
	burndown.actualRemaining = burndown.calcActualRemaining(trello)
	return &burndown
}

func main() {
	trello := NewTrello()
	burndown := NewBurndown(trello)
	fmt.Printf("%v\n", burndown)
	os.Exit(0)
}
