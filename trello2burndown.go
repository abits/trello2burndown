package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
)

type Trello struct {
	AppKey     string
	ApiToken   string
	BoardId    string
	Domain     string
	ListTitles map[string]string
	Endpoints  map[string]string
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
		"getLists": "1/boards/%s/lists",
	}
	trello.Domain = "https://api.trello.com/"
	trello.configFromFile("./config.json")

	return &trello
}

type Card struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	ListId string `json:"idList"`
	Url    string `json:"url"`
}

type List struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

func findListNameById(lists []List, idList string) string {
	for _, l := range lists {
		if l.Id == idList {
			return l.Name
		}
	}
	return ""
}

func findListIdByName(lists []List, idName string) string {
	for _, l := range lists {
		if l.Name == idName {
			return l.Id
		}
	}
	return ""
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

func main() {
	trello := NewTrello()
	lists := trello.getLists()
	fmt.Printf("%v", lists[trello.ListTitles["open"]])
	os.Exit(0)
}
