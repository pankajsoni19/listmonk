package manager

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/knadh/listmonk/models"
	"github.com/paulbellamy/ratecounter"
)

type pipe struct {
	camp                  *models.Campaign
	rate                  *ratecounter.RateCounter
	wg                    *sync.WaitGroup
	sent                  atomic.Int64
	lastID                atomic.Uint64
	errors                atomic.Uint64
	stopped               atomic.Bool
	flagSubQueued         atomic.Bool
	withErrors            atomic.Bool
	m                     *Manager
	slidingStart          time.Time
	SlidingWindowDuration time.Duration
	slidingCount          int
}

// newPipe adds a campaign to the process queue.
func (m *Manager) newPipe(c *models.Campaign) (*pipe, error) {
	// Validate messenger.
	if _, ok := m.messengers[c.Messenger]; !ok {
		m.store.UpdateCampaignStatus(c.ID, models.CampaignStatusCancelled)
		return nil, fmt.Errorf("unknown messenger %s on campaign %s", c.Messenger, c.Name)
	}

	// Load the template.
	if err := c.CompileTemplate(m.TemplateFuncs(c)); err != nil {
		return nil, err
	}

	// Load any media/attachments.
	if err := m.attachMedia(c); err != nil {
		return nil, err
	}

	dur, _ := time.ParseDuration(c.SlidingWindowDuration)

	// Add the campaign to the active map.
	p := &pipe{
		camp:                  c,
		rate:                  ratecounter.NewRateCounter(time.Minute),
		wg:                    &sync.WaitGroup{},
		m:                     m,
		slidingStart:          time.Now(),
		SlidingWindowDuration: dur,
		slidingCount:          0,
	}

	// Increment the waitgroup so that Wait() blocks immediately. This is necessary
	// as a campaign pipe is created first and subscribers/messages under it are
	// fetched asynchronolusly later. The messages each add to the wg and that
	// count is used to determine the exhaustion/completion of all messages.
	p.wg.Add(1)

	go func() {
		// Wait for all the messages in the campaign to be processed
		// (successfully or skipped after errors or cancellation).
		p.wg.Wait()

		p.Stop(false)
		p.cleanup()
	}()

	m.pipesMut.Lock()
	m.pipes[c.ID] = p
	m.pipesMut.Unlock()
	return p, nil
}

// NextSubscribers processes the next batch of subscribers in a given campaign.
// It returns a bool indicating whether any subscribers were processed
// in the current batch or not. A false indicates that all subscribers
// have been processed, or that a campaign has been paused or cancelled.
type NextSubResult struct {
	Has   bool
	Error error
}

func (p *pipe) NextSubscribers(result chan *NextSubResult) {
	// Fetch a batch of subscribers.
	subs, err := p.m.store.NextSubscribers(p.camp.ID, p.m.cfg.BatchSize)
	if err != nil {
		result <- &NextSubResult{
			Error: fmt.Errorf("error fetching campaign subscribers (%s): %v", p.camp.Name, err),
		}
		return
	}

	// There are no subscribers.
	if len(subs) == 0 {
		result <- &NextSubResult{}
		return
	}

	// Is there a sliding window limit configured?
	hasSliding := p.camp.SlidingWindow

	// Push messages.
	for _, s := range subs {

		if p.stopped.Load() {
			break
		}

		msg, err := p.newMessage(s)
		if err != nil {
			p.m.log.Printf("error rendering message (%s) (%s): %v", p.camp.Name, s.Email, err)
			continue
		}

		// Push the message to the queue while blocking and waiting until
		// the queue is drained.
		p.m.campMsgQ <- msg

		// Check if the sliding window is active.
		if hasSliding {
			diff := time.Since(p.slidingStart)

			// Window has expired. Reset the clock.
			if diff >= p.SlidingWindowDuration {
				p.slidingStart = time.Now()
				p.slidingCount = 0
				continue
			}

			// Have the messages exceeded the limit?
			p.slidingCount++
			if p.slidingCount >= p.camp.SlidingWindowRate {
				wait := (p.SlidingWindowDuration - diff).Round(time.Second)

				p.m.log.Printf("messages exceeded (%d) for the window (%v since %s). Sleeping for %s.",
					p.slidingCount,
					p.SlidingWindowDuration,
					p.slidingStart.Format(time.RFC822Z),
					wait)

				p.slidingCount = 0
				p.pauseFor(wait)
			}
		}
	}

	result <- &NextSubResult{Has: true}
}

func (p *pipe) OnError() {
	if p.m.cfg.MaxSendErrors < 1 {
		return
	}

	// If the error threshold is met, pause the campaign.
	count := p.errors.Add(1)
	if int(count) < p.m.cfg.MaxSendErrors {
		return
	}

	p.Stop(true)
	p.m.log.Printf("error count exceeded %d. pausing campaign %s", p.m.cfg.MaxSendErrors, p.camp.Name)
}

// Stop "marks" a campaign as stopped. It doesn't actually stop the processing
// of messages. That happens when every queued message in the campaign is processed,
// marking .wg, the waitgroup counter as done. That triggers cleanup().
func (p *pipe) Stop(withErrors bool) {
	// Already stopped.
	if p.stopped.Load() {
		return
	}

	if withErrors {
		p.withErrors.Store(true)
	}

	p.stopped.Store(true)
}

func (p *pipe) newMessage(s models.Subscriber) (CampaignMessage, error) {
	msg, err := p.m.NewCampaignMessage(p.camp, s)
	if err != nil {
		return msg, err
	}

	msg.pipe = p
	p.wg.Add(1)

	return msg, nil
}

func (p *pipe) pauseFor(wait time.Duration) {
	startTime := time.Now()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if p.stopped.Load() {
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

func (p *pipe) cleanup() {
	defer func() {
		p.m.pipesMut.Lock()
		delete(p.m.pipes, p.camp.ID)
		p.m.pipesMut.Unlock()
	}()

	// Update campaign's "sent" count.
	if err := p.m.store.UpdateCampaignCounts(p.camp.ID, 0, int(p.sent.Load()), int(p.lastID.Load())); err != nil {
		p.m.log.Printf("error updating campaign counts (%s): %v", p.camp.Name, err)
	}

	// The campaign was auto-paused due to errors.
	if p.withErrors.Load() {
		if err := p.m.store.UpdateCampaignStatus(p.camp.ID, models.CampaignStatusPaused); err != nil {
			p.m.log.Printf("error updating campaign (%s) status to %s: %v", p.camp.Name, models.CampaignStatusPaused, err)
		} else {
			p.m.log.Printf("set campaign (%s) to %s", p.camp.Name, models.CampaignStatusPaused)
		}

		_ = p.m.sendNotif(p.camp, models.CampaignStatusPaused, "Too many errors")
		return
	}

	// Fetch the up-to-date campaign status from the DB.
	c, err := p.m.store.GetCampaign(p.camp.ID)
	if err != nil {
		p.m.log.Printf("error fetching campaign (%s) for ending: %v", p.camp.Name, err)
		return
	}

	// If a running campaign has exhausted subscribers, it's finished.
	if c.Status == models.CampaignStatusRunning {
		c.Status = models.CampaignStatusFinished

		if err := p.m.store.UpdateCampaignStatus(p.camp.ID, models.CampaignStatusFinished); err != nil {
			p.m.log.Printf("error finishing campaign (%s): %v", p.camp.Name, err)
		} else {
			p.m.log.Printf("campaign (%s) finished", p.camp.Name)
		}
	} else {
		p.m.log.Printf("stop processing campaign (%s)", p.camp.Name)
	}

	// Notify the admin.
	_ = p.m.sendNotif(c, c.Status, "")
}
