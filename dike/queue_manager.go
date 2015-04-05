package dike

import (
	log "github.com/Sirupsen/logrus"
	"github.com/kieranbroadfoot/horae/types"
	"time"
	"path"
)

const (
	StartingExecution    = true
	ContinuingExecution   = false
)

func queueManager(queue *types.Queue, toEunomia chan types.EunomiaRequest) {
	log.WithFields(log.Fields{"queue": queue.UUID}).Info("Queue manager started")

	channelToMonitor := make(chan types.EunomiaQueueRequest)
	channelFromMonitor := make(chan types.EunomiaResponse)

	// start a queue monitor in eunomia.
	toEunomia <- types.EunomiaRequest{Action: types.EunomiaQueueMonitor, ChannelFromQueueManager: channelToMonitor, ChannelToQueueManager: channelFromMonitor, QueueUUID: queue.UUID}

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
				resetStart := false
				if queueMaster {
					willRun := false
					if len(queue.OurPaths) == 0 {
						// if the queue has no defined paths then check against the root queue only
						// this is equivalent to always returning true but we may allow changes to the root queue in the future
						willRun = shouldRun("/")
					} else {
						for _, p := range queue.OurPaths {
							// each path of the queue.
							willRun = shouldRun(path.Dir(p))
						}
					}
					if willRun {
						go queue.StartOrContinueExecution(StartingExecution)
					} else {
						// what to do if the queue is not currently contained but should be running
						resetStart = true
					}
				}
				if resetStart {
					// let's try and restart the queue in 60 seconds. maybe one of the containing queues will
					// start operations.  We'll give up if we are about to hit the end time for this window
					endTime := queueTime(queue, "stop")
					if endTime < 60 * time.Second {
						// about to reach the end of our window.  jump to end state
						state = "end"
						timer = time.NewTimer(endTime)
					} else {
						timer = time.NewTimer(60 * time.Second)
					}
				} else {
					state = "running"
					timer = time.NewTimer(60 * time.Second)
				}
			case "running":
				continueRunning := true
				for _, p := range queue.OurPaths {
					// check if queue is *still* contained
					continueRunning = shouldRun(path.Dir(p))
				}
				if !continueRunning {
					queue.StopExecution("Lost Containment")
					// if we are no longer contained.  set state back to start.  we may be available again before we reach our end state
					state = "start"
					timer = time.NewTimer(60 * time.Second)
				} else {
					// continue running and check again for containment in 60 seconds
					endTime := queueTime(queue, "stop")
					if endTime < 60 * time.Second {
						state = "end"
						timer = time.NewTimer(queueTime(queue, "stop"))
					} else {
						state = "running"
						timer = time.NewTimer(60 * time.Second)
					}
				}
			case "end":
				// release queue via eunomia
				queue.StopExecution("Window Closed")
				channelToMonitor <- types.EunomiaQueueRequest{Action: types.EunomiaRequestReleaseMaster, QueueUUID: queue.UUID}
				state = "pre"
				timer = time.NewTimer(queueTime(queue, "pre"))
				// TODO - IF ShouldDrain is TRUE and no tasks can be found we should close and request final deletion!!!!
			}
		case queueResponse := <-channelFromMonitor:
			if queueResponse.Action == types.EunomiaResponseBecameQueueMaster {
				if queueMaster != true {
					log.WithFields(log.Fields{"queue": queue.UUID, "status": "master"}).Info("Changing queue status")
					queueMaster = true
					state = "start"
					timer = time.NewTimer(queueTime(queue, "start"))
				}
			} else if queueResponse.Action == types.EunomiaResponseBecameQueueSlave {
				if queueMaster != false {
					log.WithFields(log.Fields{"queue": queue.UUID, "status": "slave"}).Info("Changing queue status")
					queueMaster = false
					queue.StopExecution("Lost Ownership")
				}
			} else if queueResponse.Action == types.EunomiaActionCreate {
				if queueResponse.Type == types.EunomiaTask {
					// TODO - implement
				}
			} else if queueResponse.Action == types.EunomiaActionUpdate {
				if queueResponse.Type == types.EunomiaQueue {
					// reload queue from DB
					q, err := types.GetQueue(queueResponse.UUID.String())
					if err == nil {
						queue = &q
					}
					// stop execution (we don't know precisely what changed so the best bet is to reset)
					queue.StopExecution("Queue Updated")
					// reset timer to pre state
					state = "pre"
					timer = time.NewTimer(queueTime(queue, "pre"))
				} else if queueResponse.Type == types.EunomiaTask {
					// TODO - implement
				}
			} else if queueResponse.Action == types.EunomiaActionDelete {
				if queueResponse.Type == types.EunomiaQueue {
					log.WithFields(log.Fields{"queue": queueResponse.UUID.String()}).Info("Queue manager shutting down")
					return
				} else if queueResponse.Type == types.EunomiaTask {
					// TODO - implement
				}
			} else if queueResponse.Action == types.EunomiaActionComplete {
				if queueResponse.Type == types.EunomiaTask && queue.QueueType == types.QueueSync {
					// if we've received a completion message and the queue is a sync type.  two things need to
					// happen.  execute any associated promises for the task.
					// kick the execution off again to get the next task started
					queue.ReceivedCompletionForTask(queueResponse.UUID.String())
					if queue.IsRunning() {
						// only continue if the queue is still open for business
						go queue.StartOrContinueExecution(ContinuingExecution)
					}
				}
			}
		}
	}
}

func queueTime(queue *types.Queue, action string) (duration time.Duration) {
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
