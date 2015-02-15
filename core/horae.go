package core

import (
	log "github.com/Sirupsen/logrus"
	"github.com/kieranbroadfoot/horae/common"
	"github.com/kieranbroadfoot/horae/dike"
	"github.com/kieranbroadfoot/horae/eirene"
	"github.com/kieranbroadfoot/horae/eunomia"
	"github.com/kieranbroadfoot/horae/types"
	"os"
)

func StartServer(clusterName string) {
	log.WithFields(log.Fields{"cluster": clusterName}).Info("Starting horae server")

	// Create core core node type
	node := types.Node{common.GenerateUUID(), clusterName, "", ""}

	types.InitDAO(clusterName)

	/*	uuid_for_queue := gocql.TimeUUID().String()

		queue := types.Queue{
			UUID:              uuid_for_queue,
			Name:              "my_test_queue",
			QueueType:         "async",
			WindowOfOperation: "",
			ShouldDrain:       false}

		log.Print("initial create")
		types.CreateOrUpdateQueue(queue)

		types.SetTagsForQueue(queue, []string{"tag_fuck", "tag_you"})
		types.SetPathsForQueue(queue, []string{"/foo/bar", "/hello/world"})

		for _, tag := range types.GetTagsForQueue(queue) {
			log.Print(tag)
		}
		for _, path := range types.GetPathsForQueue(queue) {
			log.Print(path)
		}

		log.Print("check object")
		thing := types.GetQueue(uuid_for_queue)
		log.Print(thing)

		log.Print("update object")
		queue.QueueType = "sync"
		types.CreateOrUpdateQueue(queue)

		log.Print("check again")
		thing = types.GetQueue(uuid_for_queue)
		log.Print(thing)

		types.DeleteQueue(queue)

		log.Print("check again (third)")
		thing = types.GetQueue(uuid_for_queue)
		log.Print(thing)

		things := types.GetQueues()
		log.Print(things)

		things = types.GetQueuesByTag("tag1")
		log.Print(things)

		thing1, _ := types.GetQueueByPath("/datacenter/dc2/west/rack1")
		log.Print(thing1)

		thing2, _ := types.GetQueueByPath("/fuck/you")
		log.Print(thing2)

		log.Fatal("Exit")*/

	// Signal failure to core core
	coreFailureCh := make(chan bool)
	// signal to this core an enriched Node object containing our core dataset
	eireneToCore := make(chan types.Node)
	// signal master/slave status
	eunomiaToEireneCh := make(chan types.EireneStrategyAction)
	// signal action requests to Eunomia
	allToEunomiaCh := make(chan types.EunomiaRequest)

	// Start etcd Manager
	go eirene.StartEirene(node, eireneToCore, coreFailureCh, allToEunomiaCh, eunomiaToEireneCh)

	for {
		select {
		case <-coreFailureCh:
			log.Print("Received error state. Shutting down")
			os.Exit(1)
		case node = <-eireneToCore:
			log.WithFields(log.Fields{"UUID": node.UUID, "cluster": node.Cluster, "IP": node.Address, "port": node.Port}).Info("Node generated")
			// Start API Server
			go eunomia.StartEunomia(node, coreFailureCh, eunomiaToEireneCh, allToEunomiaCh)
			// Start Queue Manager
			go dike.StartDike(node, coreFailureCh, allToEunomiaCh)
		}
	}
}
