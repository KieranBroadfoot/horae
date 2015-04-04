package eunomia

import (
	log "github.com/Sirupsen/logrus"
	"github.com/kieranbroadfoot/horae/types"
	"math/rand"
	"time"
)

func electMaster(node types.Node, toEirene chan types.EireneStrategyAction) {
	// Function creates a node
	log.Print("Starting Master Election")

	// create new etcd client connection
	client := getEtcdClient()

	// setup random TTL for the node's TTL
	// minus 2 seconds from this value to give us the update rate
	rand.Seed(time.Now().Unix())
	nodeTTL := rand.Intn(20-10) + 10
	updateRate := nodeTTL - 2

	// regularly update the node in the server list
	updateNodeCh := make(chan bool) // signal shutdown
	go updateNode(updateNodeCh, getClusterPath()+"/nodes", node, nodeTTL, updateRate)

	// now wait a couple of seconds before we start the election check
	time.Sleep(2 * time.Second)

	for {
		// every 30 seconds check the status of the cluster and
		// determine which node is currently master.  if its ourselves
		// then configure ourselves as master.  If not change state
		// to slave
		isMaster, newMasterAddr, newMasterPort := findMaster(client, getClusterPath()+"/nodes", node)
		if isMaster {
			toEirene <- types.EireneStrategyAction{"master", "", ""}
		} else {
			toEirene <- types.EireneStrategyAction{"slave", "http://" + newMasterAddr, newMasterPort}
		}
		time.Sleep(30 * time.Second)
	}
}
