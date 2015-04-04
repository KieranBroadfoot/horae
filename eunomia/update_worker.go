package eunomia

import (
	log "github.com/Sirupsen/logrus"
	"github.com/kieranbroadfoot/horae/types"
	"strings"
)

func updateWorker(nodeId int, workerCh chan types.EunomiaRequest) {
	log.WithFields(log.Fields{"worker": nodeId}).Debug("Starting eunomia update worker")
	client := getEtcdClient()
	for {
		select {
		case request := <-workerCh:
			// pull out the key and value portions of the request and post to etcd
			key := request.Key
			if !strings.HasPrefix(key, "/") {
				key = getClusterPath()+"/"+key
			}
			if request.Action == types.EunomiaStoreUpdate {
				_, err := client.Set(key, request.Value, request.TTL)
				if err != nil {
					log.WithFields(log.Fields{"key": key, "value": request.Value, "error": err}).Warn("Unable to update key")
				} else {
					log.WithFields(log.Fields{"key": key, "value": request.Value, "worker": nodeId}).Info("Updated key")
				}
			} else if request.Action == types.EunomiaStoreDelete {
				_, err := client.Delete(key, false)
				if err != nil {
					log.WithFields(log.Fields{"key": key, "value": request.Value, "error": err}).Warn("Unable to delete key")
				} else {
					log.WithFields(log.Fields{"key": key, "value": request.Value, "worker": nodeId}).Info("Deleted key")
				}
			}
		}
	}
}
