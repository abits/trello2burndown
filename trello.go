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

const CONFIG = "./config.json"

func loadConfigurationFile(filename string) (file []byte) {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Printf("Cannot read from config file: %v\n", err)
		os.Exit(1)
	}
	return
}

type Trello struct {
	AppKey    string
	ApiToken  string
	Domain    string
	Endpoints map[string]string
	board     *Board
}

type Board struct {
	BoardId    string `json:"boardId"`
	BoardName  string
	ListTitles map[string]string
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

func NewTrello(vars []byte) *Trello {
	var trello Trello
	trello.Domain = "https://api.trello.com/"
	trello.Endpoints = map[string]string{
		"getLists":   "1/boards/%s/lists",
		"getCards":   "1/lists/%s/cards",
		"getLabel":   "1/labels/%s",
		"getActions": "1/cards/%s/actions",
		"getBoard":   "1/boards/%s",
	}
	file := loadConfigurationFile(CONFIG)
	trello.configureFrom(file)
	trello.board = NewBoard(vars)
	trello.initBoard(trello.board)
	return &trello
}

func NewBoard(vars []byte) *Board {
	var board Board
	board.ListTitles = map[string]string{
		"open":    "Offen",
		"doing":   "In Arbeit",
		"done":    "Erledigt",
		"backlog": "Backlog",
	}
	board.configureFrom(vars)
	return &board
}

func (trello *Trello) configureFrom(data []byte) {
	json.Unmarshal(data, &trello)
}

func (board *Board) configureFrom(data []byte) {
	json.Unmarshal(data, &board)
}

func (trello *Trello) initBoard(board *Board) {
	lists := trello.getLists(board.BoardId)
	trello.initCardsAsync(lists, board)
}

func (trello *Trello) initCardsAsync(lists map[string]string, board *Board) {
	done := make(chan []Card)
	open := make(chan []Card)
	doing := make(chan []Card)
	go func(lists map[string]string, board *Board) {
		cards := trello.getCards(lists[board.ListTitles["done"]])
		done <- cards
	}(lists, board)
	go func(lists map[string]string, board *Board) {
		cards := trello.getCards(lists[board.ListTitles["open"]])
		open <- cards
	}(lists, board)
	go func(lists map[string]string, board *Board) {
		cards := trello.getCards(lists[board.ListTitles["doing"]])
		doing <- cards
	}(lists, board)
	board.OpenCards = <-open
	board.DoneCards = <-done
	board.DoingCards = <-doing
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
func (trello Trello) getLists(boardId string) (listMap map[string]string) {
	query := trello.buildQuery(fmt.Sprintf(trello.Endpoints["getLists"], boardId))
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
		if action.Data.ListAfter.Name == trello.board.ListTitles["done"] {
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
