package main

import (
	"fmt"
	"io/ioutil"
	"os"
)

func loadConfigurationFile(filename string) (file []byte) {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Printf("Cannot read from config file: %v\n", err)
		os.Exit(1)
	}
	return
}

func main() {
	trello := NewTrello()
	file := loadConfigurationFile("./config.json")
	trello.configFromFile(file)
	trello.initBoard()
	burndown := NewBurndown(trello)
	burndown.configFromFile(file)
	burndown.initBurndown()
	burndown.calcBurndown(trello)
	fmt.Printf("%v\n", burndown)
	os.Exit(0)
}
