package dike

import (
	log "github.com/Sirupsen/logrus"
	"github.com/kieranbroadfoot/horae/types"
)

func StartDike(node types.Node, failure chan bool, toEunomia chan types.EunomiaRequest) {
	log.Print("Starting Dike")

	// TODO - determine who should create the root queue.  Should this be the master?

	for _, queue := range types.GetQueues() {
		go queueManager(queue, toEunomia)
	}
}
