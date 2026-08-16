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
			if _, ok := conf.endpoints[args[0]]; !ok {
				conf.endpoints[args[0]] = NewLeakyEndpoint(args[0], 5, 1.0)
			}

			status, used, _ := conf.makeRequest(args[0])

			if status == 200 {
				fmt.Printf("OK %.2f\n", float64(used))
			} else {
				fmt.Println("LIMITED")
			}

		} else if cmd == "STATUS" {
			if _, ok := conf.endpoints[args[0]]; !ok {
				conf.endpoints[args[0]] = NewLeakyEndpoint(args[0], 5, 1.0)
			}

			fmt.Printf("water=%.2f\n", float64(conf.endpoints[args[0]].GetTokens(conf.time)))
		}
	}
}

type Config struct {
	time int
	//endpoints map[string]*Endpoint
	endpoints map[string]Limiter
}

func NewConfig() *Config {
	return &Config{
		time: 0,
		//endpoints: make(map[string]*Endpoint),
		endpoints: make(map[string]Limiter),
	}
}

func (c *Config) updateTime(time int) {
	c.time = time
}

func (c *Config) makeRequest(name string) (int, int, int) {
	if value, ok := c.endpoints[name]; ok {
		return value.Request(c.time)
	}
	return 429, 0, 0
}

type Limiter interface {
	Request(currentTime int) (status int, remaining int, resetTime int)
	UpdateTokens(currentTime int)
	GetTokens(currentTime int) int
}

type Endpoint struct {
	id                   string
	windowTime           int
	maxRequestsPerWindow int
	lastRequestsTime     []int
}

func NewEndpoint(id string, windowTime int, maxRequestsPerWindow int) *Endpoint {
	return &Endpoint{
		id:                   id,
		windowTime:           windowTime,
		maxRequestsPerWindow: maxRequestsPerWindow,
		lastRequestsTime:     make([]int, 0, maxRequestsPerWindow),
	}
}

func (e *Endpoint) updateAttemptsLeft(currentTime int) {

}

func (e *Endpoint) Request(currentTime int) (int, int, int) {

	if len(e.lastRequestsTime) >= e.maxRequestsPerWindow {

		if e.lastRequestsTime[0] < currentTime-e.windowTime {

			e.lastRequestsTime = e.lastRequestsTime[1:]
			e.lastRequestsTime = append(e.lastRequestsTime, currentTime)

		} else {
			return 429, len(e.lastRequestsTime), 0
		}
	} else {
		e.lastRequestsTime = append(e.lastRequestsTime, currentTime)
	}

	return 200, len(e.lastRequestsTime), 0
}

type TokenEndpoint struct {
	id              string
	maxTokens       float64
	currentTokens   float64
	refillRate      float64
	lastRequestTime int
}

func NewTokenEndpoint(id string, maxTokens int, refillRate float64) *TokenEndpoint {

	return &TokenEndpoint{
		id:              id,
		maxTokens:       float64(maxTokens),
		currentTokens:   float64(maxTokens),
		refillRate:      refillRate,
		lastRequestTime: 0,
	}
}

func (t *TokenEndpoint) UpdateTokens(currentTime int) {

	newTokens := float64(currentTime-t.lastRequestTime)*t.refillRate + t.currentTokens

	if newTokens > t.maxTokens {
		t.currentTokens = t.maxTokens
	} else {
		t.currentTokens = newTokens
	}
	t.lastRequestTime = currentTime
}

func (t *TokenEndpoint) Request(currentTime int) (int, int, int) {

	t.UpdateTokens(currentTime)

	if t.currentTokens < 1 {
		return 429, 0, 0
	}

	t.currentTokens--
	return 200, int(t.currentTokens), 0
}

type LeakyEndpoint struct {
	id              string
	maxTokens       float64
	currentTokens   float64
	refillRate      float64
	lastRequestTime int
}

func NewLeakyEndpoint(id string, maxTokens int, refillRate float64) *LeakyEndpoint {

	return &LeakyEndpoint{
		id:              id,
		maxTokens:       float64(maxTokens),
		currentTokens:   0,
		refillRate:      refillRate,
		lastRequestTime: 0,
	}
}

func (l *LeakyEndpoint) UpdateTokens(currentTime int) {

	newTokens := l.currentTokens - float64(currentTime-l.lastRequestTime)*l.refillRate

	if newTokens <= 0 {
		l.currentTokens = 0
	} else {
		l.currentTokens = newTokens
	}
	l.lastRequestTime = currentTime
}

func (l *LeakyEndpoint) Request(currentTime int) (int, int, int) {

	l.UpdateTokens(currentTime)

	if l.currentTokens >= 5 {
		return 429, 0, 0
	}

	l.currentTokens++
	return 200, int(l.currentTokens), 0
}

func (l *LeakyEndpoint) GetTokens(currentTime int) int {
	l.UpdateTokens(currentTime)
	return int(l.currentTokens)
}
