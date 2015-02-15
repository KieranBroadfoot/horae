package eunomia

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/kieranbroadfoot/horae/types"
	"time"
)

func updateNode(killChannel chan bool, path string, node types.Node, nodeTTL int, updateRate int) {
	// Function creates (and regularly updates) a node entry under /core/clusters/<cluster_name>/
	client := getEtcdClient()
	nodeJson, err := json.Marshal(node)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Warn("Unable to generate node as json")
	} else {
		log.Debug("Creating node entry")
		log.WithFields(log.Fields{"key": path + "/" + node.UUID.String()}).Info("Created node key")
		_, err := client.Create(path+"/"+node.UUID.String(), string(nodeJson), uint64(nodeTTL))
		if err != nil {
			log.WithFields(log.Fields{"error": err}).Warn("Unable to create node")
		}
		for {
			ticker := time.NewTicker(time.Duration(updateRate) * time.Second)
			select {
			case <-killChannel:
				return
			case <-ticker.C:
				log.WithFields(log.Fields{"key": path + "/" + node.UUID.String(), "ttl": fmt.Sprintf("%v", updateRate)}).Debug("Updating node key")
				_, err := client.Update(path+"/"+node.UUID.String(), string(nodeJson), uint64(nodeTTL))
				if err != nil {
					log.WithFields(log.Fields{"error": err}).Warn("Unable to update node")
				}
			}
		}
	}
}
