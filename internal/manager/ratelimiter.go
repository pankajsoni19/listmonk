package manager

import (
	"time"

	"go.uber.org/ratelimit"
)

type customClock struct {
	p *pipe
}

func (p *pipe) attachRateLimiter() {
	if !p.camp.SlidingWindow {
		return
	}

	dur, _ := time.ParseDuration(p.camp.SlidingWindowDuration)
	clock := &customClock{
		p: p,
	}

	p.ratelimiter = ratelimit.New(p.camp.SlidingWindowRate,
		ratelimit.WithSlack(0),
		ratelimit.Per(dur),
		ratelimit.WithClock(clock),
	)
}

func (c *customClock) Now() time.Time { return time.Now() }

// pause strict on hitting rate limit
func (c *customClock) Sleep(wait time.Duration) {
	startTime := time.Now()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:

			if c.p.stopped.Load() {
				return
			}

			if time.Since(startTime) > wait {
				return
			}
		}
	}
}

// long running campaign, wait for event
func (p *pipe) waitForEvent() bool {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	p.m.log.Printf("campaign: %s, will wait for events, at 1 min interval", p.camp.Name)

	for {
		select {
		case <-ticker.C:
			p.m.log.Printf("campaign %s, stopped: %t, has: %t", p.camp.Name, p.stopped.Load(), p.flagSubQueued.Load())
			if p.stopped.Load() {
				return false
			}

			if p.flagSubQueued.Load() {
				p.flagSubQueued.Store(false)
				p.m.log.Printf("campaign: %s, got event", p.camp.Name)
				return true
			}
		}
	}
}
