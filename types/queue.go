package types

import (
	"errors"
	log "github.com/Sirupsen/logrus"
	"github.com/gocql/gocql"
	"github.com/relops/cqlr"
)

const (
	QueueSync  = "sync"
	QueueAsync = "async"
)

type Queue struct {
	UUID                   gocql.UUID  `cql:"queue_uuid" json:"uuid,required" description:"The unique identifier of the queue"`
	Name                   string      `cql:"name" json:"name,omitempty" description:"The unique name of the queue"`
	QueueType              string      `cql:"queue_type" json:"queueType,omitempty" description:"The type of queue: sync or async"`
	WindowOfOperation      string      `cql:"window_of_operation" json:"windowOfOperation,omitempty" description:"The window of operation for the queue if defined as sync"`
	ShouldDrain            bool        `cql:"should_drain" json:"shouldDrain,omitempty" description:"The expected behaviour of the queue when it is deleted. If true the queue will drain (and no longer accept new requests) before it is deleted.  Defaults to true"`
	BackPressureAction     *gocql.UUID `cql:"backpressure_action" json:"backpressureAction,omitempty" description:"The unique identifier of an action to be called in the event that the backpressure definition is breached"`
	BackpressureDefinition string      `cql:"backpressure_definition" json:"backpressureDefinition,omitempty" description:"For synchronous queues the backpressure definition defines the number of waiting task slots before the backpressure API endpoint is called."`
	OurTags                []string    `json:"tags,omitempty" description:"Tags assigned to the queue."`
	OurPaths               []string    `json:"paths,omitempty" description:"Paths assigned to the queue."`
	Tasks                  []Task      `json:"-"`
	Window                 Window      `json:"-"`
	Running                bool        `json:"-"`
}

// Query
func GetQueues() []Queue {
	query := session.Query("select * from queues")
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
	query := session.Query("SELECT * FROM queues where queue_uuid = ?", id)
	bind := cqlr.BindQuery(query)
	var queue Queue
	bind.Scan(&queue)
	queue.LoadTags()
	queue.LoadPaths()
	return queue, nil
}

// CRUD
func (queue *Queue) CreateOrUpdate() error {
	// TODO - paths must not end in /
	// TODO - paths should be unique????
	// TODO - no-one can define a path of '/'
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
	queue.CreateOrUpdateTags()
	queue.CreateOrUpdatePaths()
	bind := cqlr.Bind(`insert into queues (queue_uuid, name, queue_type, window_of_operation, should_drain, backpressure_action, backpressure_definition) values (?, ?, ?, ?, ?, ?, ?)`, queue)
	if err := bind.Exec(session); err != nil {
		return err
	} else {
		return nil
	}
}

func (queue Queue) Delete() error {
	queue.DeletePaths()
	queue.DeleteTags()
	bind := cqlr.Bind(`delete from queues where queue_uuid = ?`, queue)
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

func (q Queue) CreateOrUpdatePaths() {
	// set paths on queue
	pathsFromDB := LoadPathsFromDB(q.UUID)
	for _, path := range q.OurPaths {
		if isStringInSlice(path, pathsFromDB) {
			pathsFromDB = findAndRemoveInSlice(path, pathsFromDB)
		} else {
			session.Query(`insert into paths (queue_uuid, path) VALUES (?, ?)`, q.UUID, path).Exec()
		}
	}
	for _, pathToDelete := range pathsFromDB {
		session.Query(`delete from paths where queue_uuid = ? and path = ?`, q.UUID, pathToDelete).Exec()
	}
}

func (q Queue) DeletePaths() {
	// delete paths on queue
	session.Query(`delete from paths where queue_uuid = ?`, q.UUID).Exec()
}

// Queue Execution
func (q *Queue) LoadTasks() {
	log.WithFields(log.Fields{"name": q.Name}).Info("Loading tasks on Queue")
	// TODO - should order tasks by execution time and priority
	// TODO - kill this function.  no longer necessary to load. Start Execution will load as it goes.
	if q.Name == "root" {
		// root never receives tasks
		q.Tasks = []Task{}
	} else {
		uuid := gocql.UUID{}
		iteration := session.Query("select task_uuid from tasks where queue_uuid = ?", q.UUID).Iter()
		for iteration.Scan(&uuid) {
			task, err := GetTaskWithQueue(q.UUID, uuid)
			if err == nil {
				q.Tasks = append(q.Tasks, task)
			}
		}
	}
}



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

func (q *Queue) StartExecution() {
	log.WithFields(log.Fields{"name": q.Name}).Info("Starting execution on Queue")
	q.Running = true

	//select * from tasks where queue_uuid = 11111111-1111-1111-1111-111111111111 and when > dateof(now()) and when < '2015-03-15 17:00';

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

		if sync (which means we execute in order and wait for completion)
			read first task
			register interest with queue monitor
			execute
			wait for update from queue monitor for completion
			repeat

		When we receive a NEW message from the queue monitor determine what to do:
		async - ignore if when is not within the current timeslice (10min slot)
		sync - ignore, we are only executing one item at a time

		When we receive an UPDATE message from the queue monitor:
		async - if the item is in this timeslice, update local store (and reset timers perhaps). if its being executed.. too late ignore
		sync - ignore because we're already executing it

	*/

	//if q.QueueType == QueueSync {
	// execute task at top of the list
	// wait for completion - seen via etcd update
	// then start the next

	//} else {
	// async queue.  find first task
	for _, task := range q.Tasks {
		task.Execute(false)
	}

	// if queue is async

	// every minute do the following: for all tasks executing in the next minute start timers to execute
}

func (q *Queue) StopExecution(reason string) {
	log.WithFields(log.Fields{"name": q.Name, "reason": reason}).Info("Stopping execution on Queue")
	q.Running = false
}
