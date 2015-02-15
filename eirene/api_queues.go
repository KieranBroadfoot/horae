package eirene

import (
	"encoding/json"
	"github.com/kieranbroadfoot/horae/types"
	"net/http"
	"net/url"
)

// @Title queues
// @Description The queues endpoint provides information regarding the available queues known to Horae. This will always include the "default" asynchronous queue.
// @Accept  json
// @Param   tag     query    string     false        "Tag against which you wish to limit queues returned"
// @Success 200 {array}  types.Queue
// @Failure 400 {object} types.Error
// @Resource /queues
// @Router /queues [get]
func getQueues(w http.ResponseWriter, r *http.Request, toEunomia chan types.EunomiaRequest) {
	// TODO - consider how best to marshall tags and paths into resulting object

	// this gets us the variables out of the route from gorilla
	//vars := mux.Vars(r)
	//fmt.Println(vars["tag"])

	// gets any query parameters from the URL itself
	u, _ := url.Parse(r.URL.String())
	queryParams := u.Query()
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if val, ok := queryParams["tag"]; ok {
		if err := json.NewEncoder(w).Encode(types.GetQueuesByTag(val[0])); err != nil {
			panic(err)
		}
	} else {
		if err := json.NewEncoder(w).Encode(types.GetQueues()); err != nil {
			panic(err)
		}
	}
}
