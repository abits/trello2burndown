package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
	"os"
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
	vars, err := ioutil.ReadAll(req.Body)
	if err != nil {
		panic("Cannot parse JSON from request body.")
	}

	config := loadConfigurationFile(CONFIG)
	trello := NewTrello(config, vars)
	burndown := NewBurndown(vars, trello)
	burndown.calculate()
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
	router.Handle("/", http.FileServer(http.Dir("static"))).Methods("GET")
	router.HandleFunc("/burndown", handleBurndown).Methods("POST")
	http.ListenAndServe(":8080", router)
}
