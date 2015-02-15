package eunomia

import (
	log "github.com/Sirupsen/logrus"
	"github.com/kieranbroadfoot/horae/types"
)

func updateWorker(nodeId int, workerCh chan types.EunomiaRequest) {
	log.WithFields(log.Fields{"worker": nodeId}).Debug("Starting eunomia update worker")
	client := getEtcdClient()
	for {
		select {
		case request := <-workerCh:
			// pull out the key and value portions of the request and post to etcd
			if request.Action == types.EunomiaActionUpdate {
				_, err := client.Set(request.Key, request.Value, request.TTL)
				if err != nil {
					log.WithFields(log.Fields{"key": request.Key, "error": err}).Warn("Unable to update key")
				} else {
					log.WithFields(log.Fields{"key": request.Key, "worker": nodeId}).Info("Updated key")
				}
			} else if request.Action == types.EunomiaActionDelete {
				_, err := client.Delete(request.Key, false)
				if err != nil {
					log.WithFields(log.Fields{"key": request.Key, "error": err}).Warn("Unable to delete key")
				} else {
					log.WithFields(log.Fields{"key": request.Key, "worker": nodeId}).Info("Deleted key")
				}
			}
		}
	}
}
