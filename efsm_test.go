package efsm

import (
	"fmt"
	"time"
)

func ExampleFSM() {

	f := &FSM{ID:"Example"}

	in := make(chan Event, 1)
	out := make(chan Event, 1)

	f.Engine(in, out)

	var err error
	reboot := Event{
		Name: "reboot",
		Scope: f.ID,
	}

	reset := Event{
		Name: "reset",
		Scope: f.ID,
	}
	// In state Running, when event reboot or reset is received, go to Rebooting with a 5s timeout
	err = f.When("Running").Case([]Event{reboot, reset}, func(f *FSM, s *State, e Event) *State {
		fmt.Printf("Case %s %s\n", s.Name, e)
		x := 0

		if s.Data != nil {

			x = *s.Data.(*int)
		}
		fmt.Printf("Rebooting %d time\n", x)
		x += 1
		return f.Goto("Rebooting").ForMax(5 * time.Second).Using(&x)
	})
	if err  != nil {
		return
	}

	s := f.When("Rebooting")

	booted := Event{
		Name: "booted",
		Scope: f.ID,
	}

	// If event booted is received go back to Running
	err = s.Case([]Event{booted}, func(f *FSM, s *State, e Event) *State {
		fmt.Printf("Case %s %s\n", s.Name, e)

		return f.Goto("Running").Using(s.Data)
	})

	if err  != nil {
		return
	}

	rebootedTimeout := Event{
		Name: "Rebooting-timeout",
		Scope: f.ID,
	}
	// if timeout is received - moved to failed.
	err = s.Case([]Event{rebootedTimeout}, func(f *FSM, s *State, e Event) *State {
		fmt.Printf("Case %s %s\n", s.Name, e)

		return f.Goto("Failed").Using(s.Data)
	})

	if err  != nil {
		return
	}

	// In state Failed, when event reset is received, go to Rebooting with a 5s timeout
	err = f.When("Failed").Case([]Event{reset}, func(f *FSM, s *State, e Event) *State {
		fmt.Printf("Case %s %s\n", s.Name, e)

		x := 0
		if s.Data != nil {
			x = *s.Data.(*int)
		}
		fmt.Printf("Rebooting %d time\n", x)
		x += 1

		return f.Goto("Rebooting").ForMax(5 * time.Second).Using(&x)
	})
	if err  != nil {
		return
	}

	// Run the FSM engine
	go f.Run("Running")

	// Send a reboot event and after 10s send a reset
	go func() {
		in <- reboot
		time.Sleep(10 * time.Second)
		in <- reset
	}()



	time.Sleep(30 * time.Second)
	fmt.Printf("Done")

	//Output:
	//Case Running reboot
	//Rebooting 0 time
	//Case Rebooting Rebooting-timeout
	//Case Failed reset
	//Rebooting 1 time
	//Case Rebooting Rebooting-timeout
	//Done
}
