package types

import (
	"github.com/gocql/gocql"
)

const (
	EunomiaQueue                     = "queue"
	EunomiaTask                      = "task"
	EunomiaStoreUpdate               = "store_update"
	EunomiaStoreDelete               = "store_delete"
	EunomiaQueuesMonitor             = "action_queues_monitor" // monitor all queues for changes
	EunomiaQueueMonitor              = "action_queue_monitor"  // monitor a specific queue for changes
	EunomiaRequestBecomeMaster       = "state_master"
	EunomiaRequestReleaseMaster      = "state_release"
	EunomiaResponseBecameQueueMaster = "became_queue_master"
	EunomiaResponseBecameQueueSlave  = "became_queue_slave"
	EunomiaActionCreate              = "eunomia_action_create"
	EunomiaActionUpdate              = "eunomia_action_update"
	EunomiaActionDelete              = "eunomia_action_delete"
	EunomiaActionComplete            = "eunomia_action_complete"
)

type EunomiaRequest struct {
	// General purpose request object to signal activity within eunomia
	// either updates to a key in etcd or to request the setup of a new queue monitor
	Action                  string // update / delete / monitor
	Key                     string // key/value updates via workers in Eunomia
	Value                   string
	TTL                     uint64
	QueueUUID               gocql.UUID // additional fields required for queue monitor setup
	ChannelFromQueueManager chan EunomiaQueueRequest
	ChannelToQueueManager   chan EunomiaResponse
}

type EunomiaQueueRequest struct {
	// used to signal a desire to become or release master ownership of a queue
	Action    string
	QueueUUID gocql.UUID
}

type EunomiaResponse struct {
	// signalling from eunomia to dike regarding changes seen in etcd
	// changes include master/slave updates, queue/task updates via the API
	Type   string     // queue/task
	Action string     // create/update/delete
	UUID   gocql.UUID // UUID of changing object
}
