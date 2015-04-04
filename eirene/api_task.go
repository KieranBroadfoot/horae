package eirene

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/kieranbroadfoot/horae/types"
	"io/ioutil"
	"net/http"
)

// @Title querytask
// @Description The task endpoint will return a known Task with the appropriate UUID.  UUIDs are created by Horae during creation.
// @Accept  json
// @Param   uuid     path    string     false        "UUID of the requested task"
// @Success 200 {object} types.Task
// @Failure 400 {object} types.Error
// @Resource /tasks
// @Router /task/{uuid} [get]
func getTask(w http.ResponseWriter, r *http.Request, toEunomia chan types.EunomiaRequest) {
	vars := mux.Vars(r)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	task, terr := types.GetTask(vars["uuid"])
	if terr != nil {
		returnError(w, 404, "Task not found")
	} else {
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(task); err != nil {
			panic(err)
		}
	}
}

// @Title createtask
// @Description The endpoint defines a method to create a task within Horae.  The task must always provide an action reference to be executed on initiation.  It must also define EITHER a queue into which it should be placed or an execution time (in UTC).  If an execution time is requested the task MUST be placed into the "default" queue.  Optionally a task may define a series of tags in order to aid in searching.
// @Accept  json
// @Param   task     query    types.Task     true        "A task object"
// @Success 200 {object} types.Success
// @Failure 400 {object} types.Error
// @Resource /tasks
// @Router /task [put]
func createTask(w http.ResponseWriter, r *http.Request, toEunomia chan types.EunomiaRequest) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	task := new(types.Task)
	err := json.NewDecoder(r.Body).Decode(task)
	if err != nil {
		returnError(w, 400, "Badly formed request")
	} else {
		// TODO - update docs to show that uuid should not be passed on creation
		if task.UUID.String() != "00000000-0000-0000-0000-000000000000" {
			// marshalling json will create a dummy UUID if one was not specified.
			returnError(w, 400, "Task not saved: cannot specify UUID on create")
		} else {
			terr := task.CreateOrUpdate()
			if terr != nil {
				returnError(w, 400, "Task not saved: "+terr.Error())
			} else {
				w.WriteHeader(http.StatusOK)
				if err := json.NewEncoder(w).Encode(task); err != nil {
					panic(err)
				}
				toEunomia <- types.EunomiaRequest{Action: types.EunomiaStoreUpdate, Key: "updates/tasks/"+task.UUID.String(), Value: types.EunomiaActionCreate, TTL: 20}
			}
		}
	}
}

// @Title updatetask
// @Description A task may ONLY update its callback API endpoint or execution time (if it exists within the default queue).  If the task needs to be moved between queues then both delete and create should be undertaken.  The task may require changes to its behaviour to meet the requisite queues behaviour.
// @Accept  json
// @Param   uuid     path   string     	true        "UUID for updated task"
// @Param	queue	 query	types.Task  true		"A task object"
// @Success 200 {object} types.Success
// @Failure 400 {object} types.Error
// @Resource /tasks
// @Router /task/{uuid} [put]
func updateTask(w http.ResponseWriter, r *http.Request, toEunomia chan types.EunomiaRequest) {
	// TODO - check for queue changes and fail.
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	vars := mux.Vars(r)
	task, terr := types.GetTask(vars["uuid"])
	if terr != nil {
		returnError(w, 400, "Task not updated: "+terr.Error())
	} else {
		data, ioerr := ioutil.ReadAll(r.Body)
		if ioerr != nil {
			returnError(w, 400, "Unable to read incoming json")
		} else {
			err := json.Unmarshal(data, &task)
			if err != nil {
				returnError(w, 400, "Badly formed request")
			} else {
				terr := task.CreateOrUpdate()
				if terr != nil {
					returnError(w, 400, "Task not updated: "+terr.Error())
				} else {
					returnSuccess(w, "Task updated")
					toEunomia <- types.EunomiaRequest{Action: types.EunomiaStoreUpdate, Key: "updates/tasks/"+task.UUID.String(), Value: types.EunomiaActionUpdate, TTL: 20}
				}
			}
		}
	}
}

// @Title deletetask
// @Description When a task is deleted it will be immediately removed unless it is currently in execution (via a sync queue).  In this case the task will continue to complete as expected.
// @Accept  json
// @Param   uuid     	path    string     	true    "UUID of the task to be deleted"
// @Success 200 {object} types.Success
// @Failure 400 {object} types.Error
// @Resource /tasks
// @Router /task/{uuid} [delete]
func deleteTask(w http.ResponseWriter, r *http.Request, toEunomia chan types.EunomiaRequest) {
	vars := mux.Vars(r)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	task, terr := types.GetTask(vars["uuid"])
	if terr != nil {
		returnError(w, 404, "Task not found")
	} else {
		terr := task.Delete()
		if terr != nil {
			returnError(w, 400, "Task not deleted: "+terr.Error())
		} else {
			returnSuccess(w, "Task deleted")
			toEunomia <- types.EunomiaRequest{Action: types.EunomiaStoreUpdate, Key: "updates/tasks/"+task.UUID.String(), Value: types.EunomiaActionDelete, TTL: 20}
		}
	}
}

// @Title completetask
// @Description When a task is defined within a synchronous queue it is essential that it signals completion to Horae.  This endpoint provides that completion mechanism.
// @Accept  json
// @Param   uuid     	path    string     	true    "UUID of completing task"
// @Success 200 {object} types.Success
// @Failure 400 {object} types.Error
// @Resource /tasks
// @Router /task/{uuid}/complete [get]
func completeTask(w http.ResponseWriter, r *http.Request, toEunomia chan types.EunomiaRequest) {
	vars := mux.Vars(r)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	task, terr := types.GetTask(vars["uuid"])
	if terr != nil {
		returnError(w, 404, "Task not found")
	} else {
		terr := task.SetStatus(types.TaskComplete)
		if terr != nil {
			returnError(w, 400, "Task not completed: "+terr.Error())
		} else {
			returnSuccess(w, "Task completed")
			toEunomia <- types.EunomiaRequest{Action: types.EunomiaStoreUpdate, Key: "updates/tasks/"+task.UUID.String(), Value: types.EunomiaActionComplete, TTL: 20}
		}
	}
}
