package eirene

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/kieranbroadfoot/horae/types"
	"net/http"
)

// @Title queryaction
// @Description The action endpoint will return a known Action with the appropriate UUID.  UUIDs are created by Horae during creation.
// @Accept  json
// @Param   uuid     path    string     false        "UUID of the requested action"
// @Success 200 {object} types.Action
// @Failure 400 {object} types.Error
// @Resource /actions
// @Router /action/{uuid} [get]
func getAction(w http.ResponseWriter, r *http.Request, toEunomia chan types.EunomiaRequest) {
	vars := mux.Vars(r)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	action, terr := types.GetAction(vars["uuid"])
	if terr != nil {
		returnError(w, 404, "Action not found")
	} else {
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(action); err != nil {
			panic(err)
		}
	}
}

// @Title createaction
// @Description The endpoint defines a method to create an action within Horae.  The action must always provide the URI and Operation to call when initiated.  It may also include an optional payload value (typically a json blob) to be sent to the executing service. Optionally a action may define a series of tags in order to aid in searching.
// @Accept  json
// @Param   action     query    types.Action     true        "A action object"
// @Success 200 {object} types.Success
// @Failure 400 {object} types.Error
// @Resource /actions
// @Router /action [put]
func createAction(w http.ResponseWriter, r *http.Request, toEunomia chan types.EunomiaRequest) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	action := new(types.Action)
	err := json.NewDecoder(r.Body).Decode(action)
	if err != nil {
		returnError(w, 400, "Badly formed request")
	} else {
		// TODO - update docs to show that uuid should not be passed on creation
		if action.UUID.String() != "00000000-0000-0000-0000-000000000000" {
			// marshalling json will create a dummy UUID if one was not specified.
			returnError(w, 400, "Action not saved: cannot specify UUID on create")
		} else {
			terr := action.CreateOrUpdate()
			if terr != nil {
				returnError(w, 400, "Action not saved: "+terr.Error())
			} else {
				w.WriteHeader(http.StatusOK)
				if err := json.NewEncoder(w).Encode(action); err != nil {
					panic(err)
				}
				//toEunomia <- "FOO"
			}
		}
	}
}

// @Title updateaction
// @Description A action may update its callback API endpoint, payload or operation types.
// @Accept  json
// @Param   uuid     path   string     	true        "UUID for updated action"
// @Param	queue	 query	types.Action  true		"An action object"
// @Success 200 {object} types.Success
// @Failure 400 {object} types.Error
// @Resource /actions
// @Router /action/{uuid} [put]
func updateAction(w http.ResponseWriter, r *http.Request, toEunomia chan types.EunomiaRequest) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	action := new(types.Action)
	err := json.NewDecoder(r.Body).Decode(action)
	if err != nil {
		returnError(w, 400, "Badly formed request")
	} else {
		terr := action.CreateOrUpdate()
		if terr != nil {
			returnError(w, 400, "Action not updated: "+terr.Error())
		} else {
			//toEunomia <- "FOO"
			returnSuccess(w, "Action updated")
		}
	}
}

// @Title deleteaction
// @Description When an action is deleted it will be immediately removed.  It is advised to ensure tasks associated with the action are disabled/deleted in advance.
// @Accept  json
// @Param   uuid     	path    string     	true    "UUID of the action to be deleted"
// @Success 200 {object} types.Success
// @Failure 400 {object} types.Error
// @Resource /actions
// @Router /action/{uuid} [delete]
func deleteAction(w http.ResponseWriter, r *http.Request, toEunomia chan types.EunomiaRequest) {
	vars := mux.Vars(r)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	action, terr := types.GetAction(vars["uuid"])
	if terr != nil {
		returnError(w, 404, "Action not found")
	} else {
		terr := action.Delete()
		if terr != nil {
			returnError(w, 400, "Action not deleted: "+terr.Error())
		} else {
			//toEunomia <- "FOO"
			returnSuccess(w, "Action deleted")
		}
	}
}
