package hottub

import (
	"fmt"
	"testing"
	"time"
)

// TestStackSizeEmpty
func TestStackSizeEmpty(t *testing.T) {
	fmt.Println("TestStackSizeEmpty")
	s := newStack()
	if !s.isEmpty() {
		t.Errorf("Expected empty stack, instead got back isEmpty:%t", s.isEmpty())
	}
}

// TestStackPush
func TestStackPush(t *testing.T) {
	fmt.Println("TestStackPush")
	s := newStack()
	s.push(newReceiver())
	assertStackSize(t, s, 1)
}

// TestStackPop
func TestStackPop(t *testing.T) {
	fmt.Println("TestStackPop")
	s := newStack()
	p := newReceiver()
	s.push(p)
	assertStackSize(t, s, 1)
	q := s.pop()
	assertStackSize(t, s, 0)
	if p != q {
		t.Errorf("Expected pushed receiver to be same as popped, pushed:%p, popped:%p", p, q)
	}
}

// TestReceiverTimeout
func TestReceiverTimeout(t *testing.T) {
	fmt.Println("TestReceiverTimeout")
	p := newReceiver()
	p.start(time.Millisecond * 5)
	time.Sleep(time.Millisecond * 20)
	if !p.timedOut {
		t.Error("Expected receiver to time out")
	}
}

func assertStackSize(t *testing.T, s stack, expected int) {
	if s.size() != expected {
		t.Errorf("Expected stack size %d, got %d", expected, s.size())
	}
}
