package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func main() {
	conf := NewConfig()
	sc := bufio.NewScanner(os.Stdin)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		parts := strings.Split(line, " ")
		cmd := parts[0]
		args := parts[1:]

		if cmd == "NOW" {
			time, err := strconv.Atoi(args[0])
			if err != nil {
				fmt.Println("Invalid time")
				continue
			}
			conf.updateTime(time)
		} else if cmd == "REQUEST" {
			if _, ok := conf.requests[args[0]]; !ok {
				conf.requests[args[0]] = NewEndpoint(args[0], conf.time)
			}

			status, remaining, resetTime := conf.makeRequest(args[0])

			fmt.Println("X-RateLimit-Limit: 10")
			fmt.Printf("X-RateLimit-Remaining: %d\n", remaining)
			fmt.Printf("X-RateLimit-Reset: %d\n", resetTime)
			fmt.Printf("status: %d\n", status)
		}
	}
}

type Config struct {
	time     int
	requests map[string]*Endpoint
}

func NewConfig() *Config {
	return &Config{
		time:     0,
		requests: make(map[string]*Endpoint),
	}
}

func (c *Config) updateTime(time int) {
	c.time = time
}

func (c *Config) makeRequest(name string) (int, int, int) {
	if value, ok := c.requests[name]; ok {
		return value.request(c.time)
	}
	return 429, 0, 0
}

type Endpoint struct {
	id                string
	lastTimeOfRequest int
	attemptsLeft      int
}

func NewEndpoint(id string, lastTimeOfRequest int) *Endpoint {
	return &Endpoint{
		id:                id,
		lastTimeOfRequest: lastTimeOfRequest,
		attemptsLeft:      10,
	}
}

func (e *Endpoint) updateAttemptsLeft(currentTime int) {
	if currentTime/60 > e.lastTimeOfRequest/60 {
		e.attemptsLeft = 10
	}
	e.lastTimeOfRequest = currentTime
}

func (e *Endpoint) request(currentTime int) (int, int, int) {
	e.updateAttemptsLeft(currentTime)

	resetTime := (currentTime/60)*60 + 60

	if e.attemptsLeft <= 0 {
		return 429, 0, resetTime
	}

	e.attemptsLeft--
	return 200, e.attemptsLeft, resetTime
}
