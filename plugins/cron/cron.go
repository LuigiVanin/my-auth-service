package cron

import (
	"fmt"
	"regexp"
	"time"
)

type CronService interface {
	Run(Cron) error
}

type CronOption struct {
	Metadata map[string]any
}

type Cron struct {
	Interval time.Duration
	Name     string
	Service  CronService

	Option CronOption
}

type CronManager struct {
	services []Cron
	Channels []chan bool
	Quit     chan bool
}

func NewCron() CronManager {
	return CronManager{}
}

func (this *CronManager) Register(cron Cron) {
	rgx := regexp.MustCompile("^[a-z-]+$")

	if !rgx.MatchString(cron.Name) {
		fmt.Println("Deu ruim rapa")
		panic("Deu ruim")
	}

	this.services = append(this.services, cron)
}

func Interval(cron Cron, channel chan bool, quit chan bool) {
	runner := func(quit chan bool) {
		time.Sleep(cron.Interval)
		if <-quit {
			return
		}
		cron.Service.Run(cron)
		channel <- true
	}

	go runner(quit)

	for {
		if <-quit {
			break
		}

		if <-channel {
			channel <- false
			go runner(quit)
		}
	}

}

func (this *CronManager) Start(cron Cron) {
	status := make([]bool, len(this.services))
	this.Channels = make([]chan bool, len(this.services))
	quit := make(chan bool)
	quit <- false

	this.Quit = quit

	for index, cron := range this.services {

		go Interval(cron, this.Channels[index], this.Quit)

		status[index] = <-this.Channels[index]
	}
}

func (this *CronManager) Stop() {
	this.Quit <- true
}
