package fsm

import (
	"fmt"
	"time"
)

func ExampleFSM() {

	f := &FSM{}

	f.States = map[State]State{
		"Running":State("Running"),
		"Rebooting":State("Rebooting"),
		"Failed":State("Failed"),
	}

	c := make(chan Event, 1)

	// initial state is Running
	f.Engine("Running", c)

	var err error

	// In state Running, when event reboot is received, go to Rebooting
	err = f.When("Running").Case("reboot", func(s State, e Event)State {
		fmt.Printf("Case %s %s\n", s, e)
		return "Rebooting"
	})
	if err  != nil {
		return
	}
	//Creating a timer to avoid staying in rebooting for ever
	timer := Timeout{
		Name: "boot_timeout",
		duration: time.Duration(5 * time.Second),
	}
	// In state Rebooting, add a timeout of 10s
	transition := f.When("Rebooting").Timeout(&timer)

	// If event booted is received go back to Running
	err = transition.Case("booted", func(s State, e Event)State {
		fmt.Printf("Case %s %s\n", s, e)
		return "Running"
	})

	if err  != nil {
		return
	}

	// if timeout is received - moved to failed.
	err = transition.Case("boot_timeout", func(s State, e Event)State {
		fmt.Printf("Case %s %s\n", s, e)
		return "Failed"
	})

	if err  != nil {
		return
	}

	// In state Failed, when event reset is received, go to Rebooting
	err = f.When("Failed").Case("reset", func(s State, e Event)State {
		fmt.Printf("Case %s %s\n", s, e)
		return "Rebooting"
	})
	if err  != nil {
		return
	}

	// Run the FSM engine
	go f.Run()

	// Send a reboot event and after 10s send a reset
	go func() {
		c <- "reboot"
		time.Sleep(10 * time.Second)
		c <- "reset"
	}()

	time.Sleep(30 * time.Second)
	fmt.Printf("Done")

	//Output:
	//Case Running reboot
	//Case Rebooting boot_timeout
	//Case Failed reset
	//Case Rebooting boot_timeout
	//Done
}
