package dike

import (
	log "github.com/Sirupsen/logrus"
	"github.com/kieranbroadfoot/horae/types"
	"path"
)

var knownQueues []*types.Queue

func StartDike(node types.Node, failure chan bool, toEunomia chan types.EunomiaRequest) {
	log.Print("Starting Dike")

	for _, queue := range types.GetQueues() {
		savedQ := queue
		knownQueues = append(knownQueues, &savedQ)
		go queueManager(&savedQ, toEunomia)
	}

	// monitor for updates (create/delete) in etcd
	channelFromMonitor := make(chan types.EunomiaResponse)
	toEunomia <- types.EunomiaRequest{Action: types.EunomiaQueuesMonitor, ChannelToQueueManager: channelFromMonitor}

	for {
		select {
		case queueResponse := <-channelFromMonitor:
			if queueResponse.Action == types.EunomiaActionCreate {
				queue, err := types.GetQueue(queueResponse.UUID.String())
				if err == nil {
					knownQueues = append(knownQueues, &queue)
					go queueManager(&queue, toEunomia)
				}
			} else if queueResponse.Action == types.EunomiaActionDelete {
				for idx, b := range knownQueues {
					if b.UUID == queueResponse.UUID {
						knownQueues = append(knownQueues[:idx], knownQueues[idx+1:]...)
					}
				}
			}
		}
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
