package eunomia

import (
	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-etcd/etcd"
	"github.com/kieranbroadfoot/horae/types"
)

const (
	rootPath = "/horae/clusters/"
)

var etcdAddress string
var clusterPath string

/*

Structure of keys in etcd for core

Root: /core/clusters/<clustername> (where clustername is passed on command line or is set as "default"

Every update below should be set with a TTL, either for election purposes or simply to ensure updates are
quickly captured and updated

/nodes/<UUID> - value { addr: , port: } - use indexes to elect/monitor leader
/queues/<Queue UUID>/<Node UUID> - value { addr:, port: } - use indexes to elect/monitor queue leader for each queue
/updates/queues/<Queue UUID> - value Action (Update/Create/Delete) - used to indicate changes to queues from API, update is read from DB
/updates/tasks/<Queue UUID>/<Task UUID> - value Action (Update/Create/Delete) - used to indicate changes to tasks from API, update is read from DB

*/

func InitETCD(address string) {
	etcdAddress = address
	// check etcd is up
	client := getEtcdClient()
	_, err := client.Get("/", false, false)
	if err != nil {
		log.WithFields(log.Fields{"reason": err}).Fatal("Unable to init ETCD connection")
	}
}

func getEtcdClient() *etcd.Client {
	return etcd.NewClient([]string{"http://" + etcdAddress})
}

func getClusterPath() string {
	return clusterPath
}

func setupEtcd(node types.Node) {
	clusterPath = rootPath+node.Cluster
	for _, value := range [4]string{"/nodes", "/queues", "/updates/queues", "/updates/tasks"} {
		client := getEtcdClient()
		// check for root dir for this cluster
		_, err := client.Get(getClusterPath()+value, false, false)
		if err != nil {
			// Creating the default root path for this cluster; accepting that this may fail as another node is creating simultaneously
			log.WithFields(log.Fields{"path": getClusterPath() + value}).Info("Creating node for cluster")
			client.CreateDir(getClusterPath()+value, 0)
		}
	}
}

func StartEunomia(node types.Node, failure chan bool, toEirene chan types.EireneStrategyAction, requestsFromAll chan types.EunomiaRequest) {
	log.Print("Starting Eunomia")
	setupEtcd(node)
	go electMaster(node, toEirene)

	workerCh := make(chan types.EunomiaRequest)
	for i := 0; i <= 9; i++ {
		go updateWorker(i, workerCh)
	}

	for {
		select {
		case request := <-requestsFromAll:
			if request.Action == types.EunomiaQueuesMonitor {
				// case: receive message from Dike to set up a monitor for newly created/deleted queues
				go monitorQueues(request)
			} else if request.Action == types.EunomiaQueueMonitor {
				// case: receive message from a queue manager to set up a queue monitor
				go monitorQueue(node, request, requestsFromAll)
			} else if request.Action == types.EunomiaStoreUpdate || request.Action == types.EunomiaStoreDelete {
				// Do nothing more than pass it on to one of our workers
				// TODO - is this a bottleneck?  or can we ensure other actions in this case are quick to exec?
				workerCh <- request
			}
		}
	}
}
