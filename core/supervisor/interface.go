package main

import (
	"crane/core/messages"
	"crane/core/utils"
	"fmt"
	"log"
)

// Supervisor, the slave node for accepting the schedule from the master node
// and execute the task, spouts or bolts
type Supervisor struct {
	Sub *messages.Subscriber
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
	go s.Sub.RequestMessage()
	go s.Sub.ReadMessage()
	s.SendJoinRequest()
	for {
		select {
		case rcvMsg := <-s.Sub.PublishBoard:
			log.Printf("Receive Message from %s: %s\n", rcvMsg.SourceConnId, rcvMsg.Payload)
			payload := utils.CheckType(rcvMsg.Payload)
			switch payload.Header.Type {
			case utils.BOLT_TASK:
				task := &utils.BoltTaskMessage{}
				utils.Unmarshal(payload.Content, task)
				log.Printf("Receive Bolt Dispatch %s Previous workers\n", task.Name, task.PrevBoltAddr)
			case utils.SPOUT_TASK:
				task := &utils.SpoutTaskMessage{}
				utils.Unmarshal(payload.Content, task)
				log.Printf("Receive Bolt Dispatch %s \n", task.Name)
			}
		default:

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
