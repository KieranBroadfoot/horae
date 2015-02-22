package dike

import (
	log "github.com/Sirupsen/logrus"
	"github.com/kieranbroadfoot/horae/types"
)

func StartDike(node types.Node, failure chan bool, toEunomia chan types.EunomiaRequest) {
	log.Print("Starting Dike")
	for _, queue := range types.GetQueues() {
		go queueManager(queue, toEunomia)
	}
}
