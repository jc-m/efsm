package fsm

import (
	"sync"
	"fmt"
	"time"
	"log"
)


type Event string

type State struct {
	Name        string
	timeout     *Timeout
	transitions map[Event]func(f *FSM, s *State, e Event) *State
	Value       interface{}
}

type Timeout struct {
	EventName string
	duration  time.Duration
}

// state x event => action => [state',...]
// given a state,  and an event, determine an action/transition to one of several possible states


type FSM struct {
	sync.Mutex
	CurrentState string
	States       map[string]*State
	Events       chan Event
}

func (f *FSM) setCurrentState(newState *State) error {
	if s, ok := f.States[newState.Name]; ok {
		f.CurrentState = newState.Name
		if newState.timeout != nil {
			s.timeout = newState.timeout
		}
		s.Value = newState.Value
		if s.timeout != nil {
			timer := time.NewTimer(s.timeout.duration)
			go func() {
				<-timer.C
				log.Printf("Timeout: %s", s.timeout.EventName)
				f.Events <- Event(s.timeout.EventName)
			}()
		}

	} else {
		return fmt.Errorf("Invalid State")
	}
	return nil
}

func (f *FSM) Goto(name string) *State {
	if s, ok := f.States[name]; ok {
		f.CurrentState = name
		return s
	} else {
		panic(fmt.Errorf("Unknown state to transition to: %s", name))
	}
}

func (f *FSM) When(name string) *State {
	if _, ok := f.States[name]; ok {
		panic(fmt.Errorf("Duplicate state registration %s", name))
	}
	newState := &State{
		Name:name,
		timeout: nil,
		transitions:  make(map[Event]func(f *FSM, s *State, e Event) *State),
	}
	f.States[name] = newState

	return newState
}

func (s *State) With(v interface{}) *State {
	s.Value = v
	return s
}

func (s *State) ForMax(d time.Duration) *State {
	s.timeout = &Timeout{
		EventName: s.Name + "-timeout",
		duration: d,
	}
	return s
}

func (s *State) Case(events []Event, fn func(f *FSM, s *State, e Event) *State ) error {
	for _, e := range events {
		s.transitions[e] = fn
	}
	return nil
}
func (f *FSM) Engine(c chan Event) error {
	f.Events = c
	f.States = make(map[string]*State)
	return nil
}

func (f *FSM) Run(initial string) error {
	log.Printf("In Run")
	if s, ok := f.States[initial]; ok {
		f.setCurrentState(s)
	} else {
		panic(fmt.Errorf("Unknown initial state: %s", initial))
	}

	for e := range f.Events {
		f.Lock()
		s := f.States[f.CurrentState]
		if fn, ok := s.transitions[e]; ok {
			newState := fn(f, s, e)
			if newState != nil {
				log.Printf("--> %s\n", newState.Name)
				if err := f.setCurrentState(newState); err != nil {
					panic(err)
				}
			}
		}
		f.Unlock()
		// TODO process uncaught events
	}
	return nil
}