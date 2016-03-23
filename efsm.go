package efsm

import (
	"sync"
	"fmt"
	"time"
	"log"
)
// TODO Event should be more than a string
type Event string

type Transition func(f *FSM, s *State, e Event) *State

// just here so that we know what string we are talking about
type StateName string

type State struct {
	Name         StateName
	StateTimeout *Timeout
	timers       map[StateName]*time.Timer
	transitions  map[Event]Transition
	Data         interface{}
}

type Timeout struct {
	EventName string
	duration  time.Duration
}

// state x event => action => [state',...]
// given a state,  and an event, determine an action/transition to one of several possible states


type FSM struct {
	sync.Mutex
	CurrentState StateName
	States       map[StateName]*State
	Events       chan Event
}

func (f *FSM) setCurrentState(newState *State) error {
	if s, ok := f.States[newState.Name]; ok {
		f.CurrentState = newState.Name
		s.Data = newState.Data
		if s.StateTimeout == nil {
			s.StateTimeout = newState.StateTimeout
		}

		if s.StateTimeout != nil {
			if t, ok := s.timers[s.Name]; ok {
				// if there is already a timer running for this state - stop it first
				t.Stop()
				s.StateTimeout = newState.StateTimeout
			}
			timer := time.NewTimer(s.StateTimeout.duration)
			s.timers[s.Name] = timer
			go func() {
				<-timer.C
				log.Printf("Timeout: %s", s.StateTimeout.EventName)
				f.Events <- Event(s.StateTimeout.EventName)
				// timer expired or stopped - delete from map
				delete(s.timers, s.Name)
			}()
		}

	} else {
		return fmt.Errorf("Invalid State")
	}
	return nil
}

func (f *FSM) Goto(name StateName) *State {
	if s, ok := f.States[name]; ok {
		f.CurrentState = name
		return s
	} else {
		panic(fmt.Errorf("Unknown state to transition to: %s", name))
	}
}

// TODO allow the specification of a default timeout for this state like When(name StateName, t Timeout)
func (f *FSM) When(name StateName) *State {
	if _, ok := f.States[name]; ok {
		panic(fmt.Errorf("Duplicate state registration %s", name))
	}
	newState := &State{
		Name:name,
		StateTimeout: nil,
		transitions:  make(map[Event]Transition),
		timers: make(map[StateName]*time.Timer),
	}
	f.States[name] = newState

	return newState
}

func (s *State) Using(v interface{}) *State {
	s.Data = v
	return s
}

func (s *State) ForMax(d time.Duration) *State {
	s.StateTimeout = &Timeout{
		EventName: string(s.Name) + "-timeout",
		duration: d,
	}
	return s
}

func (s *State) Case(events []Event, fn Transition) error {
	for _, e := range events {
		s.transitions[e] = fn
	}
	return nil
}
func (f *FSM) Engine(c chan Event) error {
	f.Events = c
	f.States = make(map[StateName]*State)
	return nil
}

func (f *FSM) Run(initial StateName) error {
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
				// stopping timer before transition
				if t, ok := s.timers[s.Name]; ok {
					log.Printf("Stoping Timer %s\n", s.Name)
					t.Stop()
					delete(s.timers, s.Name)
				}
			}
			if err := f.setCurrentState(newState); err != nil {
				panic(err)
			}
		}
		f.Unlock()
	}
	// TODO process uncaught events
	return nil
}