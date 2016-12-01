package hottub

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestCreatePool(t *testing.T) {
	fmt.Println("TestCreatePool")

	_, err := NewPool(nil, &PoolParams{})
	if err != ErrGeneratorRequired {
		t.Errorf("NewPool with nil generator: expected error %s, instead got %s", ErrGeneratorRequired, err)
	}

	var tests = []struct {
		params             PoolParams
		expectedMax        uint
		expectedTimeout    time.Duration
		expectedAliveCheck time.Duration
	}{
		{PoolParams{0, DefaultTakeTimeout, DefaultAliveCheck}, DefaultMaxResources, DefaultTakeTimeout, DefaultAliveCheck},
		{PoolParams{DefaultMaxResources, 0, DefaultAliveCheck}, DefaultMaxResources, DefaultTakeTimeout, DefaultAliveCheck},
		{PoolParams{DefaultMaxResources, DefaultTakeTimeout, 0}, DefaultMaxResources, DefaultTakeTimeout, DefaultAliveCheck},
		{PoolParams{uint(5), 0, 0}, uint(5), DefaultTakeTimeout, DefaultAliveCheck},
		{PoolParams{DefaultMaxResources, time.Millisecond * 20, DefaultAliveCheck}, DefaultMaxResources, time.Millisecond * 20, DefaultAliveCheck},
		{PoolParams{DefaultMaxResources, DefaultTakeTimeout, time.Millisecond * 20}, DefaultMaxResources, DefaultTakeTimeout, time.Millisecond * 20},
	}
	for i, test := range tests {
		newPool, err := NewPool(testGenerator(), &test.params)
		if err != nil {
			t.Errorf("There was an error creating pool for case %d: %s", i+1, err)
			continue
		}
		p := newPool.(*pool)
		if p.Max() != test.expectedMax {
			t.Errorf("Test %d: Expected max items: %d, got %d", i+1, test.expectedMax, p.timeout)
		}
		if p.timeout != test.expectedTimeout {
			t.Errorf("Test %d: Expected timeout %d, got %d", i+1, test.expectedTimeout, p.timeout)
		}
		if p.check != test.expectedAliveCheck {
			t.Errorf("Test %d: Expcted alive check %d, got %d", i+1, test.expectedAliveCheck, p.check)
		}
	}
}

func TestTake(t *testing.T) {
	fmt.Println("TestTake")
	params := &PoolParams{
		Timeout:      time.Millisecond * 50,
		MaxResources: uint(1),
	}
	p := assertNewPool(t, testGenerator(), params)
	assertTakeResource(t, p) // take the first resource
	_, err := p.Take()
	if err != ErrTimeoutTakingResource {
		t.Errorf("Expected error taking from pool: %s, got %s", ErrTimeoutTakingResource, err)
	}
}

func TestReturn(t *testing.T) {
	fmt.Println("TestReturn")
	params := &PoolParams{
		MaxResources: uint(2),
	}
	p := assertNewPool(t, testGenerator(), params)
	assertAvailableResources(t, p, 2)
	r := assertTakeResource(t, p)
	assertAvailableResources(t, p, 1)
	assertReturnResource(t, p, r)
	assertAvailableResources(t, p, 2)
}

func TestTakeToFullAndReturn(t *testing.T) {
	fmt.Println("TestTakeToFullAndReturn")
	p := assertNewPoolDefaultParams(t, testGenerator())
	r1 := assertTakeResource(t, p)
	assertTakeResource(t, p)
	assertTakeResource(t, p)
	assertTakeResource(t, p)
	assertTakeResource(t, p)
	wg := sync.WaitGroup{}
	wg.Add(2)
	var r6 Resource
	var err6 error
	go func() {
		r6, err6 = p.Take()
		wg.Done()
	}()
	time.Sleep(time.Millisecond * 10)
	go func() {
		assertReturnResource(t, p, r1)
		wg.Done()
	}()
	wg.Wait()
	if r6 != r1 {
		t.Errorf("Expected to get last returned resource after taking 'til empty, r1:%s, r2:%s", r1, r6)
	}
}

func TestUnmanagedResource(t *testing.T) {
	fmt.Println("TestUnmanagedResource")
	p := assertNewPoolDefaultParams(t, testGenerator())
	unmanaged := testGenerator()()
	err := p.Return(unmanaged)
	if err != ErrUnmanagedResource {
		t.Errorf("Expected returned error to be %s, got %s instead", ErrUnmanagedResource, err)
	}
}

func TestManagedResource(t *testing.T) {
	fmt.Println("TestManagedResource")
	p := assertNewPoolDefaultParams(t, testGenerator())
	r1 := assertTakeResource(t, p)
	unmanaged := testGenerator()()
	if !p.Manages(r1) {
		t.Error("Pool should be managing this resource")
	}
	if p.Manages(unmanaged) {
		t.Error("Pool should not be managing this resource")
	}
}

func TestResourceNotAlive(t *testing.T) {
	fmt.Println("TestResourceNotAlive")
	p := assertNewPoolDefaultParams(t, testGenerator2())
	for i := uint(0); i < p.Max(); i++ {
		r, err := p.Take()
		fmt.Printf("r:%s, err:%s\n", r, err)
	}
}

func TestReturnTakeAfterTimeout(t *testing.T) {
	fmt.Println("TestReturnTakeAfterTimeout")
	p := assertNewPoolDefaultParams(t, testGenerator())
	rs := assertDrainPool(t, p)
	wg := sync.WaitGroup{}
	wg.Add(3)
	var timeout1, timeout2, noTimeout Resource
	var err1, err2, err3 error
	go func() {
		timeout1, err1 = p.Take()
		fmt.Printf("1st reply:%s %s\n", timeout1, err1)
		wg.Done()
	}()
	go func() {
		timeout2, err2 = p.Take()
		fmt.Printf("2nd reply:%s %s\n", timeout2, err2)
		wg.Done()
	}()
	time.Sleep(20 * time.Millisecond)
	go func() {
		noTimeout, err3 = p.Take()
		fmt.Printf("3rd reply:%s %s\n", noTimeout, err3)
		wg.Done()
	}()
	time.Sleep(40 * time.Millisecond)
	p.Return(rs[0])
	wg.Wait()

	assertErrorReturned(t, err1, ErrTimeoutTakingResource)
	assertErrorReturned(t, err2, ErrTimeoutTakingResource)
	assertErrorReturned(t, err3, nil)
	assertResourceReturned(t, timeout1, nil)
	assertResourceReturned(t, timeout2, nil)
	assertResourceReturned(t, noTimeout, rs[0])
}

func TestClose(t *testing.T) {
	fmt.Println("TestClose")
	// params := &PoolParams{
	// 	MaxResources: testUintPtr(5),
	// }
	// p := assertNewPool(t, testGenerator, params)
	// assertAvailableResources(t, p, 5)
	// p.Close()
	// assertAvailableResources(t, p, 0)
}

func assertNewPoolDefaultParams(t *testing.T, g Generator) Pool {
	params := &PoolParams{
		MaxResources: uint(5),
		Timeout:      50 * time.Millisecond,
	}
	return assertNewPool(t, g, params)
}

func assertNewPool(t *testing.T, g Generator, params *PoolParams) Pool {
	p, err := NewPool(g, params)
	if err != nil {
		t.Error(err)
	}
	return p
}

func assertDrainPool(t *testing.T, p Pool) []Resource {
	var rs []Resource
	for i := uint(0); i < p.Max(); i++ {
		r := assertTakeResource(t, p)
		rs = append(rs, r)
	}
	return rs
}

func assertTakeResource(t *testing.T, p Pool) Resource {
	r, err := p.Take()
	if err != nil {
		t.Error(err)
	}
	return r
}

func assertReturnResource(t *testing.T, p Pool, r Resource) error {
	err := p.Return(r)
	if err != nil {
		t.Error(err)
		return err
	}
	return err
}

func assertAvailableResources(t *testing.T, p Pool, available int) {
	if p.Available() != available {
		t.Errorf("Expected available resources %d, instead got %d", available, p.Available())
	}
}

func assertResourceReturned(t *testing.T, returned, expected Resource) {
	if returned != expected {
		t.Errorf("Expected resource %p returned, instead got %p", &expected, &returned)
	}
}

func assertErrorReturned(t *testing.T, err, expected error) {
	if err != expected {
		t.Errorf("Expected error %s returned, not %s", expected, err)
	}
}

type testResource struct {
	id     int
	alive  func() (bool, error)
	closer func() error
}

func (tr *testResource) Alive() (bool, error) {
	return tr.alive()
}

func (tr *testResource) Close() error {
	return tr.closer()
}

func (tr *testResource) String() string {
	return fmt.Sprintf("%d", tr.id)
}

var testAlive1 = func() (bool, error) {
	return true, nil
}

var errTestAlive = errors.New("An error!")

var testAlive2 = func(i int) func() (bool, error) {
	return func() (bool, error) {
		return i%2 == 0, errTestAlive
	}
}

var testClose1 = func() error {
	return nil
}

var testGenerator = func() Generator {
	i := 0
	return func() Resource {
		i++
		return &testResource{i, testAlive1, testClose1}
	}
}

var testGenerator2 = func() Generator {
	i := 0
	return func() Resource {
		i++
		return &testResource{i, testAlive2(i), testClose1}
	}
}
