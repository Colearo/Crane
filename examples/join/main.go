package main 

import (
	"crane/bolt"
	"crane/core/utils"
	"crane/spout"
	"crane/topology"
)

func main() {
	// Create a topology
	tm := topology.Topology{}

	// Create a spout
	sp := spout.NewSpoutInst("GenderSpout", "process.so", "GenderSpout", utils.GROUPING_BY_FIELD, 0)
	sp.SetInstanceNum(1)
	tm.AddSpout(sp)

	// Create a bolt
	// Params: name, pluginFile, pluginSymbol, groupingHint, fieldIndex
	bm := bolt.NewBoltInst("GenderAgeJoinBolt", "process.so", "GenderAgeJoinBolt", utils.GROUPING_BY_ALL, 0)
	bm.SetInstanceNum(2)
	bm.AddPrevTaskName("GenderSpout")
	tm.AddBolt(bm)

	tm.SubmitFile("./process.so", "process.so")
	tm.Submit(":5050")
}