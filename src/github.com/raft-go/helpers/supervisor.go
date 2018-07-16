package helpers

import (
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

type ResartType int

const (
	ONEFORONE ResartType = iota // restart crashed/killed goroutine
	ALLFORONE                   // restart all goroutine
	ESCALATE                    // crash supervisor(and let parent if any handle)
)

//restart child based on RestartType
type SuperVisorStrategy struct {
	MaxRetries      uint
	BackoffInterval time.Duration
	RestartType     ResartType
}

// A child can set exitCode variable in returnCondition.By deaflt its -1, if 0 sup
// won't start child.
type ChildSpec struct {
	Name               string
	StartFunc          func(chan bool, ...interface{}) int
	Args               []interface{} //args to start func
	SuperVisorStrategy SuperVisorStrategy
	Signal             chan bool //to signal channel to shutdown, not used now
}

type childStatus struct {
	numRestarts    uint
	isDone         bool
	isRunning      bool
	mergedStrategy SuperVisorStrategy
}

//TODO implement supervisor-like
type superVisor struct {
	name               string
	childSpecs         []ChildSpec
	SuperVisorStrategy //used to fill in missing values for child
	childStatuses      sync.Map
	allDone            chan interface{} // a signal variable to indicate all done
	allDoneGuard       sync.Once
}

// ??? Should we return via pointer?
func New(name string, childSpecs []ChildSpec, defaultStrategy SuperVisorStrategy, signal chan interface{}) *superVisor {
	s := superVisor{name, childSpecs, defaultStrategy,
		sync.Map{}, signal, sync.Once{}}

	go func() {
		for x := range time.Tick(1 * time.Second) {
			_ = x
			logrus.Infof("Checking if supervisor: %v is DONE...", s.name)
			if s.areWeDone() == true {
				s.setDone()
			}
		}
	}()

	return &s
}

/************** 	Supervisor methods      **************/

/****** exported methods ******/
func (s *superVisor) Start() {
	logrus.Infof("Starting supervisor: %v", s.name)

	//start all child
	for _, c := range s.childSpecs {
		strategy := s.mergedStrategy(c.SuperVisorStrategy)
		go s.startChild(c, strategy)
	}
}

/****** internal methods ******/

func (s *superVisor) areWeDone() bool {
	needToEnd := true
	s.childStatuses.Range(func(k, v interface{}) bool {
		child := v.(childStatus)
		//if a single child running, return and wait for it
		if child.isDone == false {
			needToEnd = false
		}
		return true
	})
	return needToEnd
}

func (s *superVisor) startChild(c ChildSpec, strategy SuperVisorStrategy) {
	// should we instead use a channel for communicating exitStatus?? That would
	// require a supervisor goroutine
	exitStatus := -1
	//check if we already have childSTatus, if we are repeating child
	maybeChildStatus, ok := s.childStatuses.Load(c.Name)
	childS := childStatus{}
	if ok {
		childS = maybeChildStatus.(childStatus)
	}

	childS.mergedStrategy = strategy
	childS.numRestarts++
	childS.isDone = false
	childS.isRunning = true
	//store update childStatus
	s.childStatuses.Store(c.Name, childS)

	defer func() {
		recover() //TODO catch child panic unless child indicates not to, by having a type switch and a flag
		logrus.Infof("Child %v stopped with %d", c.Name, exitStatus)
		//notify supervisor we are done passing the exitStatus
		s.childDone(c, exitStatus)
	}()

	// call child func
	exitStatus = c.StartFunc(c.Signal, c.Args...)
	logrus.Infof("Function %v returned with exitStatus: %v", c.Name, exitStatus)
}

//This is unused for now
func (s *superVisor) shutdownChildren() {
	runningChildren := make(map[string]bool)

	s.childStatuses.Range(func(k, v interface{}) bool {
		name := k.(string)
		child := v.(childStatus)
		if child.isRunning && !child.isDone {
			logrus.Infof("Adding %v to runningChildren", name)
			runningChildren[name] = true
		}
		return true
	})

	logrus.Infof("Children to shut: %v", runningChildren)

	for _, child := range s.childSpecs {
		if _, ok := runningChildren[child.Name]; ok {
			logrus.Infof("Sending signal to %v", child.Name)
			child.Signal <- true //signal child to shutdown
		}
	}

	for _, child := range s.childSpecs {
		if _, ok := runningChildren[child.Name]; ok {
			<-child.Signal
		}
	}

	logrus.Infof("Shut down all child...")
}

//can be called from multiple Goroutine, hence use concurrent DS for supervisor
func (s *superVisor) childDone(c ChildSpec, exitStatus int) {
	logrus.Infof("Child %v done with exitStatus: %d", c.Name, exitStatus)
	t, _ := s.childStatuses.Load(c.Name)
	childS := t.(childStatus) //this is expected not to fail here
	childS.isRunning = false
	s.childStatuses.Store(c.Name, childS)
	strategy := childS.mergedStrategy

	if exitStatus == 0 || childS.numRestarts > strategy.MaxRetries {
		logrus.Infof("Child %v exited with status: %v with restarts(%v)/retries(%v)", c.Name, exitStatus,
			childS.numRestarts, strategy.MaxRetries)
		t, _ := s.childStatuses.Load(c.Name)
		childS := t.(childStatus)
		childS.isDone = true
		s.childStatuses.Store(c.Name, childS)
		if s.areWeDone() == true {
			//cancel timer too
			s.setDone()
		}
		return
	}

	if strategy.RestartType == ESCALATE {
		logrus.Infof("Shutting down all running child of %v", s.name)
		s.shutdownChildren()
		s.setDone()
		return
	}
	logrus.Infof("Restarting child: %v after %v seconds", c.Name, strategy.BackoffInterval)
	//the assumption here is we only have ONEFORONE strategy, in case of ALLFORONE we still may not want to restart
	// succeeded child
	time.AfterFunc(strategy.BackoffInterval*time.Millisecond, func() {
		strategy := s.mergedStrategy(c.SuperVisorStrategy)
		go s.startChild(c, strategy)
	})
}

func (s *superVisor) setDone() {
	s.allDoneGuard.Do(func() {
		s.allDone <- true
	})
}

func (s *superVisor) mergedStrategy(c SuperVisorStrategy) SuperVisorStrategy {
	mergedStrategy := s.SuperVisorStrategy
	if c.BackoffInterval != 0 {
		mergedStrategy.BackoffInterval = c.BackoffInterval
	}

	if c.MaxRetries != 0 {
		mergedStrategy.MaxRetries = c.MaxRetries
	}

	if c.RestartType != ONEFORONE {
		mergedStrategy.RestartType = c.RestartType
	}
	return mergedStrategy
}
