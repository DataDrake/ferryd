//
// Copyright © 2017 Ikey Doherty <ikey@solus-project.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package jobs

import (
	"ferryd/core"
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
	"reflect"
	"runtime"
	"sync"
	"time"
)

// A Processor is responsible for the main dispatch and bulking of jobs
// to ensure they're handled in the most optimal fashion.
type Processor struct {
	manager        *core.Manager
	sequentialjobs chan *Job
	backgroundJobs chan *Job
	quit           chan bool
	mut            *sync.RWMutex
	wg             *sync.WaitGroup
	closed         bool
	njobs          int
	jobTable       map[string]*Job
}

// NewProcessor will return a new Processor with the specified number
// of jobs. Note that "njobs" only refers to the number of *background jobs*,
// the majority of operations will run sequentially
func NewProcessor(m *core.Manager, njobs int) *Processor {
	if njobs < 0 {
		njobs = runtime.NumCPU()
	}

	fmt.Fprintf(os.Stderr, "Capped backgroundJobs to %d\n", njobs)

	ret := &Processor{
		manager:        m,
		sequentialjobs: make(chan *Job),
		backgroundJobs: make(chan *Job),
		quit:           make(chan bool, 1+njobs),
		mut:            &sync.RWMutex{},
		wg:             &sync.WaitGroup{},
		closed:         false,
		njobs:          njobs,
		jobTable:       make(map[string]*Job),
	}
	return ret
}

// Close an existing Processor, waiting for all jobs to complete
func (j *Processor) Close() {
	j.mut.Lock()
	defer j.mut.Unlock()
	if j.closed {
		return
	}
	j.closed = true

	// Disallow further messaging
	close(j.sequentialjobs)
	close(j.backgroundJobs)

	// Ensure all goroutines get the quit broadcast
	for i := 0; i < j.njobs+1; i++ {
		j.quit <- true
	}
	j.wg.Wait()
}

// Begin will start the main job processor in parallel
func (j *Processor) Begin() {
	j.mut.Lock()
	defer j.mut.Unlock()
	if j.closed {
		return
	}
	j.wg.Add(2)
	go j.processSequentialQueue()
	go j.processBackgroundQueue()
}

// reportError will report a failed job to the log
func (j *Processor) reportError(job *Job, e error) {
	job.status = StatusFailed
	log.WithFields(log.Fields{
		"id":    job.id,
		"error": e,
		"type":  reflect.TypeOf(job.task).Elem().Name(),
	}).Error("Job failed with error")
}

// executeJob will execute a single job and update the meta information
// for it.
func (j *Processor) executeJob(job *Job) {
	job.timing.Started = time.Now()
	job.status = StatusRunning
	err := job.task.Perform(j.manager)
	job.timing.Completed = time.Now()

	parents := job.NotifyDone()

	// If any parent tasks have been freed at this point, put them back into
	// execution
	if parents != nil {
		for _, p := range parents {
			go j.StartJob(p)
		}
	}

	j.popJob(job)

	if err != nil {
		j.reportError(job, err)
		return
	}

	job.status = StatusSuccess
}

// popJob will remove the job from the internal state processor
func (j *Processor) popJob(job *Job) {
	j.mut.Lock()
	defer j.mut.Unlock()
	delete(j.jobTable, job.id)
}

// processSequentialQueue is responsible for dealing with the sequential queue
func (j *Processor) processSequentialQueue() {
	defer j.wg.Done()

	for {
		select {
		case job := <-j.sequentialjobs:
			if job == nil {
				return
			}
			j.executeJob(job)
		case <-j.quit:
			return
		}
	}
}

// processBackgroundQueue will set up the background workers which will block
// waiting for non-sequential work that cannot run on the main queue, however
// it may put work back on the sequential queue.
func (j *Processor) processBackgroundQueue() {
	defer j.wg.Done()
	j.wg.Add(j.njobs)

	for i := 0; i < j.njobs; i++ {
		go j.backgroundWorker()
	}
}

// backgroundWorker will handle the non sequential tasks as and when they come
// in. The majority of tasks will be sequential on the main queue, so we're free
// to spend more CPU time here dealing with large tasks like the construction
// of delta packages.
func (j *Processor) backgroundWorker() {
	defer j.wg.Done()

	for {
		select {
		case job := <-j.backgroundJobs:
			if job == nil {
				return
			}
			j.executeJob(job)
		case <-j.quit:
			return
		}
	}
}

// initMetadata assigns the initial ID bits
// TODO: Consider using UUID and stop being hacky.
func (j *Processor) initMetadata(job *Job) {
	now := time.Now()

	j.mut.RLock()
	defer j.mut.RUnlock()

	counter := 0
	job.timing.Created = now
	job.status = StatusPending

	nom := reflect.TypeOf(job.task).Elem().Name()
	unix := now.UTC().Unix()

	for {
		job.id = fmt.Sprintf("%s-%d-%d", nom, unix, counter)
		if _, ok := j.jobTable[job.id]; !ok {
			return
		}
		counter++
	}
}

// pushJobInternal is where the real work happens
func (j *Processor) pushJobInternal(task Runnable) *Job {
	if task == nil {
		panic("passed nil job!")
	}

	// We might spin here to get a valid ID, so we won't write lock and other
	// jobs can still be added
	job := &Job{
		task: task,
	}
	job.init()
	j.initMetadata(job)
	job.task.Init(j)

	j.mut.Lock()
	defer j.mut.Unlock()

	// We're closed, sorry.
	if j.closed {
		return job
	}

	j.jobTable[job.id] = job

	return job
}

// PushJobLater will add the job to the system but won't start it, with the
// assumption that it has dependencies
func (j *Processor) PushJobLater(task Runnable) *Job {
	return j.pushJobInternal(task)
}

// PushJob will take the new job and push it to the appropriate queing system
// For sanity reasons this will lock on the new job add, even if the processing
// is then parallel.
func (j *Processor) PushJob(task Runnable) *Job {
	job := j.pushJobInternal(task)
	j.StartJob(job)
	return job
}

// StartJob will actually set the job up for execution
func (j *Processor) StartJob(job *Job) {
	j.mut.RLock()
	defer j.mut.RUnlock()

	if j.closed {
		return
	}

	// Stick the jobs in the queue now
	if job.task.IsSequential() {
		j.sequentialjobs <- job
	} else {
		j.backgroundJobs <- job
	}
}
