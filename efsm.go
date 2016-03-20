package fsm

import (
	"sync"
	"fmt"
	"time"
	"log"
)

type Transition struct {
	state  State
	timeout *Timeout
	events map[Event]func(State, Event) State
}

type Event string

type State string

type Timeout struct {
	Name string
	duration time.Duration
}

// state x event => action => [state',...]
// given a state,  and an event, determine an action/transition to one of several possible states


type FSM struct {
	sync.Mutex
	CurrentState State
	States       map[State]State
	Events       chan Event
	transitions  map[State]*Transition
}

func (f *FSM) setCurrentState(s State) error {
	if _, ok := f.States[s]; ok {
		f.CurrentState = s
		if t, ok := f.transitions[s]; ok {
			if t.timeout != nil {
				timer := time.NewTimer(t.timeout.duration)
				go func() {
					<- timer.C
					log.Printf("Timeout: %s",t.timeout.Name)
					f.Events <- Event(t.timeout.Name)
				}()
			}
		}
	} else {
		return fmt.Errorf("Invalid State")
	}
	return nil
}

func (f *FSM) When(s State) *Transition {
	t := Transition{
		state: s,
		timeout: nil,
		events: make(map[Event]func(State, Event) State),
	}
	f.transitions[s] = &t
	return &t
}

func (t *Transition) Timeout(timer *Timeout) *Transition {
	t.timeout = timer
	return t
}

func (t *Transition) Case(e Event, fn func(s State, e Event) State) error {
	t.events[e] = fn
	return nil
}
func (f *FSM) Engine(initial State, c chan Event) error {
	f.setCurrentState(initial)
	f.Events = c
	f.transitions = make(map[State]*Transition)
	return nil
}

func (f *FSM) Run() error {
	log.Printf("In Run")
	for e := range f.Events {
		if transitions, ok := f.transitions[f.CurrentState]; ok {
			if fn, ok := transitions.events[e]; ok {
				newState := fn(f.CurrentState, e)
				if newState != "" {
					log.Printf("--> %s\n", newState)
					if err := f.setCurrentState(newState); err != nil {
						panic(err)
					}
				}
			}
		}
		// TODO process uncaught events
	}
	return nil
}