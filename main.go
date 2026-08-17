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
			if len(args) < 2 {
				continue
			}
			ip := args[0]
			user := args[1]
			conf.makeRequest(ip, user)
		}
	}
}

type Config struct {
	time       int
	ipLimits   map[string]*RateLimit
	userLimits map[string]*RateLimit
	global     *RateLimit
}

func NewConfig() *Config {
	return &Config{
		time:       0,
		ipLimits:   make(map[string]*RateLimit),
		userLimits: make(map[string]*RateLimit),
		global:     NewRateLimit(1000),
	}
}

func (c *Config) updateTime(time int) {
	c.time = time
}

func (c *Config) makeRequest(ip, user string) {
	if _, ok := c.ipLimits[ip]; !ok {
		c.ipLimits[ip] = NewRateLimit(5)
	}
	if _, ok := c.userLimits[user]; !ok {
		c.userLimits[user] = NewRateLimit(50)
	}

	// 1. CHECK PHASE (In Order)
	if !c.ipLimits[ip].Allow(c.time) {
		fmt.Println("LIMITED tier=ip")
		return
	}
	if !c.userLimits[user].Allow(c.time) {
		fmt.Println("LIMITED tier=user")
		return
	}
	if !c.global.Allow(c.time) {
		fmt.Println("LIMITED tier=global")
		return
	}

	c.ipLimits[ip].Increment(c.time)
	c.userLimits[user].Increment(c.time)
	c.global.Increment(c.time)

	fmt.Println("OK")
}

type RateLimit struct {
	windowStart int
	count       int
	limit       int
}

func NewRateLimit(limit int) *RateLimit {
	return &RateLimit{
		windowStart: -1,
		count:       0,
		limit:       limit,
	}
}

func (r *RateLimit) sync(currentTime int) {
	currentWindow := currentTime / 60
	if r.windowStart != currentWindow {
		r.windowStart = currentWindow
		r.count = 0
	}
}

func (r *RateLimit) Allow(currentTime int) bool {
	r.sync(currentTime)
	return r.count < r.limit
}

func (r *RateLimit) Increment(currentTime int) {
	r.sync(currentTime)
	r.count++
}

//type Config struct {
//	time      int
//	endpoints map[string]Limiter
//}
//
//func NewConfig() *Config {
//	return &Config{
//		time:      0,
//		endpoints: make(map[string]Limiter),
//	}
//}
//
//func (c *Config) updateTime(time int) {
//	c.time = time
//}
//
//func (c *Config) makeRequest(name string) (int, int, int) {
//	if value, ok := c.endpoints[name]; ok {
//		return value.Request(c.time)
//	}
//	return 429, 0, 0
//}

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

func (e *Endpoint) UpdateTokens(currentTime int) {

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

func (e *Endpoint) GetTokens(currentTime int) int {
	return -1
}

type FixedEndpoint struct {
	id                string
	lastTimeOfRequest int
	maxTokens         int
	attemptsLeft      int
	refillRate        float64
}

func NewFixedEndpoint(id string, maxTokens int, refillRate float64) *FixedEndpoint {
	return &FixedEndpoint{
		id:                id,
		lastTimeOfRequest: 0,
		maxTokens:         maxTokens,
		attemptsLeft:      maxTokens,
		refillRate:        refillRate,
	}
}

func (f *FixedEndpoint) UpdateTokens(currentTime int) {
	if currentTime/60 > f.lastTimeOfRequest/60 {
		f.attemptsLeft = f.maxTokens
	}
	f.lastTimeOfRequest = currentTime
}

func (f *FixedEndpoint) Request(currentTime int) (int, int, int) {
	f.UpdateTokens(currentTime)

	resetTime := (currentTime/60)*60 + 60

	if f.attemptsLeft <= 0 {
		return 429, 0, resetTime
	}

	f.attemptsLeft--
	return 200, f.maxTokens - f.attemptsLeft, resetTime
}

func (f *FixedEndpoint) GetTokens(currentTime int) int {
	return -1
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
