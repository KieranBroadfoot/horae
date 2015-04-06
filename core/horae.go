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

var isRunning bool

func StartServer() {
	types.InitConfig()
	log.WithFields(log.Fields{"cluster": types.Configuration.ClusterName}).Info("Starting horae server")

	// Create core node type
	node := types.Node{UUID: common.GenerateUUID(), Cluster: types.Configuration.ClusterName}

	eunomia.InitETCD(types.Configuration.ETCDAddress)
	types.InitDAO(types.Configuration.CassandraAddress, types.Configuration.ClusterName)

	// Signal failure to core core
	coreFailureCh := make(chan bool)
	// signal to this core an enriched Node object containing our core dataset
	eireneToCore := make(chan types.Node)
	// signal master/slave status
	eunomiaToEireneCh := make(chan types.EireneStrategyAction)
	// signal action requests to Eunomia
	allToEunomiaCh := make(chan types.EunomiaRequest)

	// Start etcd Manager
	go eirene.StartEirene(node, types.Configuration.StaticPort, eireneToCore, coreFailureCh, allToEunomiaCh, eunomiaToEireneCh)

	for {
		select {
		case <-coreFailureCh:
			log.Print("Received error state. Shutting down")
			os.Exit(1)
		case node = <-eireneToCore:
			if !isRunning {
				log.WithFields(log.Fields{"UUID": node.UUID, "cluster": node.Cluster, "IP": node.Address, "port": node.Port}).Info("Node generated")
				// Start API Server
				go eunomia.StartEunomia(node, coreFailureCh, eunomiaToEireneCh, allToEunomiaCh)
				// Start Queue Manager
				go dike.StartDike(node, coreFailureCh, allToEunomiaCh)
				isRunning = true
			} else {
				// received an updated Node object from eirene, store and update configuration
				types.Configuration.MasterURI = node.MasterURI
			}
		}
	}
}
