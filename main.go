package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Plan struct {
	capacity   float64
	refillRate float64
	unlimited  bool
}

var plans = map[string]Plan{
	"free":       {capacity: 10.0, refillRate: 0.1, unlimited: false},
	"pro":        {capacity: 100.0, refillRate: 1.0, unlimited: false},
	"enterprise": {capacity: 0.0, refillRate: 0.0, unlimited: true},
}

func main() {
	conf := NewConfig()
	sc := bufio.NewScanner(os.Stdin)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)

	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		parts := strings.Split(line, " ")
		cmd := parts[0]
		args := parts[1:]

		if cmd == "NOW" {
			// Time can be fractional (e.g., 1.5)
			t, err := strconv.ParseFloat(args[0], 64)
			if err == nil {
				conf.updateTime(t)
			}
		} else if cmd == "PLAN" {
			if len(args) >= 2 {
				user := args[0]
				planName := args[1]
				conf.setPlan(user, planName)
			}
		} else if cmd == "REQUEST" {
			if len(args) >= 1 {
				user := args[0]
				conf.request(user)
			}
		}
	}
}

type TokenBucket struct {
	tokens         float64
	lastRefillTime float64
	plan           Plan
}

func (tb *TokenBucket) sync(currentTime float64) {
	if tb.plan.unlimited {
		return
	}

	elapsed := currentTime - tb.lastRefillTime
	if elapsed > 0 {
		tb.tokens += elapsed * tb.plan.refillRate

		if tb.tokens > tb.plan.capacity {
			tb.tokens = tb.plan.capacity
		}
	}
	tb.lastRefillTime = currentTime
}

type Config struct {
	time  float64
	users map[string]*TokenBucket
}

func NewConfig() *Config {
	return &Config{
		time:  0.0,
		users: make(map[string]*TokenBucket),
	}
}

func (c *Config) updateTime(t float64) {
	c.time = t
}

func (c *Config) setPlan(user string, planName string) {
	plan, exists := plans[planName]
	if !exists {
		plan = plans["free"]
	}

	tb, ok := c.users[user]
	if !ok {
		c.users[user] = &TokenBucket{
			tokens:         plan.capacity,
			lastRefillTime: c.time,
			plan:           plan,
		}
	} else {

		tb.sync(c.time)
		tb.plan = plan

		if !plan.unlimited && tb.tokens > plan.capacity {
			tb.tokens = plan.capacity
		}
	}
}

func (c *Config) request(user string) {
	tb, ok := c.users[user]
	if !ok {
		c.setPlan(user, "free")
		tb = c.users[user]
	}

	tb.sync(c.time)

	if tb.plan.unlimited {
		fmt.Println("OK unlimited")
		return
	}

	if tb.tokens >= 1.0 {
		tb.tokens -= 1.0

		fmt.Printf("OK %.2f\n", tb.tokens)
	} else {
		fmt.Println("LIMITED")
	}
}
