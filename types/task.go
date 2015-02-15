package types

import (
	log "github.com/Sirupsen/logrus"
	"github.com/gocql/gocql"
	"github.com/relops/cqlr"
	"time"
)

const (
	TaskGet             = "GET"
	TaskPut             = "PUT"
	TaskPost            = "POST"
	TaskHead            = "HEAD"
	TaskDelete          = "DELETE"
	TaskPending         = "Pending"
	TaskRunning         = "Running"
	TaskComplete        = "Complete"
	TaskFailed          = "Failure"
	TaskPartiallyFailed = "Partially Failed"
)

type Task struct {
	UUID            gocql.UUID `cql:"task_uuid" json:"uuid,required" description:"The unique identifier of the task"`
	Name            string     `cql:"name" json:"name,omitempty" description:"The name of the task"`
	Priority        uint64     `cql:"priority" json:"uuid,omitempty" description:"The priority of the task. If the queue is sync ordered by priority otherwise ordered by exec time and then priority"`
	Queue           gocql.UUID `cql:"queue_uuid" json:"queue,omitempty" description:"The UUID of the hosting queue"`
	When            time.Time  `cql:"when" json:"when,omitempty" description:"The future execution timestamp of the task"`
	PromiseAction   gocql.UUID `cql:"promise_action" json:"promise,omitempty" description:"The unique identifier of the promise, executed on successful completion of the execution action"`
	ExecutionAction gocql.UUID `cql:"execution_action" json:"execution,required" description:"The unique identifier of the executing action"`
	Status          string     `cql:"status" json:"status,required" description:"The status of the task (Pending/Running/Complete/Failed/Partially Failed)"`
	Promise         Action     `json:"-"`
	Execution       Action     `json:"-"`
}

func GetTasksByTag(tag string) []Task {
	var id gocql.UUID
	var task Task
	tasks := []Task{}
	iteration := session.Query("select object_uuid from tags where type = 'task' and tag = ? allow filtering", tag).Iter()
	for iteration.Scan(&id) {
		q := session.Query("select * from task where task_uuid = ?", id)
		b := cqlr.BindQuery(q)
		b.Scan(&task)
		tasks = append(tasks, task)
	}
	return tasks
}

func GetTaskWithQueue(queueUUID gocql.UUID, taskUUID gocql.UUID) Task {
	query := session.Query("select * from tasks where queue_uuid = ? and task_uuid = ?", queueUUID, taskUUID)
	return bindActionsToTask(query)
}

func GetTask(taskUUID gocql.UUID) Task {
	query := session.Query("select * from tasks where task_uuid = ? allow filtering", taskUUID)
	return bindActionsToTask(query)
}

func bindActionsToTask(query *gocql.Query) Task {
	bind := cqlr.BindQuery(query)
	var task Task
	bind.Scan(&task)
	// at this point we have the Task object. Now find associated actions
	if task.PromiseAction.String() != "" {
		task.Promise = GetAction(task.PromiseAction)
	}
	if task.ExecutionAction.String() != "" {
		task.Execution = GetAction(task.ExecutionAction)
	}
	return task
}

func (task Task) CreateOrUpdate() error {
	// TODO - if Queue is null, set to root queue and check when is set
	// TODO - update actions also.
	bind := cqlr.Bind(`insert into tasks (queue_uuid, task_uuid, execution_action, name, priority, promise_action, status, when) values (?, ?, ?, ?, ?, ?, ?, ?)`, task)
	if err := bind.Exec(session); err != nil {
		return err
	} else {
		return nil
	}
}

func (task Task) Delete() error {
	bind := cqlr.Bind(`delete from tasks where queue_uuid = ? and task_uuid = ?`, task)
	if err := bind.Exec(session); err != nil {
		return err
	} else {
		return nil
	}
}

func (t Task) Tags() []string {
	return GetTagsForObject(t.UUID)
}

func (t Task) SetTags(tags []string) {
	SetTagsForObject(t.UUID, tags, "task")
}

func (t Task) Execute(sync bool) {
	log.WithFields(log.Fields{"task": t.UUID}).Info("Executing Task")
	successes := 0
	count := 0
	if t.ExecutionAction.String() != "00000000-0000-0000-0000-000000000000" {
		count++
		if t.Execution.ExecuteAction(sync) {
			successes++
		}
	}
	// only execute promise at this point if the sync type is async, e.g. we are not waiting on completion message
	// TODO - how to execute promise when completion message is returned?
	if t.PromiseAction.String() != "00000000-0000-0000-0000-000000000000" && sync == false {
		count++
		if t.Promise.ExecuteAction(sync) {
			successes++
		}
	}
	if successes == 0 {
		t.Status = TaskFailed
	} else if count != successes {
		t.Status = TaskPartiallyFailed
	} else {
		t.Status = TaskComplete
	}
	t.CreateOrUpdate()
}
