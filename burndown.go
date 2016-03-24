package main

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"
)

type Burndown struct {
	LengthOfSprint      int
	BeginOfSprint       time.Time
	BeginOfSprintString string `json:"BeginOfSprint"`
	TotalStoryPoints    int
	IdealRemaining      []float64
	ActualRemaining     []int
	IdealSpeed          float64
	ActualSpeed         float64
	Matrix              map[string]int
	trello              *Trello
}

func NewBurndown(file []byte, trello *Trello) *Burndown {
	var burndown Burndown
	burndown.trello = trello
	burndown.configFromFile(file)
	return &burndown
}

func (burndown *Burndown) setBeginOfSprint(beginOfSprintString string) {
	burndown.BeginOfSprintString = beginOfSprintString
	beginOfSprint := fmt.Sprintf("%sT00:00:00Z", beginOfSprintString)
	burndown.BeginOfSprint, _ = time.Parse(time.RFC3339, beginOfSprint)
}

func (burndown *Burndown) configFromFile(file []byte) {
	json.Unmarshal(file, &burndown)
	burndown.setBeginOfSprint(burndown.BeginOfSprintString)
}

func (burndown *Burndown) setParametersFromRequest(vars map[string]string) {
	if boardId, ok := vars["boardId"]; ok {
		burndown.trello.BoardId = boardId
		fmt.Printf("burndown.trello.BoardId: %v\n", burndown.trello.BoardId)
	}
	if beginOfSprint, ok := vars["begin"]; ok {
		burndown.setBeginOfSprint(beginOfSprint)
	}
	if lengthOfSprint, ok := vars["length"]; ok {
		lengthOfSprintInt, _ := strconv.Atoi(lengthOfSprint)
		burndown.LengthOfSprint = lengthOfSprintInt
	}
}

func (burndown *Burndown) calculate(vars map[string]string) {
	burndown.setParametersFromRequest(vars)
	burndown.TotalStoryPoints = burndown.calculateTotalStoryPoints()
	burndown.IdealSpeed = burndown.calculateIdealSpeed()
	burndown.ActualSpeed = burndown.calculateActualSpeed()
	burndown.IdealRemaining = burndown.calculateIdealRemaining()
	burndown.ActualRemaining = burndown.calculateActualRemaining()
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

func (burndown Burndown) calculateIdealSpeed() float64 {
	return float64(burndown.TotalStoryPoints) / float64(burndown.LengthOfSprint)
}

func (burndown Burndown) calculateActualSpeed() (actualSpeed float64) {
	donePoints := float64(burndown.evaluateList(burndown.trello.DoneCards))
	actualSpeed = float64(donePoints) / float64(burndown.getCurrentDayOfWork())
	return
}

func (burndown Burndown) calculateIdealRemaining() (idealRemaining []float64) {
	lengthOfSprint := int(burndown.LengthOfSprint)
	for day := 1; day <= lengthOfSprint; day++ {
		idealRemaining = append(idealRemaining, (float64(burndown.TotalStoryPoints) - float64(day)*burndown.IdealSpeed))
	}
	return
}

func (burndown Burndown) calculateActualRemaining() (actualRemaining []int) {
	for idx := 0; idx < burndown.LengthOfSprint; idx++ {
		actualRemaining = append(actualRemaining, int(burndown.TotalStoryPoints))
	}
	for _, card := range burndown.trello.DoneCards {
		storyPoints := burndown.evaluateCard(card)
		doneAction, _ := burndown.trello.getLatestDoneAction(card)
		dayOfWork := burndown.getDayOfWork(doneAction.Time)
		for idx := dayOfWork; idx < len(actualRemaining); idx++ {
			actualRemaining[idx] -= storyPoints
		}
	}
	return
}

func (burndown Burndown) calculateTotalStoryPoints() (totalStoryPoints int) {
	doneStoryPoints := burndown.evaluateList(burndown.trello.DoneCards)
	openStoryPoints := burndown.evaluateList(burndown.trello.OpenCards)
	doingStoryPoints := burndown.evaluateList(burndown.trello.DoingCards)
	fmt.Printf("doneStoryPoints: %v\n", doneStoryPoints)
	fmt.Printf("openStoryPoints: %v\n", openStoryPoints)
	fmt.Printf("doingStoryPoints: %v\n", doingStoryPoints)
	totalStoryPoints = doneStoryPoints + openStoryPoints + doingStoryPoints
	return
}

func (burndown Burndown) evaluateCard(card Card) (storyPoints int) {
	for _, label := range card.Labels {
		if val, ok := burndown.Matrix[label.Name]; ok {
			storyPoints = storyPoints + val
		}
	}
	return
}

func (burndown Burndown) evaluateList(cardList []Card) (storyPoints int) {
	for _, card := range cardList {
		storyPoints = storyPoints + burndown.evaluateCard(card)
	}
	return
}
