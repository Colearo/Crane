package main

import (
	"crane/core/boltworker"
	"crane/core/messages"
	"crane/core/spoutworker"
	"crane/core/utils"
	"flag"
	"fmt"
	"log"
	"os/exec"
	"os/user"
	// "time"
)

// Supervisor, the slave node for accepting the schedule from the master node
// and execute the task, spouts or bolts
type Supervisor struct {
	Sub          *messages.Subscriber
	BoltWorkers  []*boltworker.BoltWorker
	SpoutWorkers []*spoutworker.SpoutWorker
	VmIndexMap   map[int]string
	FilePathMap  map[string]string
}

// Factory mode to return the Supervisor instance
func NewSupervisor(driverAddr string) *Supervisor {
	supervisor := &Supervisor{}
	supervisor.Sub = messages.NewSubscriber(driverAddr)
	if supervisor.Sub == nil {
		return nil
	}
	supervisor.BoltWorkers = make([]*boltworker.BoltWorker, 0)
	supervisor.SpoutWorkers = make([]*spoutworker.SpoutWorker, 0)
	supervisor.VmIndexMap = make(map[int]string)
	supervisor.FilePathMap = make(map[string]string)
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
			payload := utils.CheckType(rcvMsg.Payload)

			switch payload.Header.Type {
			case utils.FILE_PULL:
				filePull := &utils.FilePull{}
				utils.Unmarshal(payload.Content, filePull)
				if filePull.Filename != "None" {
					s.GetFile(filePull.Filename)
				}

			case utils.BOLT_TASK:
				task := &utils.BoltTaskMessage{}
				utils.Unmarshal(payload.Content, task)
				bw := boltworker.NewBoltWorker(10, "./"+task.PluginFile, task.Name, task.Port, task.PrevBoltAddr,
					task.PrevBoltGroupingHint, task.PrevBoltFieldIndex,
					task.SuccBoltGroupingHint, task.SuccBoltFieldIndex)
				s.BoltWorkers = append(s.BoltWorkers, bw)
				log.Printf("Receive Bolt Dispatch %s with Port %s, Previous workers %v\n", task.Name, task.Port, task.PrevBoltAddr)

			case utils.SPOUT_TASK:
				task := &utils.SpoutTaskMessage{}
				utils.Unmarshal(payload.Content, task)
				sw := spoutworker.NewSpoutWorker("./"+task.PluginFile, task.Name, task.Port, task.GroupingHint, task.FieldIndex)
				s.SpoutWorkers = append(s.SpoutWorkers, sw)
				log.Printf("Receive Spout Dispatch %s with Port %s\n", task.Name, task.Port)

			case utils.TASK_ALL_DISPATCHED:
				fmt.Printf("Finished receive Bolt and Spout Dispatchs\n")
				for _, sw := range s.SpoutWorkers {
					go sw.Start()
				}
				for _, bw := range s.BoltWorkers {
					go bw.Start()
				}
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

// Get the plugin file from distributed file system
func (s *Supervisor) GetFile(remoteName string) {
	_, ok := s.FilePathMap[remoteName]
	if ok {
		return
	}
	// Execute the sdfs client to get the remote file
	usr, _ := user.Current()
	usrHome := usr.HomeDir
	cmd := exec.Command(usrHome+"/go/src/crane/tools/sdfs_client/sdfs_client", "-master", "fa18-cs425-g29-01.cs.illinois.edu:5000", "get", remoteName, "./"+remoteName)
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", stdoutStderr)
	s.FilePathMap[remoteName] = "./" + remoteName
}

func main() {
	driverIpPtr := flag.String("h", "127.0.0.1", "Driver's IP address")
	vmIndexPtr := flag.Int("vm", 0, "VM index in cluster")
	flag.Parse()
	vms := utils.GetVmMap()
	var ip string
	if *vmIndexPtr <= 0 && *driverIpPtr == "127.0.0.1" {
		ip = *driverIpPtr
		log.Println("Enter Local Mode")
	} else if *vmIndexPtr != 0 {
		if *vmIndexPtr > 10 {
			log.Fatal("VM Cluster Index out of range")
			return
		} else {
			ip = vms[*vmIndexPtr]
			log.Println("Enter Cluster mode")
		}
	} else {
		ip = *driverIpPtr
		log.Println("Enter Remote Mode")
	}

	LocalIP := utils.GetLocalIP().String()
	LocalHostname := utils.GetLocalHostname()
	log.Printf("Local Machine Info [%s] [%s]\n", LocalIP, LocalHostname)

	supervisor := NewSupervisor(ip + ":" + fmt.Sprintf("%d", utils.DRIVER_PORT))
	if supervisor == nil {
		log.Println("Initialize supervisor failed")
		return
	}
	supervisor.VmIndexMap = vms
	supervisor.StartDaemon()
}
