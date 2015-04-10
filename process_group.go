package main

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"sync"
	"syscall"
)

type ProcessGroup struct {
	set *processSet
	wg  sync.WaitGroup
}

type processSet struct {
	sync.Mutex
	set map[*os.Process]bool
}

func MakeProcessGroup() *ProcessGroup {
	return &ProcessGroup{
		set: newProcessSet(),
	}
}

func (self *ProcessGroup) StartProcess(c *ProcessConfig) (process *os.Process, err error) {
	self.wg.Add(1)

  // TODO: Remove that
	ioReader, ioWriter, err := os.Pipe()
	if err != nil {
		return nil, err
	}

	env := os.Environ()
	env = append(env, "EINHORN_FDS=3")
	env = append(env, "LISTEN_FDS=1")
	// FIXME: LISTEN_PID

	procAttr := &os.ProcAttr{
		Env:   env,
		Files: []*os.File{os.Stdin, ioWriter, ioWriter, c.sockfile},
		Sys:   &syscall.SysProcAttr{},
	}

	if c.user != nil {
		uid, _ := strconv.Atoi(c.user.Uid)
		gid, _ := strconv.Atoi(c.user.Gid)

		procAttr.Sys.Credential = &syscall.Credential{
			Uid: uint32(uid),
			Gid: uint32(gid),
		}
	}

	log.Println("Starting", c.cmdpath)
	process, err = os.StartProcess(c.cmdpath, []string{}, procAttr)
	if err != nil {
		return
	}

	// Add to set
	self.set.Add(process)

	// TODO: Remove that
	// Prefix stdout and stderr lines with the [pid] and send it to the log
	go logOutput(ioReader, process.Pid, self.wg)

	// Handle the process death
	go func() {
		state, err := process.Wait()

		log.Println(process.Pid, state, err)

		// Remove from set
		self.set.Remove(process)

		// Process is gone
		ioReader.Close()
		self.wg.Done()
	}()

	return
}

func (self *ProcessGroup) SignalAll(signal os.Signal, except *os.Process) {
	self.set.Each(func(process *os.Process) {
		if process != except {
			process.Signal(signal)
		}
	})
}

func (self *ProcessGroup) WaitAll() {
	self.wg.Wait()
}

// A thread-safe process set
func newProcessSet() *processSet {
	set := new(processSet)
	set.set = make(map[*os.Process]bool)
	return set
}

func (self *processSet) Add(process *os.Process) {
	self.Lock()
	defer self.Unlock()

	self.set[process] = true
}

func (self *processSet) Each(fn func(*os.Process)) {
	self.Lock()
	defer self.Unlock()

	for process, _ := range self.set {
		fn(process)
	}
}

func (self *processSet) Remove(process *os.Process) {
	self.Lock()
	defer self.Unlock()
	delete(self.set, process)
}

func logOutput(input *os.File, pid int, wg sync.WaitGroup) {
	var err error
	var line string
	wg.Add(1)

	reader := bufio.NewReader(input)

	for err == nil {
		line, err = reader.ReadString('\n')
		if line != "" {
			log.Printf("[%d] %s", pid, line)
		}
	}

	wg.Done()
}
