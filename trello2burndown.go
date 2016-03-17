package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"
)

type Trello struct {
	AppKey           string
	ApiToken         string
	BoardId          string
	Domain           string
	BeginOfSprint    time.Time
	BeginOfSprintRaw string
	ListTitles       map[string]string
	Endpoints        map[string]string
	Matrix           map[string]int
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
func (trello Trello) getLatestDoneAction(card Card) (latestDoneAction Action) {
	query := trello.buildQuery(fmt.Sprintf(trello.Endpoints["getActions"], card.Id))
	params := map[string]string{
		"filter": "updateCard:idList",
	}
	content := executeQuery(query, params)
	actionList := getActionList(content)
	for _, action := range actionList {
		if action.Data.ListAfter.Name == trello.ListTitles["done"] {
			latestDoneAction = action
			break
		}
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
	deltaHours := trello.BeginOfSprint.Sub(time).Hours()
	fmt.Printf("%v\n", deltaHours)
	return int(deltaHours * 24)
}

func evaluate(cardList []Card, matrix map[string]int) (storyPoints int) {
	for _, card := range cardList {
		for _, label := range card.Labels {
			if val, ok := matrix[label.Name]; ok {
				storyPoints = storyPoints + val
			}
		}
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

func main() {
	trello := NewTrello()
	lists := trello.getLists()
	//openCards := trello.getCards(lists[trello.ListTitles["open"]])
	doneCards := trello.getCards(lists[trello.ListTitles["done"]])
	//doingCards := trello.getCards(lists[trello.ListTitles["doing"]])
	//storyPoints := evaluate(openCards, trello.Matrix)
	//fmt.Printf("open: %v\n", storyPoints)
	//storyPoints = evaluate(doneCards, trello.Matrix)
	//fmt.Printf("done: %v\n", storyPoints)
	//storyPoints = evaluate(doingCards, trello.Matrix)
	//fmt.Printf("doing: %v\n", storyPoints)
	latestDoneAction := trello.getLatestDoneAction(doneCards[0])
	fmt.Printf("%v\n", latestDoneAction)
	dayOfWorkOfLastAction := trello.getDayOfWork(latestDoneAction.Time)
	fmt.Printf("%v", dayOfWorkOfLastAction)
	os.Exit(0)
}
