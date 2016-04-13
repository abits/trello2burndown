package main

import (
	"encoding/json"
	"fmt"
	"math"
	"time"
)

type Burndown struct {
	LengthOfSprint      int `json:"length"`
	BeginOfSprint       time.Time
	BeginOfSprintString string `json:"begin"`
	TotalStoryPoints    int
	IdealRemaining      []float64
	ActualRemaining     []int
	IdealSpeed          float64
	ActualSpeed         float64
	Metric              map[string]int `json:"metric"`
	trello              *Trello
}

type ValuedDoneAction struct {
	storyPoints int
	doneAction  *Action
}

func NewBurndown(trello *Trello, vars []byte) *Burndown {
	var burndown Burndown
	burndown.trello = trello
	burndown.configFrom(vars)
	return &burndown
}

func (burndown *Burndown) setBeginOfSprint(beginOfSprintString string) {
	burndown.BeginOfSprintString = beginOfSprintString
	beginOfSprint := fmt.Sprintf("%sT00:00:00Z", beginOfSprintString)
	burndown.BeginOfSprint, _ = time.Parse(time.RFC3339, beginOfSprint)
}

func (burndown *Burndown) configFrom(data []byte) {
	json.Unmarshal(data, &burndown)
	burndown.setBeginOfSprint(burndown.BeginOfSprintString)
}

func (burndown *Burndown) calculate() {
	burndown.TotalStoryPoints = burndown.calculateTotalStoryPoints()
	burndown.IdealSpeed = burndown.calculateIdealSpeed()
	burndown.IdealRemaining = burndown.calculateIdealRemaining()
	burndown.ActualRemaining = burndown.calculateActualRemainingAsync()
	burndown.ActualSpeed = burndown.calculateActualSpeed()
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
	currentDayOfWork := burndown.getDayOfWork(time.Now())
	if currentDayOfWork > burndown.LengthOfSprint {
		currentDayOfWork = burndown.LengthOfSprint
	}
	return currentDayOfWork
}

func (burndown Burndown) calculateIdealSpeed() float64 {
	return float64(burndown.TotalStoryPoints) / float64(burndown.LengthOfSprint)
}

func (burndown Burndown) calculateActualSpeed() (actualSpeed float64) {
	currentDayOfWork := burndown.getCurrentDayOfWork()
	donePoints := float64(burndown.evaluateList(burndown.trello.board.DoneCards))
	actualSpeed = float64(donePoints) / float64(currentDayOfWork)
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
	for idx := 0; idx < burndown.getCurrentDayOfWork(); idx++ {
		actualRemaining = append(actualRemaining, int(burndown.TotalStoryPoints))
	}
	for _, card := range burndown.trello.board.DoneCards {
		storyPoints := burndown.evaluateCard(card)
		doneAction, _ := burndown.trello.getLatestDoneAction(card)
		dayOfWork := burndown.getDayOfWork(doneAction.Time)
		for idx := dayOfWork; idx < len(actualRemaining); idx++ {
			actualRemaining[idx] -= storyPoints
		}
	}
	return
}

func (burndown Burndown) calculateActualRemainingAsync() (actualRemaining []int) {
	currentDayOfWork := burndown.getCurrentDayOfWork()
	for idx := 0; idx < currentDayOfWork; idx++ {
		actualRemaining = append(actualRemaining, int(burndown.TotalStoryPoints))
	}
	ch := make(chan *ValuedDoneAction, len(burndown.trello.board.DoneCards))
	for _, card := range burndown.trello.board.DoneCards {
		storyPoints := burndown.evaluateCard(card)
		go func(card Card) {
			doneAction, _ := burndown.trello.getLatestDoneAction(card)
			ch <- &ValuedDoneAction{storyPoints, &doneAction}
		}(card)
	}
	counter := 0
	for {
		select {
		case r := <-ch:
			dayOfWork := burndown.getDayOfWork(r.doneAction.Time)
			for idx := dayOfWork; idx < len(actualRemaining); idx++ {
				actualRemaining[idx] -= r.storyPoints
			}
			counter++
			if counter >= len(burndown.trello.board.DoneCards) {
				return actualRemaining[:currentDayOfWork]
			}
		}
	}

	return actualRemaining[:currentDayOfWork]
}

func (burndown Burndown) calculateTotalStoryPoints() (totalStoryPoints int) {
	doneStoryPoints := burndown.evaluateList(burndown.trello.board.DoneCards)
	openStoryPoints := burndown.evaluateList(burndown.trello.board.OpenCards)
	doingStoryPoints := burndown.evaluateList(burndown.trello.board.DoingCards)
	totalStoryPoints = doneStoryPoints + openStoryPoints + doingStoryPoints
	return
}

func (burndown Burndown) evaluateCard(card Card) (storyPoints int) {
	for _, label := range card.Labels {
		if val, ok := burndown.Metric[label.Name]; ok {
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
