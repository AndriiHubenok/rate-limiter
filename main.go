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
				conf.endpoints[args[0]] = NewTokenEndpoint(args[0], 10.0, 1.0)
			}

			status, used := conf.makeRequest(args[0])

			if status == 200 {
				fmt.Printf("OK %.2f\n", used)
			} else {
				fmt.Println("LIMITED")
			}

		} else if cmd == "STATUS" {
			if _, ok := conf.endpoints[args[0]]; !ok {
				conf.endpoints[args[0]] = NewTokenEndpoint(args[0], 10.0, 1.0)
			}

			conf.endpoints[args[0]].updateTokens(conf.time)
			fmt.Printf("tokens=%.2f\n", conf.endpoints[args[0]].currentTokens)
		}
	}
}

type Config struct {
	time int
	//endpoints map[string]*Endpoint
	endpoints map[string]*TokenEndpoint
}

func NewConfig() *Config {
	return &Config{
		time: 0,
		//endpoints: make(map[string]*Endpoint),
		endpoints: make(map[string]*TokenEndpoint),
	}
}

func (c *Config) updateTime(time int) {
	c.time = time
}

func (c *Config) makeRequest(name string) (int, float64) {
	if value, ok := c.endpoints[name]; ok {
		return value.request(c.time)
	}
	return 429, 0
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

func (e *Endpoint) request(currentTime int) (int, int) {

	if len(e.lastRequestsTime) >= e.maxRequestsPerWindow {

		if e.lastRequestsTime[0] < currentTime-e.windowTime {

			e.lastRequestsTime = e.lastRequestsTime[1:]
			e.lastRequestsTime = append(e.lastRequestsTime, currentTime)

		} else {
			return 429, len(e.lastRequestsTime)
		}
	} else {
		e.lastRequestsTime = append(e.lastRequestsTime, currentTime)
	}

	return 200, len(e.lastRequestsTime)
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

func (t *TokenEndpoint) updateTokens(currentTime int) {

	newTokens := float64(currentTime-t.lastRequestTime)*t.refillRate + t.currentTokens

	if newTokens > t.maxTokens {
		t.currentTokens = t.maxTokens
	} else {
		t.currentTokens = newTokens
	}
}

func (t *TokenEndpoint) request(currentTime int) (int, float64) {

	t.updateTokens(currentTime)

	if t.currentTokens < 1 {
		return 429, 0
	}

	t.currentTokens--
	return 200, t.currentTokens
}
