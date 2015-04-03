package dike

import (
	log "github.com/Sirupsen/logrus"
	"github.com/kieranbroadfoot/horae/types"
	"path"
)

var knownQueues []*types.Queue

func StartDike(node types.Node, failure chan bool, toEunomia chan types.EunomiaRequest) {
	log.Print("Starting Dike")
	// TODO - Dike needs to track new queues from etcd so that new managers can be fired up when signalled by the API
	for _, queue := range types.GetQueues() {
		savedQ := queue
		knownQueues = append(knownQueues, &savedQ)
		go queueManager(&savedQ, toEunomia)
	}
}

func shouldRun(thepath string) bool {
	foundMatch := false
	for _, q := range knownQueues {
		if q.MatchesPath(thepath) {
			foundMatch = true
			if thepath == "/" {
				return true
			} else {
				if q.IsRunning() {
					return shouldRun(path.Dir(thepath))
				}
			}
		}
	}
	if foundMatch == false {
		// there was no queue that exists at this path. that's ok. keep going
		return shouldRun(path.Dir(thepath))
	}
	return false
}
