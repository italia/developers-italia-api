package webhooks

import (
	"log"
	"sync"
	"time"

	"github.com/italia/developers-italia-api/internal/models"
)

// Dispatcher is the function called by the Debouncer when a coalesced
// event is ready to be sent.
type Dispatcher func(models.Event)

// Debouncer collapses bursts of webhook events on the same
// (EntityType, EntityID, Type) key into a single dispatch. A trailing
// timer fires `delay` after the last Submit for that key, capped at
// `cap` from the first Submit so a key under continuous churn still
// gets dispatched.
type Debouncer struct {
	delay    time.Duration
	cap      time.Duration
	dispatch Dispatcher
	now      func() time.Time
	newTimer func(time.Duration, func()) Timer
	mu       sync.Mutex
	pending  map[string]*pendingEvent
}

type pendingEvent struct {
	timer    Timer
	deadline time.Time
	event    models.Event
}

type Timer interface {
	Stop() bool
}

type realTimer struct{ t *time.Timer }

func (r realTimer) Stop() bool { return r.t.Stop() }

func NewDebouncer(delay, capDuration time.Duration, dispatch Dispatcher) *Debouncer {
	return &Debouncer{
		delay:    delay,
		cap:      capDuration,
		dispatch: dispatch,
		now:      time.Now,
		newTimer: func(d time.Duration, f func()) Timer {
			return realTimer{t: time.AfterFunc(d, f)}
		},
		pending: make(map[string]*pendingEvent),
	}
}

func (d *Debouncer) Submit(event models.Event) {
	if d.delay <= 0 {
		d.dispatch(event)

		return
	}

	key := event.EntityType + "/" + event.EntityID + "/" + event.Type

	d.mu.Lock()

	now := d.now()

	prev, ok := d.pending[key]
	if ok {
		prev.timer.Stop()
		prev.event = event
	} else {
		prev = &pendingEvent{
			deadline: now.Add(d.cap),
			event:    event,
		}
		d.pending[key] = prev
	}

	wait := d.delay
	if d.cap > 0 {
		if remaining := prev.deadline.Sub(now); remaining < wait {
			wait = remaining
		}

		if wait < 0 {
			wait = 0
		}
	}

	prev.timer = d.newTimer(wait, func() { d.flush(key) })

	d.mu.Unlock()
}

// Drain stops every pending timer and dispatches the events that were
// waiting, synchronously. Use it on graceful shutdown so a burst that
// arrived in the last `delay` window before SIGTERM is not lost.
//
// Submit must not be called concurrently with Drain or after it returns.
func (d *Debouncer) Drain() {
	d.mu.Lock()

	events := make([]models.Event, 0, len(d.pending))

	for key, pending := range d.pending {
		pending.timer.Stop()
		events = append(events, pending.event)
		delete(d.pending, key)
	}

	d.mu.Unlock()

	for _, event := range events {
		d.safeDispatch(event)
	}
}

func (d *Debouncer) flush(key string) {
	d.mu.Lock()

	pending, ok := d.pending[key]
	if !ok {
		d.mu.Unlock()

		return
	}

	delete(d.pending, key)

	event := pending.event

	d.mu.Unlock()

	d.safeDispatch(event)
}

// safeDispatch isolates the user-supplied Dispatcher so a panic from it
// (a nil deref in the dispatcher closure, a panicking GORM call, etc.)
// does not bring down the timer goroutine and through it the process.
func (d *Debouncer) safeDispatch(event models.Event) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("debouncer: dispatch panicked for %s/%s/%s: %v",
				event.EntityType, event.EntityID, event.Type, r)
		}
	}()

	d.dispatch(event)
}
