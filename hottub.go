package hottub

import (
	"errors"
	"sync"
	"time"
)

const (
	// DefaultMaxResources is the default number of resources in a pool
	DefaultMaxResources = uint(10)
	// DefaultTakeTimeout is the default time for a resource timeout
	DefaultTakeTimeout = time.Millisecond * 50
	// DefaultAliveCheck is how often to check available Resources' statuses
	DefaultAliveCheck = time.Millisecond * 1000
	// TimeoutBlocking means asking for a Resource won't time out
	TimeoutBlocking = -1
)

var (
	// ErrCannotClosePool happens when there's a problem closing the resource pool
	ErrCannotClosePool = errors.New("Error closing resource people")
	// ErrTimeoutTakingResource occurs when getting a resource times out
	ErrTimeoutTakingResource = errors.New("Timed out taking resource")
	// ErrTimeoutClosingPool occurs when closing a resource pool times out
	ErrTimeoutClosingPool = errors.New("Closing resource pool timed out")
	// ErrGeneratorRequired is when you try and create a pool with no Generator
	ErrGeneratorRequired = errors.New("Cannot create a Pool with no Generator")
	// ErrUnmanagedResource when you try and return a Resource this pool doesn't manage
	ErrUnmanagedResource = errors.New("This resource isn't managed by this Pool")
	// ErrAlreadyClosed when Close() is called on a closed Pool
	ErrAlreadyClosed = errors.New("Pool is already closed")
	// ErrPoolIsClosed when a pending
)

// Pool defines a resource pool
type Pool interface {
	// Take will return a channel to receive a Resource and if the Resource is still alive
	Take() (Resource, error)
	// Return gives a taken Resource back to the Pool
	Return(Resource) error
	// Close will shutdown the entire Resource Pool
	Close() error
	// Manages determines if the Resource is appropriate for this Pool
	Manages(r Resource) bool
	// Max tells you the maximum number of available Resources
	Max() uint
	// Available tells you the number of available Resources
	Available() int
}

// Resource is anything pooled
type Resource interface {
	// Alive tells you if the Resource is still alive, or available
	Alive() (bool, error)
	// Close will perform any tearing down of a Resource
	Close() error
}

// Generator is a function that creates a Resource
type Generator func() Resource

// PoolParams are initialization parameters for a Resource Pool
type PoolParams struct {
	// MaxResources is the maximum number of ressources this pool will manage and provide
	MaxResources uint
	// Timeout determines how long to wait for a Resource, Timeout < 0 = blocking, Timeout 0 = DefaultTakeTimeout
	Timeout time.Duration
	// Check determines how often to check if Resources are alive, < 0 = DefaultAliveCheck
	Check time.Duration
}

// pool is the internal structural representation of a Pool
type pool struct {
	max        uint
	gen        Generator
	regenerate bool
	timeout    time.Duration
	check      time.Duration
	closed     bool

	resources resources
	out       resources

	requests stack
}

// NewPool takes the provided PoolParams and generates a Pool, or an error
func NewPool(g Generator, params *PoolParams) (Pool, error) {
	if g == nil {
		return nil, ErrGeneratorRequired
	}
	max := params.MaxResources
	if max <= 0 {
		max = DefaultMaxResources
	}
	timeout := params.Timeout
	if timeout == 0 {
		timeout = DefaultTakeTimeout
	}
	check := params.Check
	if check <= 0 {
		check = DefaultAliveCheck
	}

	p := &pool{
		gen:     g,
		max:     max,
		timeout: timeout,
		check:   check,

		resources: make(resources, max),
		out:       make(resources, 0),

		requests: newStack(),
	}

	for i := uint(0); i < p.max; i++ {
		p.resources[i] = p.gen()
	}

	return p, nil
}

func (p *pool) Take() (Resource, error) {
	rec := newReceiver()
	if len(p.resources) == 0 {
		p.requests.push(rec)
		rec.start(p.timeout)
	} else {
		go func() {
			r := p.resources.pop()
			ok, err := r.Alive()
			if err != nil {
				rec.response <- err
			}
			if !ok {
				r = p.gen()
			}
			rec.resource <- r
			rec.replied = true
		}()
	}

	for {
		select {
		case r := <-rec.resource:
			p.out.push(r)
			return r, nil
		case err := <-rec.response:
			return nil, err
		}
	}
}

func (p *pool) Return(r Resource) error {
	idx, manages := p.manages(r)
	if !manages {
		return ErrUnmanagedResource
	}
	p.out.remove(idx)
	p.resources.push(r)

	for !p.requests.isEmpty() {
		rec := p.requests.pop()
		if rec.replied || rec.timedOut {
			continue
		}

		rec.resource <- r
		break
	}

	return nil
}

func (p *pool) Manages(r Resource) bool {
	_, managed := p.manages(r)
	return managed
}

func (p *pool) manages(r Resource) (int, bool) {
	return p.out.indexOf(r)
}

// TODO: this isn't working, leverage a channel
func (p *pool) Close() error {
	if p.closed {
		return ErrAlreadyClosed
	}
	p.closed = true

	wg := sync.WaitGroup{}
	wg.Add(len(p.resources))
	wg.Add(len(p.out))
	for _, r := range p.resources {
		r.Close()
		wg.Done()
	}
	wg.Wait()
	return nil
}

func (p *pool) Max() uint {
	return p.max
}

func (p *pool) Available() int {
	return len(p.resources)
}
