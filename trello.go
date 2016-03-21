package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"
)

type Trello struct {
	AppKey     string
	ApiToken   string
	BoardId    string
	Domain     string
	ListTitles map[string]string
	Endpoints  map[string]string
	DoneCards  []Card
	DoingCards []Card
	OpenCards  []Card
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

func NewTrello(file []byte) *Trello {
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
	trello.configFromFile(file)
	trello.initBoard()
	return &trello
}

func (trello *Trello) configFromFile(file []byte) {
	json.Unmarshal(file, &trello)
}

func (trello *Trello) initBoard() {
	lists := trello.getLists()
	trello.DoneCards = trello.getCards(lists[trello.ListTitles["done"]])
	trello.OpenCards = trello.getCards(lists[trello.ListTitles["open"]])
	trello.DoingCards = trello.getCards(lists[trello.ListTitles["doing"]])
}

// buildQuery returns url object for trello api domain.
func (trello Trello) buildQuery(endpoint string) (trelloApi *url.URL) {
	trelloApi, _ = url.Parse(trello.Domain)
	trelloApi.Path = endpoint
	var q = trelloApi.Query()
	q.Add("key", trello.AppKey)
	q.Add("token", trello.ApiToken)
	trelloApi.RawQuery = q.Encode()

	return
}

// executeQuery endpoint and return response content.
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

// getLists on trello board, needed for list ids as params in other queries.
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

	return
}

// getCards on a certain list
func (trello Trello) getCards(listId string) (cardList []Card) {
	query := trello.buildQuery(fmt.Sprintf(trello.Endpoints["getCards"], listId))
	params := map[string]string{
		"fields": "labels,id,name,idList",
	}
	content := executeQuery(query, params)
	cardList = make([]Card, 0)
	json.Unmarshal(content, &cardList)

	return
}

// getLatestDoneAction for a certain card
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

// getLabel information for a certain label id
func (trello Trello) getLabel(labelId string) {
	query := trello.buildQuery(fmt.Sprintf(trello.Endpoints["getLabel"], labelId))
	params := map[string]string{
		"fields": "name",
	}
	content := executeQuery(query, params)
	cardList := make([]Card, 0)
	json.Unmarshal(content, &cardList)
}

func getActionList(content []byte) (actionList []Action) {
	actionList = make([]Action, 0)
	json.Unmarshal(content, &actionList)
	for idx, action := range actionList {
		actionTime, _ := time.Parse(
			time.RFC3339Nano,
			action.DateString)
		actionList[idx].Time = actionTime
	}
	return
}
