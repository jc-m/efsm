package efsm

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// TODO event matching need to be revisited - for now, only look at the name
type Event struct {
	Name  string
	Scope string
	Data  interface{}
}

type Transition func(f *FSM, s *State, e Event) *State

// just here so that we know what string we are talking about
type StateName string

type State struct {
	Name         StateName
	StateTimeout *Timeout
	timers       map[StateName]*time.Timer
	transitions  map[string]Transition
	Data         interface{}
}

type Timeout struct {
	Event    Event
	duration time.Duration
}

// state x event => action => [state',...]
// given a state,  and an event, determine an action/transition to one of several possible states

type FSM struct {
	ID string
	sync.Mutex
	CurrentState StateName
	States       map[StateName]*State
	In           chan Event
	Out          chan Event
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
				s.StateTimeout.Event.Scope = f.ID
				log.Printf("%s| Timeout: %+v", f.ID, s.StateTimeout.Event)
				f.In <- Event(s.StateTimeout.Event)
				// timer expired or stopped - delete from map
				delete(s.timers, s.Name)
			}()
		}

		log.Printf("%s| --> %s\n", f.ID, newState.Name)
		eOut := Event{
			Name:  string(newState.Name),
			Scope: f.ID,
		}
		// Sending event without blocking

		select {
		case f.Out <- eOut:
			log.Printf("%s| Sending Event: %+v\n", f.ID, eOut)
		default:
		}

	} else {
		return fmt.Errorf("%s| Invalid State", f.ID)
	}
	return nil
}

func (f *FSM) Goto(name StateName) *State {
	if s, ok := f.States[name]; ok {
		f.CurrentState = name
		return s
	} else {
		panic(fmt.Errorf("%s| Unknown state to transition to: %s", f.ID, name))
	}
}

// TODO allow the specification of a default timeout for this state like When(name StateName, t Timeout)
func (f *FSM) When(name StateName) *State {
	if _, ok := f.States[name]; ok {
		return f.States[name]
	}
	newState := &State{
		Name:         name,
		StateTimeout: nil,
		transitions:  make(map[string]Transition),
		timers:       make(map[StateName]*time.Timer),
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
		Event:    Event{Name: string(s.Name) + "-timeout"},
		duration: d,
	}
	return s
}

func (s *State) Case(events []Event, fn Transition) error {
	for _, e := range events {
		if _, ok := s.transitions[e.Name]; ok {
			return fmt.Errorf("Duplicate Event name: %s", e.Name)
		}
		s.transitions[e.Name] = fn
	}
	return nil
}
func (f *FSM) Engine(In, Out chan Event) error {
	f.In = In
	f.Out = Out
	f.States = make(map[StateName]*State)

	return nil
}

func (f *FSM) Run(initial StateName) error {
	log.Printf("%s| Running ....", f.ID)

	if s, ok := f.States[initial]; ok {
		f.setCurrentState(s)
	} else {
		panic(fmt.Errorf("%s| Unknown initial state: %s", f.ID, initial))
	}

	for eIn := range f.In {
		f.Lock()
		log.Printf("%s| Received Event: %+v\n", f.ID, eIn)
		s := f.States[f.CurrentState]
		if fn, ok := s.transitions[eIn.Name]; ok {
			newState := fn(f, s, eIn)
			if newState != nil {
				// stopping timer before transition
				if t, ok := s.timers[s.Name]; ok {
					log.Printf("%s| Stoping Timer %s\n", f.ID, s.Name)
					t.Stop()
					delete(s.timers, s.Name)
				}

				if err := f.setCurrentState(newState); err != nil {
					panic(err)
				}
			}
		} else {
			log.Printf("%s| Ignored Event: %+v\n", f.ID, eIn)
		}
		f.Unlock()
	}
	// TODO process uncaught events
	return nil
}
