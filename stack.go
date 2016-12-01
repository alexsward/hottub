package hottub

import "time"

type receiver struct {
	timedOut, replied bool
	resource          chan Resource
	response          chan error
}

func newReceiver() *receiver {
	return &receiver{
		resource: make(chan Resource),
		response: make(chan error),
	}
}

func (r *receiver) start(d time.Duration) {
	go func() {
		var c <-chan time.Time
		if d > 0 {
			tick := time.NewTicker(d)
			c = tick.C
			defer tick.Stop()
		}
		for {
			select {
			case <-c:
				r.timedOut = true
				r.response <- ErrTimeoutTakingResource
			}
		}
	}()
}

type stack []*receiver

func newStack() stack {
	return make(stack, 0)
}

func (s *stack) isEmpty() bool {
	return s.size() == 0
}

func (s *stack) size() int {
	return len(*s)
}

func (s *stack) pop() *receiver {
	idx := len((*s)) - 1
	popped := (*s)[idx]
	(*s)[idx] = nil
	(*s) = (*s)[:idx]
	return popped
}

func (s *stack) push(p *receiver) {
	(*s) = append(*s, nil)
	copy((*s)[1:], (*s)[0:])
	(*s)[0] = p
}

type resources []Resource

func (r *resources) pop() Resource {
	return r.remove(len(*r) - 1)
}

func (r *resources) push(res Resource) {
	(*r) = append(*r, nil)
	copy((*r)[1:], (*r)[0:])
	(*r)[0] = res
}

func (r *resources) indexOf(res Resource) (int, bool) {
	for i := range *r {
		if res == (*r)[i] {
			return i, true
		}
	}
	return len(*r), false
}

func (r *resources) remove(idx int) Resource {
	removed := (*r)[idx]
	copy((*r)[idx:], (*r)[idx+1:])
	(*r)[len(*r)-1] = nil
	*r = (*r)[:len(*r)-1]
	return removed
}
