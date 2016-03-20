package main

import (
	"encoding/json"
	"fmt"
	"math"
	"time"
)

type Burndown struct {
	LengthOfSprint      int
	BeginOfSprint       time.Time
	BeginOfSprintString string `json:"BeginOfSprint"`
	totalStoryPoints    float64
	idealRemaining      []float64
	actualRemaining     []int
	idealSpeed          float64
	actualSpeed         float64
	Matrix              map[string]int
}

func NewBurndown(trello *Trello) *Burndown {
	var burndown Burndown
	return &burndown
}

func (burndown *Burndown) calcBurndown(trello *Trello) {
	burndown.totalStoryPoints = float64(burndown.calculateStoryPoints(trello))
	burndown.idealSpeed = burndown.calcIdealSpeed()
	burndown.actualSpeed = burndown.calcActualSpeed(trello)
	burndown.idealRemaining = burndown.calcIdealRemaining()
	burndown.actualRemaining = burndown.calcActualRemaining(trello)
}

func (burndown Burndown) getDayOfWork(time time.Time) (dayOfWork int) {
	deltaHours := math.Ceil((time.Sub(burndown.BeginOfSprint).Hours()))
	dayOfWork = int(deltaHours) / 24
	if math.Mod(deltaHours, 24) > 0 {
		dayOfWork = dayOfWork + 1
	}
	weeks := dayOfWork / 7
	dayOfWork = dayOfWork - weeks*2

	return
}

func (burndown Burndown) getCurrentDayOfWork() int {
	return burndown.getDayOfWork(time.Now())
}

func (burndown *Burndown) configFromFile(file []byte) {
	json.Unmarshal(file, &burndown)
}

func (burndown *Burndown) initBurndown() {
	beginOfSprint := fmt.Sprintf("%sT00:00:00Z", burndown.BeginOfSprintString)
	burndown.BeginOfSprint, _ = time.Parse(time.RFC3339, beginOfSprint)
}

func (burndown Burndown) calcIdealSpeed() float64 {
	return float64(burndown.totalStoryPoints) / float64(burndown.LengthOfSprint)
}

func (burndown Burndown) calcActualSpeed(trello *Trello) float64 {
	donePoints := float64(burndown.evaluateList(trello.DoneCards, burndown.Matrix))
	actualSpeed := float64(donePoints) / float64(burndown.getCurrentDayOfWork())
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
	for idx := 0; idx < burndown.LengthOfSprint; idx++ {
		actualRemaining = append(actualRemaining, int(burndown.totalStoryPoints))
	}
	for _, card := range trello.DoneCards {
		storyPoints := burndown.evaluateCard(card)
		doneAction, _ := trello.getLatestDoneAction(card)
		dayOfWork := burndown.getDayOfWork(doneAction.Time)
		for idx := dayOfWork; idx < len(actualRemaining); idx++ {
			actualRemaining[idx] -= storyPoints
		}
	}
	return
}

func (burndown Burndown) calculateStoryPoints(trello *Trello) (totalStoryPoints int) {
	totalStoryPoints = burndown.evaluateList(trello.DoneCards, burndown.Matrix) +
		burndown.evaluateList(trello.OpenCards, burndown.Matrix) +
		burndown.evaluateList(trello.DoingCards, burndown.Matrix)
	return
}

func (burndown Burndown) evaluateCard(card Card) (storyPoints int) {
	for _, label := range card.Labels {
		if val, ok := burndown.Matrix[label.Name]; ok {
			storyPoints = storyPoints + val
		}
	}
	return storyPoints
}

func (burndown Burndown) evaluateList(cardList []Card, matrix map[string]int) (storyPoints int) {
	for _, card := range cardList {
		storyPoints = storyPoints + burndown.evaluateCard(card)
	}
	return storyPoints
}
