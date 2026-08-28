package engine

import "sync"

// call dai dien cho mot tac vu dang duoc thuc thi
type call struct {
	wg  sync.WaitGroup
	val any
	err error
}

// Group quan ly viec gop cac loi goi ham trung lap
type Group struct {
	mu sync.Mutex
	m  map[string]*call
}

func NewSingleFlightGroup() *Group {
	return &Group{
		m: make(map[string]*call),
	}
}

// Do dam bao chi co 1 ham fn duoc thuc thi cho moi key tai mot thoi diem
func (g *Group) Do(key string, fn func() (any, error)) (any, error) {
	g.mu.Lock()
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err
	}

	c := new(call)
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()

	c.val, c.err = fn()
	c.wg.Done()

	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()

	return c.val, c.err
}
