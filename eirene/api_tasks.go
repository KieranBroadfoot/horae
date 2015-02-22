package eirene

import (
	"encoding/json"
	"github.com/kieranbroadfoot/horae/types"
	"net/http"
	"net/url"
)

// @Title tasks
// @Description This endpoint will return tasks known to Horae, either as an array or based on specific criteria (tag or queue).
// @Accept  json
// @Param   tag     query    string     false        "Tag against which you wish to limit tasks returned"
// @Param   queue   query    string     false        "UUID of queue to scope tasks returned"
// @Success 200 {array}  types.Task
// @Failure 400 {object} types.Error
// @Resource /tasks
// @Router /tasks [get]
func getTasks(w http.ResponseWriter, r *http.Request, toEunomia chan types.EunomiaRequest) {
	u, _ := url.Parse(r.URL.String())
	queryParams := u.Query()
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if val, ok := queryParams["tag"]; ok {
		if err := json.NewEncoder(w).Encode(types.GetTasksByTag(val[0])); err != nil {
			panic(err)
		}
	} else {
		if err := json.NewEncoder(w).Encode(types.GetTasks()); err != nil {
			panic(err)
		}
	}
}
