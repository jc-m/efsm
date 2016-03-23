package efsm

import (
	"fmt"
	"time"
)

func ExampleHFSM(){
	var err error

	master := &FSM{ID:"master"}

	masterIn := make(chan Event, 1)
	masterOut := make(chan Event, 1)

	master.Engine(masterIn, masterOut)

	slave := &FSM{ID:"slave"}

	slaveIn := make(chan Event, 1)
	slave.Engine(slaveIn, masterIn)

	stop := Event{
		Name: "stop",
		Source: slave.ID,
	}
	err = slave.When("Running").Case([]Event{stop}, func(f *FSM, s *State, e Event) *State {
		fmt.Printf("Case %s %s\n", s.Name, e)

		return f.Goto("Stopped")
	})
	if err  != nil {
		return
	}
	start := Event{
		Name: "start",
		Source: slave.ID,
	}
	err = slave.When("Stopped").Case([]Event{start}, func(f *FSM, s *State, e Event) *State {
		fmt.Printf("Case %s %s\n", s.Name, e)

		return f.Goto("Running")
	})
	if err  != nil {
		return
	}

	stopped := Event{
		Name: "Stopped",
		Source: slave.ID,
	}
	err = master.When("Running").Case([]Event{stopped}, func(f *FSM, s *State, e Event) *State {
		fmt.Printf("Case %s %s\n", s.Name, e)

		return f.Goto("Failed")
	})
	if err  != nil {
		return
	}
	running := Event{
		Name: "Running",
		Source: slave.ID,
	}
	err = master.When("Failed").Case([]Event{running}, func(f *FSM, s *State, e Event) *State {
		fmt.Printf("Case %s %s\n", s.Name, e)

		return f.Goto("Running")
	})
	if err  != nil {
		return
	}

	go master.Run("Running")
	go slave.Run("Running")


	go func() {
		time.Sleep(2 * time.Second)
		slaveIn <- stop
		time.Sleep(2 * time.Second)
		slaveIn <- start
	}()
	time.Sleep(10 * time.Second)
	fmt.Printf("Done")


	//Output:
	//Case Running {stop slave}
	//Case Running {Stopped slave}
	//Case Stopped {start slave}
	//Case Failed {Running slave}
	//Done
}