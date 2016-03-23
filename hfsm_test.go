package efsm

import (
	"fmt"
	"time"
)

func ExampleHFSM() {
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
		Scope: slave.ID,
	}
	err = slave.When("Running").Case([]Event{stop}, func(f *FSM, s *State, e Event) *State {
		fmt.Printf("Case %s %s\n", s.Name, e)

		return f.Goto("Stopped")
	})
	if err != nil {
		return
	}
	start := Event{
		Name: "start",
		Scope: slave.ID,
	}
	err = slave.When("Stopped").Case([]Event{start}, func(f *FSM, s *State, e Event) *State {
		fmt.Printf("Case %s %s\n", s.Name, e)

		return f.Goto("Running")
	})
	if err != nil {
		return
	}

	stopped := Event{
		Name: "Stopped",
		Scope: slave.ID,
	}
	err = master.When("Running").Case([]Event{stopped}, func(f *FSM, s *State, e Event) *State {
		fmt.Printf("Case %s %s\n", s.Name, e)

		return f.Goto("Failed")
	})
	if err != nil {
		return
	}
	running := Event{
		Name: "Running",
		Scope: slave.ID,
	}
	err = master.When("Failed").Case([]Event{running}, func(f *FSM, s *State, e Event) *State {
		fmt.Printf("Case %s %s\n", s.Name, e)

		return f.Goto("Running")
	})
	if err != nil {
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

func createNode(name string, c chan Event) *FSM {
	var err error

	f := &FSM{ID:name}

	in := make(chan Event, 10)

	f.Engine(in, c)

	stop := Event{
		Name: "stop",
		Scope: f.ID,
	}

	err = f.When("Running").Case([]Event{stop}, func(f *FSM, s *State, e Event) *State {
		fmt.Printf("%s:Case %s %s\n", f.ID, s.Name, e)

		return f.Goto("Stopped")
	})
	if err != nil {
		return nil
	}
	start := Event{
		Name: "start",
		Scope: f.ID,
	}

	err = f.When("Stopped").Case([]Event{start}, func(f *FSM, s *State, e Event) *State {
		fmt.Printf("%s:Case %s %s\n", f.ID, s.Name, e)

		return f.Goto("Running")
	})
	if err != nil {
		return nil
	}

	go f.Run("Running")

	return f

}

func ExampleCluster() {
	var (
		err error
		nbnodes=5
	)

	cluster := &FSM{ID:"cluster"}

	clusterIn := make(chan Event, nbnodes*2)
	clusterOut := make(chan Event, nbnodes*2)

	cluster.Engine(clusterIn, clusterOut)

	stopped := Event{
		Name: "Stopped",
		Scope: "",
	}

	running := Event{
		Name: "Running",
		Scope: "",
	}

	h := cluster.When("Healthy")

	err = h.Case([]Event{stopped}, func(f *FSM, s *State, e Event) *State {
		healthyCount :=0

		if s.Data != nil {
			healthyCount = *s.Data.(*int)
		}
		healthyCount -= 1

		if healthyCount < nbnodes {
			return f.Goto("Degraded").Using(&healthyCount)
		} else {
			return f.Goto("Healthy").Using(&healthyCount)
		}
	})
	if err != nil {
		return
	}

	err = h.Case([]Event{running}, func(f *FSM, s *State, e Event) *State {
		healthyCount :=0

		if s.Data != nil {
			healthyCount = *s.Data.(*int)
		}
		healthyCount += 1

		if healthyCount < nbnodes {
			return f.Goto("Degraded").Using(&healthyCount)
		} else {
			return f.Goto("Healthy").Using(&healthyCount)
		}
	})
	if err != nil {
		return
	}



	d := cluster.When("Degraded")

	err = d.Case([]Event{running}, func(f *FSM, s *State, e Event) *State {
		healthyCount :=0

		if s.Data != nil {

			healthyCount = *s.Data.(*int)
		}
		healthyCount += 1

		if healthyCount < nbnodes {
			return f.Goto("Degraded").Using(&healthyCount)
		} else {
			return f.Goto("Healthy").Using(&healthyCount)
		}
	})

	if err != nil {
		return
	}

	err = d.Case([]Event{stopped}, func(f *FSM, s *State, e Event) *State {
		healthyCount :=0

		if s.Data != nil {
			healthyCount = *s.Data.(*int)
		}
		healthyCount -= 1

		return f.Goto("Degraded").Using(&healthyCount)
	})

	go cluster.Run("Degraded")

	nodes := make(map[string]*FSM)

	for i := 0; i < nbnodes; i++ {
		name := fmt.Sprintf("node-%d", i)
		nodes[name] = createNode(name, clusterIn)
	}

	go func() {
		for e := range clusterOut {
			fmt.Printf("Cluster: %s\n",e.Name)
		}
	}()

	go func() {
		time.Sleep(10 * time.Second)
		// stopping node 3
		n := nodes["node-3"]
		n.In <- Event{
			Name:"stop",
			Scope:n.ID,
		}
	}()
	time.Sleep(20 * time.Second)
	fmt.Printf("Done")

	//Output:
	//Cluster: Degraded
	//Cluster: Degraded
	//Cluster: Degraded
	//Cluster: Degraded
	//Cluster: Degraded
	//Cluster: Healthy
	//node-3:Case Running {stop node-3}
	//Cluster: Degraded
	//Done
}