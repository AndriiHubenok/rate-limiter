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
				conf.endpoints[args[0]] = NewEndpoint(args[0], 60, 5)
			}

			status, used := conf.makeRequest(args[0])

			if status == 200 {
				fmt.Printf("OK %d\n", used)
			} else {
				fmt.Println("LIMITED")
			}
		}
	}
}

type Config struct {
	time      int
	endpoints map[string]*Endpoint
}

func NewConfig() *Config {
	return &Config{
		time:      0,
		endpoints: make(map[string]*Endpoint),
	}
}

func (c *Config) updateTime(time int) {
	c.time = time
}

func (c *Config) makeRequest(name string) (int, int) {
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
