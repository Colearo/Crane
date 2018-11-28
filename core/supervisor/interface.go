package main

import (
	"crane/core/boltworker"
	"crane/core/messages"
	"crane/core/spoutworker"
	"crane/core/utils"
	"fmt"
	"log"
)

// Supervisor, the slave node for accepting the schedule from the master node
// and execute the task, spouts or bolts
type Supervisor struct {
	Sub          *messages.Subscriber
	BoltWorkers  []*boltworker.BoltWorker
	SpoutWorkers []*spoutworker.SpoutWorker
}

// Factory mode to return the Supervisor instance
func NewSupervisor(driverAddr string) *Supervisor {
	supervisor := &Supervisor{}
	supervisor.Sub = messages.NewSubscriber(driverAddr)
	if supervisor.Sub == nil {
		return nil
	}
	return supervisor
}

// Daemon function for supervisor service
func (s *Supervisor) StartDaemon() {
	s.BoltWorkers = make([]*boltworker.BoltWorker, 0)
	s.SpoutWorkers = make([]*spoutworker.SpoutWorker, 0)

	go s.Sub.RequestMessage()
	go s.Sub.ReadMessage()
	s.SendJoinRequest()

	for rcvMsg := range s.Sub.PublishBoard {
		fmt.Println("yes")
		log.Printf("Receive Message from %s: %s\n", rcvMsg.SourceConnId, rcvMsg.Payload)
		payload := utils.CheckType(rcvMsg.Payload)

		switch payload.Header.Type {
		case utils.BOLT_TASK:
			fmt.Println("1")
			task := &utils.BoltTaskMessage{}
			utils.Unmarshal(payload.Content, task)
			bw := boltworker.NewBoltWorker(10, task.PluginFile, task.Name, task.Port, task.PrevBoltAddr,
				task.PrevBoltGroupingHint, task.PrevBoltFieldIndex,
				task.SuccBoltGroupingHint, task.SuccBoltFieldIndex)
			s.BoltWorkers = append(s.BoltWorkers, bw)
			log.Printf("Receive Bolt Dispatch %s Previous workers %v\n", task.Name, task.PrevBoltAddr)

		case utils.SPOUT_TASK:
			fmt.Println("2")
			task := &utils.SpoutTaskMessage{}
			utils.Unmarshal(payload.Content, task)
			sw := spoutworker.NewSpoutWorker(task.PluginFile, task.Name, task.Port, task.GroupingHint, task.FieldIndex)
			s.SpoutWorkers = append(s.SpoutWorkers, sw)
			log.Printf("Receive Spout Dispatch %s \n", task.Name)

		case utils.TASK_ALL_DISPATCHED:
			fmt.Println("3")
			for _, sw := range s.SpoutWorkers {
				go sw.Start()
			}
			for _, bw := range s.BoltWorkers {
				go bw.Start()
			}

		}
	}
}

// Send join request to join the cluster
func (s *Supervisor) SendJoinRequest() {
	join := utils.JoinRequest{Name: "vm [" + s.Sub.Conn.LocalAddr().String() + "]"}
	b, err := utils.Marshal(utils.JOIN_REQUEST, join)
	if err != nil {
		log.Println(err)
		return
	}
	s.Sub.Request <- messages.Message{
		Payload:      b,
		TargetConnId: s.Sub.Conn.RemoteAddr().String(),
	}
}

func main() {
	supervisor := NewSupervisor(":" + fmt.Sprintf("%d", utils.DRIVER_PORT))
	if supervisor == nil {
		log.Println("Initialize supervisor failed")
		return
	}
	supervisor.StartDaemon()
}
