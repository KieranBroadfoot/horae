// @APIVersion 1.0
// @APITitle Horae API
// @APIDescription Horae is a thought experiment in datacenter scheduling for micro-services. Managing datacenters at scale requires complex scheduling incorporating numerous windows of operation on discrete systems/platforms.  Incorporating scheduling into each and every service which may need to interact with these systems and platforms can lead to increased complexity and fails to provide a singular pane of glass through which to view the activities which are scheduled for operation.  Horae attempts to tackle this challenge by providing a simple API into which tasks may be deposited into queues for action at a later time.  It does not attempt to off-load the task activity itself but rather simply initiates a callback a service at execution time.  This ensures simplicity of the scheduler but ensures business logic is not unevenly distributed across many components of the architecture.  Simple workflows *could* be expressed where each node of the workflow is only aware of its downstream partner.  Horae also has support for backpressure management of queues by providing a mechanism for a service to monitor the behaviour of the queue itself.
// @Contact horae@kieranbroadfoot.com
// @TermsOfServiceUrl http://horae.kieranbroadfoot.com
// @License Licensed as Apache 2.0
// @LicenseUrl http://www.apache.org/licenses/LICENSE-2.0
// @SubApi Queues [/queues]
// @SubApi Tasks [/tasks]
// @SubApi Actions [/actions]

package eirene

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/kieranbroadfoot/horae/types"
	"net"
	"net/http"
	"errors"
)

func startAPIInterface(toCore chan bool, toEunomia chan types.EunomiaRequest, listener net.Listener, mw *MasterSlave) {
	log.Print("Starting API Interface")

	router := mux.NewRouter()
	router.HandleFunc("/v1/actions", func(w http.ResponseWriter, r *http.Request) { getActions(w, r, toEunomia) }).Methods("GET")
	router.HandleFunc("/v1/action/{uuid}", func(w http.ResponseWriter, r *http.Request) { getAction(w, r, toEunomia) }).Methods("GET")
	router.HandleFunc("/v1/action", func(w http.ResponseWriter, r *http.Request) { createAction(w, r, toEunomia) }).Methods("PUT")
	router.HandleFunc("/v1/action/{uuid}", func(w http.ResponseWriter, r *http.Request) { updateAction(w, r, toEunomia) }).Methods("PUT")
	router.HandleFunc("/v1/action/{uuid}", func(w http.ResponseWriter, r *http.Request) { deleteAction(w, r, toEunomia) }).Methods("DELETE")
	router.HandleFunc("/v1/tasks", func(w http.ResponseWriter, r *http.Request) { getTasks(w, r, toEunomia) }).Methods("GET")
	router.HandleFunc("/v1/task/{uuid}", func(w http.ResponseWriter, r *http.Request) { getTask(w, r, toEunomia) }).Methods("GET")
	router.HandleFunc("/v1/task", func(w http.ResponseWriter, r *http.Request) { createTask(w, r, toEunomia) }).Methods("PUT")
	router.HandleFunc("/v1/task/{uuid}", func(w http.ResponseWriter, r *http.Request) { updateTask(w, r, toEunomia) }).Methods("PUT")
	router.HandleFunc("/v1/task/{uuid}", func(w http.ResponseWriter, r *http.Request) { deleteTask(w, r, toEunomia) }).Methods("DELETE")
	router.HandleFunc("/v1/task/{uuid}/complete", func(w http.ResponseWriter, r *http.Request) { completeTask(w, r, toEunomia) }).Methods("GET")
	router.HandleFunc("/v1/queues", func(w http.ResponseWriter, r *http.Request) { getQueues(w, r, toEunomia) }).Methods("GET")
	router.HandleFunc("/v1/queue/{uuid}", func(w http.ResponseWriter, r *http.Request) { getQueue(w, r, toEunomia) }).Methods("GET")
	router.HandleFunc("/v1/queue", func(w http.ResponseWriter, r *http.Request) { createQueue(w, r, toEunomia) }).Methods("PUT")
	router.HandleFunc("/v1/queue/{uuid}", func(w http.ResponseWriter, r *http.Request) { updateQueue(w, r, toEunomia) }).Methods("PUT")
	router.HandleFunc("/v1/queue/{uuid}", func(w http.ResponseWriter, r *http.Request) { deleteQueue(w, r, toEunomia) }).Methods("DELETE")
	negroni := negroni.New(NewEireneLogger())
	negroni.Use(mw)
	negroni.UseHandler(router)

	err := http.Serve(listener, negroni)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Fatal("API Interface has failed")
		toCore <- true
	}
}

func StartEirene(node types.Node, staticPort bool, signalToCore chan types.Node, failureToCore chan bool, toEunomia chan types.EunomiaRequest, fromEunomia chan types.EireneStrategyAction) {
	log.Print("Starting Eirene")

	// initialise a listener with a random port
	ipaddr, _ := findExternalInterface()
	port := ":0"
	if staticPort {
		port = ":8015"
	}
	listener, err := net.Listen("tcp", ipaddr+port)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Fatal("Setting API Address")
		failureToCore <- true
	}
	log.WithFields(log.Fields{"addr": listener.Addr().String()}).Info("Setting API Address")

	// init and keep reference to middleware
	middleware := NewMasterSlave()
	middleware.setUnavailable()

	go startAPIInterface(failureToCore, toEunomia, listener, middleware)

	// Now we've init'd the core API service we can announce our existence to the core
	node.Address = listener.Addr().(*net.TCPAddr).IP.String()
	node.Port = fmt.Sprintf("%d", listener.Addr().(*net.TCPAddr).Port)
	signalToCore <- node

	for {
		select {
		case strategyUpdate := <-fromEunomia:
			// change routing strategy
			//var strategyMessage types.EireneStrategyAction
			//err := json.Unmarshal([]byte(strategyUpdate), &strategyMessage)
			if err == nil {
				if strategyUpdate.Action == "master" {
					if !middleware.isMaster() {
						log.WithFields(log.Fields{"state": "master"}).Info("Changing API Routing Strategy")
						middleware.setAvailableAsMaster(node.Address, node.Port)
					}
				} else if strategyUpdate.Action == "slave" {
					currentAddr, currentPort := middleware.currentMaster()
					if currentAddr != strategyUpdate.Address && currentPort != strategyUpdate.Port {
						log.WithFields(log.Fields{"state": "slave"}).Info("Changing API Routing Strategy")
						middleware.setAvailableAsSlave(strategyUpdate.Address, strategyUpdate.Port)
					}
				} else if strategyUpdate.Action == "unavailable" {
					log.WithFields(log.Fields{"state": "unavailable"}).Info("Changing API Routing Strategy")
					middleware.setUnavailable()
				}
				if node.MasterURI != middleware.currentMasterAsURI() {
					node.MasterURI = middleware.currentMasterAsURI()
					signalToCore <- node
				}
			}
		}
	}
}

func returnSuccess(w http.ResponseWriter, message string) {
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(types.Success{Message: message}); err != nil {
		panic(err)
	}
}

func returnError(w http.ResponseWriter, code uint32, message string) {
	w.WriteHeader(http.StatusBadRequest)
	if err := json.NewEncoder(w).Encode(types.Error{Code: code, Message: message}); err != nil {
		panic(err)
	}
}

// TODO - presumes IPV4, needs support for IPV6?
func findExternalInterface() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
				case *net.IPNet:
				ip = v.IP
				case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String(), nil
		}
	}
	return "", errors.New("No valid network connection")
}