package eunomia

import (
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-etcd/etcd"
	"github.com/kieranbroadfoot/horae/types"
)

func findMaster(client *etcd.Client, path string, node types.Node) (bool, string, string) {
	resp, err := client.Get(path, false, true)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Warn("Failed to query cluster status")
		return false, "", ""
	} else {
		var lowestIndex uint64 = 0
		master := ""
		for _, node := range resp.Node.Nodes {
			if lowestIndex == 0 {
				master = node.Value
				lowestIndex = node.CreatedIndex
			} else if node.CreatedIndex < lowestIndex {
				// found a new master
				master = node.Value
				lowestIndex = node.CreatedIndex
			}
		}
		var newMaster types.Node
		err := json.Unmarshal([]byte(master), &newMaster)

		if err != nil {
			log.Warn("Cannot unmarshall new master object")
		} else {
			if newMaster.UUID == node.UUID {
				// we are the master
				return true, "", ""
			} else {
				return false, newMaster.Address, newMaster.Port
			}
		}
	}
	// should never reach this point
	return false, "", ""
}
