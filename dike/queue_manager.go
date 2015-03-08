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

	// Load window of operation
	err := queue.LoadWindow()
	if err != nil {
		log.WithFields(log.Fields{"queue": queue.UUID}).Info("Queue failed to start (invalid window definition)")
	}

	state := "pre"
	timer := time.NewTimer(queueTime(queue, "pre"))

	for {
		select {
		case <-timer.C:
			switch state {
			case "pre":
				// claim master
				channelToMonitor <- types.EunomiaQueueRequest{Action: types.EunomiaRequestBecomeMaster, QueueUUID: queue.UUID}
				timer = time.NewTimer(queueTime(queue, "start"))
				state = "start"
			case "start":
				// start executing the queue - if we are not master we dont do anything
				if queueMaster {
					queue.StartExecution()
				}
				// in five seconds let's generate the stop timer.  see bug defined in window.go
				state = "genEnd"
				timer = time.NewTimer(5 * time.Second)
			case "genEnd":
				state = "end"
				timer = time.NewTimer(queueTime(queue, "stop"))
			case "end":
				// release queue via eunomia
				channelToMonitor <- types.EunomiaQueueRequest{Action: types.EunomiaRequestReleaseMaster, QueueUUID: queue.UUID}

				// TODO - IF ShouldDrain is TRUE and no tasks can be found we should close and request final deletion!!!!
			}
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
	now := time.Now()
	start := queue.Window.GetNextStartTime()
	if action == "pre" {
		if now.Unix() == start.Unix() {
			// queue should already be open.  return immediately and claim the queue
			return 0 * time.Second
		} else {
			// return 30 seconds before window opens so can try and claim the queue
			return (start.Sub(now) - 20*time.Second)
		}
	} else if action == "start" {
		if now.Unix() == start.Unix() {
			// even though the queue should be running we need the preTimer to fire to claim the queue
			return 20 * time.Second
		} else {
			return start.Sub(now)
		}
	} else {
		return queue.Window.GetNextEndTime().Sub(now)
	}
}
