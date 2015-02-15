package dike

import (
	log "github.com/Sirupsen/logrus"
	"github.com/kieranbroadfoot/horae/types"
	"time"
)

func queueManager(queue types.Queue, toEunomia chan types.EunomiaRequest) {
	log.WithFields(log.Fields{"queue": queue.UUID}).Info("Queue manager started")

	channelToMonitor := make(chan types.EunomiaQueueRequest)
	channelFromMonitor := make(chan types.EunomiaQueueResponse)

	// start a queue monitor in eunomia.
	toEunomia <- types.EunomiaRequest{Action: types.EunomiaActionMonitor, ChannelFromQueueManager: channelToMonitor, ChannelToQueueManager: channelFromMonitor, QueueUUID: queue.UUID}

	queueMaster := false

	preTimer := time.NewTimer(queueTime(queue, "pre"))
	startTimer := time.NewTimer(queueTime(queue, "start"))
	endTimer := time.NewTimer(queueTime(queue, "stop"))

	for {
		select {
		case <-preTimer.C:
			// claim master
			channelToMonitor <- types.EunomiaQueueRequest{Action: types.EunomiaRequestBecomeMaster, QueueUUID: queue.UUID}
		case <-startTimer.C:
			// start executing the queue - if we are not master we dont do anything
			if queueMaster {
				queue.StartExecution()
			}
		case <-endTimer.C:
			// release queue via eunomia
			channelToMonitor <- types.EunomiaQueueRequest{Action: types.EunomiaRequestReleaseMaster, QueueUUID: queue.UUID}

			// stop execution of the queue
			// if queue is draining and task list now empty we signal deletion of the queue
			// TODO - think about re-balancing of queue.  should we relinquish control when the window closes?
			// TODO - IF ShouldDrain is TRUE and no tasks can be found we should close and request final deletion!!!!
		case queueResponse := <-channelFromMonitor:
			if queueResponse.Action == types.EunomiaResponseBecameQueueMaster {
				if queueMaster != true {
					log.WithFields(log.Fields{"queue": queue.UUID, "status": "master"}).Info("Changing queue status")
					queueMaster = true
					queue.LoadTasks()
				}
			} else if queueResponse.Action == types.EunomiaResponseBecameQueueSlave {
				if queueMaster != false {
					log.WithFields(log.Fields{"queue": queue.UUID, "status": "slave"}).Info("Changing queue status")
					queueMaster = false
					queue.StopExecution()
				}
			} else if queueResponse.Action == types.EunomiaResponseActionCreate {
				// TODO - implement. type is queue or task, uuid contains the object which has updated
			} else if queueResponse.Action == types.EunomiaResponseActionUpdate {
				// TODO - implement
			} else if queueResponse.Action == types.EunomiaResponseActionDelete {
				// TODO - implement
			}
		}
	}
}

func queueTime(queue types.Queue, action string) (duration time.Duration) {
	//log.Print("Request for duration: " + action)
	// returns duration until next activity for the queue
	// if the Window is nil then pre = now, start = 5 seconds, end = largest int64 value (290 years)
	// this means a never-ending queue is pinned to a node until restart of the node
	if action == "pre" {
		return 0 * time.Second
	} else if action == "start" {
		return 5 * time.Second
	} else {
		// end duration
		return 10000 * time.Second
	}
}
