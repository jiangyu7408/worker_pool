package worker_pool

import (
	"fmt"
	"runtime"

	"golang.org/x/tools/go/ssa/interp/testdata/src/errors"
)

var (
	DebugMode bool
)

type JobID string
type JobData interface{}

type JobId interface {
	Id() JobID
}

type JobHandler interface {
	Handle() JobData
}

type JobResult struct {
	Id   JobID
	Data JobData
}

type Job interface {
	JobId
	JobHandler
}

type worker struct {
	id    int
	times int
}

func newWorker(wid int) *worker {
	w := new(worker)
	w.id = wid

	return w
}

func debug(format string, a ...interface{}) {
	if DebugMode {
		fmt.Printf(format, a...)
	}
}

func (w *worker) String() string {
	return fmt.Sprintf("[%p] id=%d, times=%d", w, w.id, w.times)
}

func (w *worker) run(job Job) *JobResult {
	w.times++
	debug("worker %d on job %s\n", w.id, job.Id())
	data := &JobResult{job.Id(), job.Handle()}
	debug("worker processed data %p, %v\n", data, *data)
	return data
}

type Manager struct {
	workers  []*worker
	request  chan int
	dispatch chan *worker
	report   chan *worker
}

func NewManager(workerCnt int) *Manager {
	if workerCnt <= 0 {
		workerCnt = runtime.NumCPU()
	}
	m := new(Manager)
	m.init(workerCnt)

	return m
}

func (m *Manager) init(workerCnt int) {
	for wid := 1; wid <= workerCnt; wid++ {
		m.workers = append(m.workers, newWorker(wid))
	}
	m.request = make(chan int)
	m.dispatch = make(chan *worker)
	m.report = make(chan *worker)
}

func (m *Manager) assign() *worker {
	m.request <- 1
	return <-m.dispatch
}

func (m *Manager) recycle(worker *worker) {
	m.report <- worker
}

func (m *Manager) stop() {
	m.request <- 0
}

func (m *Manager) String() string {
	if len(m.workers) == 0 {
		return "no-worker-available"
	}
	s := ""
	for _, w := range m.workers {
		s += fmt.Sprintf("worker => %v\n", w)
	}
	return s
}

func (m *Manager) useWorker() *worker {
	if len(m.workers) > 0 {
		w := m.workers[0]
		m.workers = m.workers[1:]
		return w
	}
	return nil
}

func (m *Manager) returnWorker(w *worker) bool {
	m.workers = append(m.workers, w)
	return true
}

func (m *Manager) run() {
	pending := 0
	for {
		select {
		case r := <-m.request:
			if r == 0 {
				debug("manager stopped")
				return
			}
			if w := m.useWorker(); w != nil {
				m.dispatch <- w
				continue
			}
			pending++
		case w := <-m.report:
			if pending > 0 {
				m.dispatch <- w
				pending--
				continue
			}
			m.returnWorker(w)
		}
	}
}

func verifyJobs(jobs *[]Job) error {
	ids := make(map[JobID]bool)
	for _, job := range *jobs {
		if _, ok := ids[job.Id()]; ok {
			return errors.New("dup job id found")
		}
		ids[job.Id()] = true
	}
	return nil
}

func (m *Manager) Handle(jobs *[]Job) *map[JobID]JobData {
	if ok := verifyJobs(jobs); ok != nil {
		panic("bad-job-id-found")
	}
	debug("Handle jobs: %v\n", *jobs)
	jobCnt := len(*jobs)
	go m.run()

	c := make(chan *JobResult)

	for _, rawJob := range *jobs {
		job, ok := rawJob.(Job)
		if !ok {
			panic("invalid-job")
		}

		w := m.assign()
		go func(w *worker, job Job) {
			data := w.run(job)
			debug("worker handover data %p\n", data)
			m.recycle(w)
			c <- data
		}(w, job)
	}

	res := make(map[JobID]JobData, jobCnt)
	for done := 0; done < jobCnt; done++ {
		data := <-c
		debug("manager recv data %p %v\n", data, data)
		res[data.Id] = data.Data
	}

	debug("manager stopping")
	m.stop()
	close(c)

	debug("manager send %p\n", &res)
	return &res
}

func ProcessJobs(jobs *[]Job, workerCnt int) *map[JobID]JobData {
	debug("Pending Jobs %v", jobs)

	manager := NewManager(workerCnt)
	res := manager.Handle(jobs)

	return res
}
