package types

import (
	"errors"
	log "github.com/Sirupsen/logrus"
	"github.com/gocql/gocql"
	"github.com/relops/cqlr"
	"time"
)

const (
	QueueSync     = "sync"
	QueueAsync    = "async"
	QueueActive   = "Active"
	QueueDeleted  = "Deleted"
	QueueDeleting = "Deleting"
)

type Queue struct {
	UUID                   gocql.UUID  `cql:"queue_uuid" json:"uuid,required" description:"The unique identifier of the queue"`
	Name                   string      `cql:"name" json:"name,omitempty" description:"The unique name of the queue"`
	QueueType              string      `cql:"queue_type" json:"queueType,omitempty" description:"The type of queue: sync or async"`
	WindowOfOperation      string      `cql:"window_of_operation" json:"windowOfOperation,omitempty" description:"The window of operation for the queue if defined as sync"`
	ShouldDrain            bool        `cql:"should_drain" json:"shouldDrain,omitempty" description:"The expected behaviour of the queue when it is deleted. If true the queue will drain (and no longer accept new requests) before it is deleted.  Defaults to true"`
	BackPressureAction     *gocql.UUID `cql:"backpressure_action" json:"backpressureAction,omitempty" description:"The unique identifier of an action to be called in the event that the backpressure definition is breached"`
	BackpressureDefinition uint64      `cql:"backpressure_definition" json:"backpressureDefinition,omitempty" description:"For queues the backpressure definition defines the number of waiting task slots before the backpressure API endpoint is called."`
	OurTags                []string    `json:"tags,omitempty" description:"Tags assigned to the queue."`
	OurPaths               []string    `json:"paths,omitempty" description:"Paths assigned to the queue."`
	Tasks                  []Task      `json:"-"`
	Window                 Window      `json:"-"`
	Running                bool        `json:"-"`
	Status                 string      `json:"-"`
}

// Query
func GetQueues() []Queue {
	query := session.Query("select * from queues where status in (?, ?) allow filtering", QueueActive, QueueDeleting)
	bind := cqlr.BindQuery(query)
	var queue Queue
	queues := []Queue{}
	for bind.Scan(&queue) {
		queue.LoadTags()
		queue.LoadPaths()
		queues = append(queues, queue)
	}
	return queues
}

func GetQueuesByTag(tag string) []Queue {
	var id gocql.UUID
	var queue Queue
	queues := []Queue{}
	iteration := session.Query("select object_uuid from tags where type = 'queue' and tag = ? allow filtering", tag).Iter()
	for iteration.Scan(&id) {
		q := session.Query("select * from queues where queue_uuid = ?", id)
		b := cqlr.BindQuery(q)
		b.Scan(&queue)
		queue.LoadTags()
		queue.LoadPaths()
		queues = append(queues, queue)
	}
	return queues
}

func GetQueue(queueUUID string) (Queue, error) {
	query := session.Query("select * from queues where queue_uuid = ?", queueUUID)
	bind := cqlr.BindQuery(query)
	var queue Queue
	if !bind.Scan(&queue) {
		return Queue{}, errors.New("Unknown queue")
	}
	queue.LoadTags()
	queue.LoadPaths()
	return queue, nil
}

func GetQueueByPath(path string) (Queue, error) {
	var id gocql.UUID
	if err := session.Query(`select queue_uuid from paths where path = ? limit 1 allow filtering`, path).Scan(&id); err != nil {
		return Queue{}, errors.New("No queue found")
	}
	query := session.Query("select * from queues where queue_uuid = ?", id)
	bind := cqlr.BindQuery(query)
	var queue Queue
	bind.Scan(&queue)
	queue.LoadTags()
	queue.LoadPaths()
	return queue, nil
}

// CRUD
func (queue *Queue) CreateOrUpdate() error {
	// ensure paths are unique for object
	if queue.UUID.String() == "00000000-0000-0000-0000-000000000000" {
		// queue was generated from json with an unknown UUID.  Fix up
		queue.UUID = gocql.TimeUUID()
	}
	if queue.QueueType != "sync" && queue.QueueType != "async" {
		return errors.New("Invalid queue type")
	}
	_, parseErr := Parse(queue.WindowOfOperation)
	if parseErr != nil {
		return errors.New("Invalid window definition: " + parseErr.Error())
	}
	pathErr := queue.CreateOrUpdatePaths()
	if pathErr != nil {
		return pathErr
	}
	queue.CreateOrUpdateTags()
	queue.Status = QueueActive
	bind := cqlr.Bind(`insert into queues (queue_uuid, name, queue_type, window_of_operation, should_drain, backpressure_action, backpressure_definition, status) values (?, ?, ?, ?, ?, ?, ?, ?)`, queue)
	if err := bind.Exec(session); err != nil {
		return err
	} else {
		return nil
	}
}

func (queue Queue) Delete() error {
	queue.DeletePaths()
	queue.DeleteTags()
	if err := session.Query(`delete from queues where queue_uuid = ? and status = ?`, queue.UUID, QueueActive).Exec(); err != nil {
		return err
	}
	// add the queue back into the DB with a Deleted/Deleting status.
	if queue.ShouldDrain == true {
		queue.Status = QueueDeleting
	} else {
		queue.Status = QueueDeleted
	}
	bind := cqlr.Bind(`insert into queues (queue_uuid, name, queue_type, window_of_operation, should_drain, backpressure_action, backpressure_definition, status) values (?, ?, ?, ?, ?, ?, ?, ?)`, queue)
	if err := bind.Exec(session); err != nil {
		return err
	} else {
		return nil
	}
}

func (q *Queue) LoadWindow() error {
	window, parseErr := Parse(q.WindowOfOperation)
	if parseErr != nil {
		// Realistically we should never reach this point. CreateOrUpdate will ensure it is valid
		return parseErr
	}
	q.Window = window
	return nil
}

func (q *Queue) LoadTags() {
	q.OurTags = GetTagsForObject(q.UUID)
}

func (q Queue) CreateOrUpdateTags() {
	SetTagsForObject(q.UUID, q.OurTags, "queue")
}

func (q Queue) DeleteTags() {
	DeleteTagsForObject(q.UUID)
}

func (q *Queue) LoadPaths() {
	// find and return paths for queue
	q.OurPaths = LoadPathsFromDB(q.UUID)
}

func LoadPathsFromDB(uuid gocql.UUID) []string {
	// find and return paths for queue
	paths := []string{}
	path := ""
	iteration := session.Query("select path from paths where queue_uuid = ?", uuid).Iter()
	for iteration.Scan(&path) {
		paths = append(paths, path)
	}
	return paths
}

func (q Queue) CreateOrUpdatePaths() error {
	// set paths on queue
	pathsFromDB := LoadPathsFromDB(q.UUID)
	for _, path := range q.OurPaths {
		if path == "/" {
			return errors.New("Cannot define queue with root path")
		}
		if path[len(path)-1:] == "/" {
			return errors.New("Cannot define path with trailing slash")
		}
		if isStringInSlice(path, pathsFromDB) {
			pathsFromDB = findAndRemoveInSlice(path, pathsFromDB)
		} else {
			session.Query(`insert into paths (queue_uuid, path) VALUES (?, ?)`, q.UUID, path).Exec()
		}
	}
	for _, pathToDelete := range pathsFromDB {
		session.Query(`delete from paths where queue_uuid = ? and path = ?`, q.UUID, pathToDelete).Exec()
	}
	return nil
}

func (q Queue) DeletePaths() {
	// delete paths on queue
	session.Query(`delete from paths where queue_uuid = ?`, q.UUID).Exec()
}

// Queue Execution
func (q *Queue) IsRunning() bool {
	return q.Running
}

func (q Queue) MatchesPath(path string) bool {
	for _, p := range q.OurPaths {
		if p == path {
			return true
		}
	}
	return false
}

func (q Queue) CountOfTasks() uint64 {
	var count uint64
	query := "select count(*) from async_tasks where queue_uuid = ? and status = 'Pending' limit 1000000"
	if q.QueueType == QueueSync {
		query = "select count(*) from sync_tasks where queue_uuid = ? and status = 'Pending' limit 1000000"
	}
	if err := session.Query(query, q.UUID).Scan(&count); err == nil {
		return count
	}
	return 0
}

func (q Queue) ReceivedCompletionForTask(task_uuid string) {
	// in a sync model we only execute a promise (if defined) when a completion message is received.
	task, err := GetTask(task_uuid)
	if err == nil {
		task.ExecutePromise()
	}
}

func (q *Queue) StartOrContinueExecution(starting bool) {
	if starting {
		log.WithFields(log.Fields{"name": q.Name, "UUID": q.UUID.String()}).Info("Starting execution on Queue")
	} else {
		log.WithFields(log.Fields{"name": q.Name, "UUID": q.UUID.String()}).Info("Continuing execution on Queue")
	}
	q.Running = true

	//select * from tasks where queue_uuid = 11111111-1111-1111-1111-111111111111 and when > dateof(now()) and when < '2015-03-15 17:00';

	// select count(*) from sync_tasks where queue_uuid = cfd66ccc-d857-4e90-b1e5-df98a3d40cd6 and status = 'Pending' limit 1000000 ;
	if q.QueueType == QueueSync {
		// sync mode
		// execute each task in order.  wait for completion and then execute the next
		for {
			var id gocql.UUID
			if err := session.Query(`select task_uuid from sync_tasks where queue_uuid = ? and status = ? limit 1;`, q.UUID, TaskPending).Scan(&id); err == nil {
				// found a valid task in the queue.  execute and return.  we'll rely on the queue manager to start us up again when the completion message is received
				task, err := GetTask(id.String())
				if err == nil {
					// found a matching task. now execute and immediately return
					if !task.Execute(true) {
						// the action failed.  which means we wont ever receive a completion message
						task.ExecutePromise()
						continue
					}
					return
				}
			}
			// we didnt find a task for this queue in scope.  so wait, and try again.
			time.Sleep(15 * time.Second)
		}
	} else if q.QueueType == QueueAsync {
		// async mode
		// execute each task independently based on timestamp

	}

	/*
		TODO - plan for queue execution
		Step 1: Update status flag on queue to "RUNNING"
		when this function starts we need to determine behaviour.
		if async (which means we execute when the timestamp fires)
			read tasks from DB for the next 10 minutes
			register each with the queue monitor for updates
			results will be ordered by cassandra so we can simply execute each in order
			for each task fire a timer to execute when "when" occurs
			when ten minutes is up request more, add to the queue monitor and execute

		When we receive a NEW message from the queue monitor determine what to do:
		async - ignore if when is not within the current timeslice (10min slot)
		sync - ignore, we are only executing one item at a time

		When we receive an UPDATE message from the queue monitor:
		async - if the item is in this timeslice, update local store (and reset timers perhaps). if its being executed.. too late ignore
		sync - ignore because we're already executing it

	*/
}

func (q *Queue) StopExecution(reason string) {
	log.WithFields(log.Fields{"name": q.Name, "reason": reason}).Info("Stopping execution on Queue")
	q.Running = false
}
