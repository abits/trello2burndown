package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func handleBurndown(res http.ResponseWriter, req *http.Request) {
	vars, err := ioutil.ReadAll(req.Body)
	if err != nil {
		panic("Cannot parse JSON from request body.")
	}

	trello := NewTrello(vars)
	burndown := NewBurndown(trello, vars)
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
