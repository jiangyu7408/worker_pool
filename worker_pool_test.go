package worker_pool

import (
	"fmt"
	"reflect"
	"testing"
)

func Test_newWorker(t *testing.T) {
	type args struct {
		wid int
	}

	w := new(worker)
	w.id = 1

	tests := []struct {
		name string
		args args
		want *worker
	}{
		{"test", args{1}, w},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newWorker(tt.args.wid); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newWorker() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Benchmark_worker(b *testing.B) {
	for i := 0; i < b.N; i++ {
		w := new(worker)
		w.id = i
	}
}

func Example_debug1() {
	DebugMode = true
	debug("%s", "ss")
	// Output: ss
}

func Example_debug2() {
	DebugMode = false
	debug("%s", "ss")
	// Output:
}

type TestJob struct {
	id  JobID
	cnt int
}

func (job TestJob) Id() JobID {
	return job.id
}

func (job TestJob) Handle() JobData {
	return fmt.Sprintf("cnt=%d", job.cnt)
}

func Test_worker_run(t *testing.T) {
	type args struct {
		job Job
	}

	tests := []struct {
		name string
		w    *worker
		args args
		want *JobResult
	}{
		{
			"test",
			newWorker(1),
			args{TestJob{JobID("1"), 100}},
			&JobResult{"1", "cnt=100"},
		},
	}

	DebugMode = false
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.w.run(tt.args.job); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("worker.run() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Benchmark_worker_run(b *testing.B) {
	for i := 0; i < b.N; i++ {
		w := newWorker(i)
		w.run(TestJob{JobID(i), 100})
	}
}

func TestProcessJobs(t *testing.T) {
	type args struct {
		jobs      *[]Job
		workerCnt int
	}

	jobId := JobID("1")
	jobs := make([]Job, 1)
	jobs[0] = TestJob{jobId, 100}

	want := make(map[JobID]JobData)
	want[jobId] = "cnt=100"

	tests := []struct {
		name string
		args args
		want *map[JobID]JobData
	}{
		{
			name: "test",
			args: args{&jobs, 1},
			want: &want,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ProcessJobs(tt.args.jobs, tt.args.workerCnt); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ProcessJobs() = %v, want %v", got, tt.want)
			}
		})
	}
}
