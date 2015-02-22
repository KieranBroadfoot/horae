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

var (
	clusterName      string
	cassandraAddress string
	etcdAddress      string
)

func StartServer() {
	InitConfig()
	log.WithFields(log.Fields{"cluster": clusterName}).Info("Starting horae server")

	// Create core node type
	node := types.Node{common.GenerateUUID(), clusterName, "", ""}

	eunomia.InitETCD(etcdAddress)
	types.InitDAO(cassandraAddress, clusterName)

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
