package eunomia

import (
	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-etcd/etcd"
	"github.com/gocql/gocql"
	"github.com/kieranbroadfoot/horae/types"
	"path"
)

func monitorQueues(request types.EunomiaRequest) {
	log.Info("Master queue monitor started")
	// this master monitor will watch for new or deleted queues to inform the core dike function of the need
	// to stop/start queue managers

	client := getEtcdClient()

	// channels for managing long-running etcd watchers
	etcdWatchQueueActivity := make(chan *etcd.Response)
	etcdWatchQueueStop := make(chan bool)

	go client.Watch(getClusterPath()+"/updates/queues/", 0, true, etcdWatchQueueActivity, etcdWatchQueueStop)

	for {
		select {
		case queueUpdate := <-etcdWatchQueueActivity:
		// seen an update to the queue in etcd.  Signal to queue manager
			if queueUpdate == nil {
				// long poll has expired. restart
				etcdWatchQueueStop <- true
				go client.Watch(getClusterPath()+"/updates/queues/", 0, true, etcdWatchQueueActivity, etcdWatchQueueStop)
			} else {
				if queueUpdate.Action != "expire" {
					log.WithFields(log.Fields{"action": queueUpdate.Node.Value, "type": types.EunomiaQueue, "UUID": path.Base(queueUpdate.Node.Key)}).Info("Updating Dike")
					uuid, _ := gocql.ParseUUID(path.Base(queueUpdate.Node.Key))
					request.ChannelToQueueManager <- types.EunomiaResponse{Action: queueUpdate.Node.Value, Type: types.EunomiaQueue, UUID: uuid}
				}
			}
		}
	}
}
