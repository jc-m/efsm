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

	err = slave.When("Running").Case([]Event{"stop"}, func(f *FSM, s *State, e Event) *State {
		fmt.Printf("Case %s %s\n", s.Name, e)

		return f.Goto("Stopped")
	})
	if err  != nil {
		return
	}
	err = slave.When("Stopped").Case([]Event{"start"}, func(f *FSM, s *State, e Event) *State {
		fmt.Printf("Case %s %s\n", s.Name, e)

		return f.Goto("Running")
	})
	if err  != nil {
		return
	}

	err = master.When("Running").Case([]Event{"Stopped"}, func(f *FSM, s *State, e Event) *State {
		fmt.Printf("Case %s %s\n", s.Name, e)

		return f.Goto("Failed")
	})
	if err  != nil {
		return
	}

	err = master.When("Failed").Case([]Event{"Running"}, func(f *FSM, s *State, e Event) *State {
		fmt.Printf("Case %s %s\n", s.Name, e)

		return f.Goto("Running")
	})
	if err  != nil {
		return
	}

	go master.Run("Running")
	go slave.Run("Running")


	go func() {
		time.Sleep(10 * time.Second)
		slaveIn <- "stop"
		time.Sleep(10 * time.Second)
		slaveIn <- "start"
	}()
	time.Sleep(30 * time.Second)
	fmt.Printf("Done")


	//Output:
	//Case Running stop
	//Case Running Stopped
	//Case Stopped start
	//Case Failed Running
	//Done
}