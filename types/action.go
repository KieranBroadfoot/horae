package types

import (
	"bytes"
	"errors"
	log "github.com/Sirupsen/logrus"
	"github.com/gocql/gocql"
	"github.com/relops/cqlr"
	"net/http"
	"time"
)

type Action struct {
	UUID      gocql.UUID `cql:"action_uuid"`
	Operation string     `cql:"operation"`
	Payload   string     `cql:"payload"`
	URI       string     `cql:"uri"`
	Status    string     `cql:"status"`
	Failure   string     `cql:"failure"`
	OurTags   []string   `json:"tags,omitempty" description:"Tags assigned to the action."`
}

func GetActions() []Action {
	query := session.Query("select * from actions")
	bind := cqlr.BindQuery(query)
	var action Action
	actions := []Action{}
	for bind.Scan(&action) {
		action.LoadTags()
		actions = append(actions, action)
	}
	return actions
}

func GetActionsByTag(tag string) []Action {
	var id gocql.UUID
	var action Action
	actions := []Action{}
	iteration := session.Query("select object_uuid from tags where type = 'action' and tag = ? allow filtering", tag).Iter()
	for iteration.Scan(&id) {
		q := session.Query("select * from actions where action_uuid = ? allow filtering", id)
		b := cqlr.BindQuery(q)
		b.Scan(&action)
		action.LoadTags()
		actions = append(actions, action)
	}
	return actions
}

func GetAction(actionUUID string) (Action, error) {
	query := session.Query("select * from actions where action_uuid = ?", actionUUID)
	bind := cqlr.BindQuery(query)
	var action Action
	if !bind.Scan(&action) {
		return Action{}, errors.New("Unknown action")
	}
	action.LoadTags()
	return action, nil
}

func (action *Action) CreateOrUpdate() error {
	bind := cqlr.Bind(`insert into actions (action_uuid, operation, uri, payload, status, failure) values (?, ?, ?, ?, ?, ?)`, action)
	if err := bind.Exec(session); err != nil {
		log.Print("GOT ERR: ", err)
		return err
	} else {
		return nil
	}
}

func (action *Action) Delete() error {
	action.DeleteTags()
	bind := cqlr.Bind(`delete from actions where action_uuid = ?`, action)
	if err := bind.Exec(session); err != nil {
		log.Print("received error from delete")
		return err
	} else {
		return nil
	}
}

func (a *Action) LoadTags() {
	a.OurTags = GetTagsForObject(a.UUID)
}

func (a Action) CreateOrUpdateTags() {
	SetTagsForObject(a.UUID, a.OurTags, "action")
}

func (a Action) DeleteTags() {
	DeleteTagsForObject(a.UUID)
}

func (action *Action) ExecuteAction(sync bool) bool {
	start := time.Now()
	log.WithFields(log.Fields{"action": action.UUID, "URI": action.URI, "verb": action.Operation}).Info("Executing Action")
	response, error := action.makeRequest()

	if error != nil {
		action.Status = TaskFailed
		action.Failure = error.Error()
	} else {
		if response.StatusCode >= 200 && response.StatusCode < 300 {
			if sync {
				// set status to Running if successful and Queue is "sync" type. Waiting for completion message
				action.Status = TaskPending
			} else {
				// set status to Complete if successful and Queue is "async" type
				action.Status = TaskComplete
			}
		} else {
			action.Status = TaskFailed
		}
	}
	action.CreateOrUpdate()
	log.WithFields(log.Fields{"task": action.UUID, "status": action.Status, "time": time.Since(start)}).Info("Finished Task Execution")
	if action.Status == TaskComplete {
		return true
	} else {
		return false
	}
}

func (a Action) makeRequest() (resp *http.Response, err error) {
	switch {
	case a.Operation == TaskGet:
		return http.Get(a.URI)
	case a.Operation == TaskPut:
		request, err := http.NewRequest(TaskPut, a.URI, nil)
		if err != nil {
			return http.DefaultClient.Do(request)
		} else {
			// TODO - unable to undertake PUT. Needs fix
			return resp, errors.New("Unable to create PUT")
		}
	case a.Operation == TaskPost:
		return http.Post(a.URI, "application/json", bytes.NewBufferString(a.Payload))
	case a.Operation == TaskHead:
		return http.Head(a.URI)
	case a.Operation == TaskDelete:
		request, err := http.NewRequest(TaskDelete, a.URI, nil)
		if err != nil {
			return http.DefaultClient.Do(request)
		} else {
			return nil, err
		}
	}
	return nil, errors.New("No valided handler for " + a.Operation)
}
