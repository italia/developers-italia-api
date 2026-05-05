package webhooks

import (
	"sync"
	"testing"
	"time"

	"github.com/italia/developers-italia-api/internal/models"
	"github.com/stretchr/testify/assert"
)

type fakeTimer struct {
	fire    func()
	wait    time.Duration
	stopped bool
}

func (f *fakeTimer) Stop() bool {
	f.stopped = true

	return true
}

type fakeClock struct {
	mu     sync.Mutex
	now    time.Time
	timers []*fakeTimer
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)
	c.mu.Unlock()
}

func (c *fakeClock) NewTimer(d time.Duration, f func()) Timer {
	t := &fakeTimer{fire: f, wait: d}

	c.mu.Lock()
	c.timers = append(c.timers, t)
	c.mu.Unlock()

	return t
}

func (c *fakeClock) LastWait() time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.timers[len(c.timers)-1].wait
}

func (c *fakeClock) FireLatest() {
	c.mu.Lock()
	t := c.timers[len(c.timers)-1]
	c.mu.Unlock()

	if !t.stopped {
		t.fire()
	}
}

func (c *fakeClock) FireAll() {
	c.mu.Lock()
	timers := append([]*fakeTimer(nil), c.timers...)
	c.mu.Unlock()

	for _, t := range timers {
		if !t.stopped {
			t.fire()
		}
	}
}

func newDebouncerWithClock(delay, capDuration time.Duration, dispatch Dispatcher) (*Debouncer, *fakeClock) {
	clock := &fakeClock{now: time.Unix(0, 0)}
	d := &Debouncer{
		delay:    delay,
		cap:      capDuration,
		dispatch: dispatch,
		now:      clock.Now,
		newTimer: clock.NewTimer,
		pending:  make(map[string]*pendingEvent),
	}

	return d, clock
}

func TestDebouncerCoalescesBurstByKey(t *testing.T) {
	var (
		mu       sync.Mutex
		received []models.Event
	)

	dispatch := func(e models.Event) {
		mu.Lock()
		received = append(received, e)
		mu.Unlock()
	}

	d, clock := newDebouncerWithClock(time.Second, 10*time.Second, dispatch)

	for i := range 5 {
		d.Submit(models.Event{
			EntityType: "Software",
			EntityID:   "abc",
			Type:       "Updated",
			ID:         string(rune('a' + i)),
		})
		clock.Advance(100 * time.Millisecond)
	}

	clock.FireLatest()

	mu.Lock()
	defer mu.Unlock()

	assert.Len(t, received, 1, "burst on same key collapses to one dispatch")
	assert.Equal(t, "e", received[0].ID, "the latest event in the burst is the one dispatched")
}

func TestDebouncerCapCeilingFiresBeforeIndefiniteDefer(t *testing.T) {
	var (
		mu       sync.Mutex
		received []models.Event
	)

	dispatch := func(e models.Event) {
		mu.Lock()
		received = append(received, e)
		mu.Unlock()
	}

	d, clock := newDebouncerWithClock(100*time.Millisecond, 300*time.Millisecond, dispatch)

	for range 6 {
		d.Submit(models.Event{EntityType: "Software", EntityID: "x", Type: "Updated"})
		clock.Advance(50 * time.Millisecond)
	}

	// After 6 submits at 50ms each, 250ms have elapsed since the first
	// Submit, so the cap leaves only 50ms before the deadline. Without
	// the cap the timer would have been re-armed for the full 100ms
	// delay every Submit, never converging.
	assert.Equal(t, 50*time.Millisecond, clock.LastWait(),
		"last timer must be capped, not the full delay")

	clock.FireLatest()

	mu.Lock()
	defer mu.Unlock()

	assert.Len(t, received, 1, "cap should force dispatch after 300ms despite 100ms delay resets")
}

func TestDebouncerSeparateKeysFireIndependently(t *testing.T) {
	var (
		mu       sync.Mutex
		received []models.Event
	)

	dispatch := func(e models.Event) {
		mu.Lock()
		received = append(received, e)
		mu.Unlock()
	}

	d, clock := newDebouncerWithClock(time.Second, 10*time.Second, dispatch)

	d.Submit(models.Event{EntityType: "Software", EntityID: "a", Type: "Updated"})
	d.Submit(models.Event{EntityType: "Software", EntityID: "b", Type: "Updated"})

	clock.FireAll()

	mu.Lock()
	defer mu.Unlock()

	assert.Len(t, received, 2)
}

func TestDebouncerZeroDelayDispatchesImmediately(t *testing.T) {
	var (
		mu       sync.Mutex
		received []models.Event
	)

	dispatch := func(e models.Event) {
		mu.Lock()
		received = append(received, e)
		mu.Unlock()
	}

	d, _ := newDebouncerWithClock(0, 0, dispatch)

	d.Submit(models.Event{EntityType: "Software", EntityID: "a", Type: "Updated"})

	mu.Lock()
	defer mu.Unlock()

	assert.Len(t, received, 1, "zero delay disables debouncing")
}

func TestDebouncerDrainFlushesPendingEvents(t *testing.T) {
	var (
		mu       sync.Mutex
		received []models.Event
	)

	dispatch := func(e models.Event) {
		mu.Lock()
		received = append(received, e)
		mu.Unlock()
	}

	d, _ := newDebouncerWithClock(time.Second, 10*time.Second, dispatch)

	d.Submit(models.Event{EntityType: "Software", EntityID: "a", Type: "Updated"})
	d.Submit(models.Event{EntityType: "Publisher", EntityID: "b", Type: "Created"})

	d.Drain()

	mu.Lock()
	defer mu.Unlock()

	assert.Len(t, received, 2, "Drain dispatches every pending key")
	assert.Empty(t, d.pending, "Drain leaves the pending map empty")
}

func TestDebouncerDispatchPanicDoesNotCrashTimer(t *testing.T) {
	var dispatched int

	dispatch := func(_ models.Event) {
		dispatched++

		panic("boom")
	}

	d, clock := newDebouncerWithClock(time.Second, 10*time.Second, dispatch)

	d.Submit(models.Event{EntityType: "Software", EntityID: "a", Type: "Updated"})

	assert.NotPanics(t, func() { clock.FireLatest() },
		"a panicking dispatcher must not propagate out of the timer goroutine")

	assert.Equal(t, 1, dispatched)
}
