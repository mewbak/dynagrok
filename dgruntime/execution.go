package dgruntime

import (
	"fmt"
	"os"
	"sync"
	"runtime"
)

type Execution struct {
	m sync.Mutex
	Goroutines []*Goroutine
	Profile    *Profile
	OutputPath string
	mergeCh    chan *Goroutine
	async      sync.WaitGroup
}

var execMu sync.Mutex
var exec *Execution

func execCheck() {
	if exec == nil {
		execMu.Lock()
		if exec == nil {
			exec = newExecution()
		}
		runtime.SetFinalizer(exec, shutdown)
		execMu.Unlock()
	}
}

func newExecution() *Execution {
	output := "/tmp/dynagrok-profile.dot"
	if os.Getenv("DGPROF") != "" {
		output = os.Getenv("DGPROF")
	}
	e := &Execution{
		Profile: &Profile{
			Calls: make(map[Call]int),
			Funcs: make(map[uintptr]*Function),
		},
		OutputPath: output,
		mergeCh: make(chan *Goroutine, 15),
	}
	e.growGoroutines()
	go func() {
		e.async.Add(1)
		for g := range e.mergeCh {
			e.merge(g)
		}
		e.async.Done()
	}()
	return e
}

func (e *Execution) Goroutine(id int64) *Goroutine {
	for id >= int64(len(e.Goroutines)) {
		e.m.Lock()
		for id >= int64(len(e.Goroutines)) {
			e.growGoroutines()
		}
		e.m.Unlock()
	}
	if e.Goroutines[id] == nil {
		e.m.Lock()
		if e.Goroutines[id] == nil {
			// Println(fmt.Sprintf("new goroutine %d", id))
			e.Goroutines[id] = newGoroutine(id)
		}
		e.m.Unlock()
	}
	return e.Goroutines[id]
}

func (e *Execution) growGoroutines() {
	n := make([]*Goroutine, (len(e.Goroutines)+1)*2)
	copy(n, e.Goroutines)
	// for i := len(e.Goroutines); i < len(n); i++ {
	// 	n[i] = newGoroutine(int64(i))
	// }
	e.Goroutines = n
}

func (e *Execution) Merge(g *Goroutine) {
	e.mergeCh<-g
}

func (e *Execution) merge(g *Goroutine) {
	e.m.Lock()
	defer e.m.Unlock()
	if !g.Closed {
		return
	}
	e.Profile.CallCount += g.CallCount
	for _, fn := range g.Funcs {
		if x, has := e.Profile.Funcs[fn.FuncPc]; has {
			x.Merge(fn)
		} else {
			e.Profile.Funcs[fn.FuncPc] = fn
		}
	}
	for call, count := range g.Calls {
		e.Profile.Calls[call] += count
	}
}

func shutdown(e *Execution) {
	fmt.Println("starting shut down")
	execMu.Lock()
	defer execMu.Unlock()
	if e == nil {
		return
	}
	for _, g := range e.Goroutines {
		if g == nil {
			continue
		}
		g.m.Lock()
		if !g.Closed && len(g.Calls) > 0 {
			g.m.Unlock()
			g.Exit()
		}
	}
	close(e.mergeCh)
	e.async.Wait()
	e.m.Lock()
	defer e.m.Unlock()
	fmt.Println("writing to:", e.OutputPath)
	fout, err := os.Create(e.OutputPath)
	if err != nil {
		panic(err)
	}
	e.Profile.Serialize(fout)
	fout.Close()
	fmt.Println("done shutting down")
}
