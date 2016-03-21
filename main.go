package main

import (
	"fmt"
	"io/ioutil"
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

func main() {
	config := loadConfigurationFile(CONFIG)
	trello := NewTrello(config)
	burndown := NewBurndown(config, trello)
	burndown.calculate()
	fmt.Printf("%v\n", burndown)
	os.Exit(0)
}
