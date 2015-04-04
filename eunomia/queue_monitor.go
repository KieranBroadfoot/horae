package eunomia

import (
	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-etcd/etcd"
	"github.com/gocql/gocql"
	"github.com/kieranbroadfoot/horae/types"
	"math/rand"
	"path"
	"time"
)

func monitorQueue(node types.Node, request types.EunomiaRequest, requestsFromAll chan types.EunomiaRequest) {
	log.WithFields(log.Fields{"queue": request.QueueUUID}).Info("Queue monitor started")
	// this queue monitor will receive requests from the Dike Queue Manager and attempt
	// to claim/monitor ownership of the queue.  It will also monitor for changes on the
	// queue in etcd to ensure the queue manager is kept in real-time sync with changes from the API

	client := getEtcdClient()
	masterTimer := time.NewTimer(1 * time.Hour)
	updateNodeCh := make(chan bool)

	// channels for managing long-running etcd watchers
	etcdWatchQueueActivity := make(chan *etcd.Response)
	etcdWatchQueueStop := make(chan bool)
	etcdWatchTaskActivity := make(chan *etcd.Response)
	etcdWatchTaskStop := make(chan bool)

	// start watching for changes relating to this queue
	startWatchers(client, request.QueueUUID, etcdWatchQueueActivity, etcdWatchQueueStop, etcdWatchTaskActivity, etcdWatchTaskStop)

	for {
		select {
		case queueManagerRequest := <-request.ChannelFromQueueManager:
			if queueManagerRequest.Action == types.EunomiaRequestBecomeMaster {
				log.WithFields(log.Fields{"queue": request.QueueUUID}).Info("Attempting to claim ownership of queue")
				rand.Seed(time.Now().Unix())
				nodeTTL := rand.Intn(20-10) + 10
				updateRate := nodeTTL - 2
				// regularly update /queues/<Queue UUID>
				go updateNode(updateNodeCh, getClusterPath()+"/queues/"+queueManagerRequest.QueueUUID.String(), node, nodeTTL, updateRate)
				// in two seconds check for master state
				masterTimer = time.NewTimer(2 * time.Second)
			} else if queueManagerRequest.Action == types.EunomiaRequestReleaseMaster {
				log.WithFields(log.Fields{"queue": request.QueueUUID}).Info("Relinquishing ownership of queue")
				requestsFromAll <- types.EunomiaRequest{Action: types.EunomiaStoreDelete, Key: getClusterPath() + "/queues/" + queueManagerRequest.QueueUUID.String() + "/" + node.UUID.String()}
				updateNodeCh <- false // kill the updater
				masterTimer.Stop()
			}
		case <-masterTimer.C:
			masterTimer = time.NewTimer(30 * time.Second)
			isMaster, _, _ := findMaster(client, getClusterPath()+"/queues/"+request.QueueUUID.String(), node)
			if isMaster {
				request.ChannelToQueueManager <- types.EunomiaResponse{Action: types.EunomiaResponseBecameQueueMaster}
			} else {
				request.ChannelToQueueManager <- types.EunomiaResponse{Action: types.EunomiaResponseBecameQueueSlave}
			}
		case queueUpdate := <-etcdWatchQueueActivity:
			// seen an update to the queue in etcd.  Signal to queue manager
			if queueUpdate == nil {
				// long poll has expired. restart
				restartWatchers(client, request.QueueUUID, etcdWatchQueueActivity, etcdWatchQueueStop, etcdWatchTaskActivity, etcdWatchTaskStop)
			} else {
				updateQueueManager(request, queueUpdate, types.EunomiaQueue)
			}
		case taskUpdate := <-etcdWatchTaskActivity:
			if taskUpdate == nil {
				restartWatchers(client, request.QueueUUID, etcdWatchQueueActivity, etcdWatchQueueStop, etcdWatchTaskActivity, etcdWatchTaskStop)
			} else {
				updateQueueManager(request, taskUpdate, types.EunomiaTask)
			}
		}
	}
}

func updateQueueManager(request types.EunomiaRequest, update *etcd.Response, actionType string) {
	if update.Action != "expire" {
		log.WithFields(log.Fields{"action": update.Node.Value, "type": actionType, "UUID": path.Base(update.Node.Key)}).Info("Updating Queue Manager")
		uuid, _ := gocql.ParseUUID(path.Base(update.Node.Key))
		request.ChannelToQueueManager <- types.EunomiaResponse{Action: update.Node.Value, Type: actionType, UUID: uuid}
	}
}

func restartWatchers(client *etcd.Client, queue gocql.UUID, queueWatch chan *etcd.Response, queueStop chan bool, taskWatch chan *etcd.Response, taskStop chan bool) {
	stopWatchers(queueStop, taskStop)
	startWatchers(client, queue, queueWatch, queueStop, taskWatch, taskStop)
}

func startWatchers(client *etcd.Client, queue gocql.UUID, queueWatch chan *etcd.Response, queueStop chan bool, taskWatch chan *etcd.Response, taskStop chan bool) {
	log.Print("MONITORING: "+getClusterPath()+"/updates/queues/"+queue.String())
	go client.Watch(getClusterPath()+"/updates/queues/"+queue.String(), 0, false, queueWatch, queueStop)
	go client.Watch(getClusterPath()+"/updates/tasks/", 0, true, taskWatch, taskStop)
}

func stopWatchers(queueStop chan bool, taskWatch chan bool) {
	// shut down long running watchers
	queueStop <- true
	taskWatch <- true
}
