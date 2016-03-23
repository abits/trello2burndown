package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
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

func handleBurndown(res http.ResponseWriter, req *http.Request) {
	config := loadConfigurationFile(CONFIG)
	trello := NewTrello(config)
	burndown := NewBurndown(config, trello)
	vars := mux.Vars(req)
	beginOfSprint := req.FormValue("beginOfSprint")
	if beginOfSprint != "" {
		vars["beginOfSprint"] = beginOfSprint
	}
	burndown.calculate(vars)
	res.Header().Set("Content-Type", "application/json")
	res.Header().Set("Access-Control-Allow-Origin", "*")

	data, error := json.Marshal(burndown)
	if error != nil {
		log.Println(error.Error())
		http.Error(res, error.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprint(res, string(data))
}

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/burndown/{boardId}", handleBurndown).Methods("GET")
	http.ListenAndServe(":8080", router)
}
