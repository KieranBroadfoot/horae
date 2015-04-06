horae
=====

horae is a thought-experiment in large scale activity scheduling for microservices. Within a large scale software architecture scheduled tasks are an inevitable requirement. In the majority of cases these tasks can be easily scheduled via a range of techniques: cron, quartz, or control-M/autosys (if you prefer enterprise s/w).  However, in many cases there is a need to dis-intermediate the scheduling requirement from the original software; by which we mean to say that the logic/rules/definitions are maintained independently of the original s/w component which would enact the task. The closest we get is the quartz java library where the scheduling rules are typically defined within the software component itself.  Nonetheless, in many cases the scheduling logic is maintained independently and hence we start to lose the strict bounded context of the service.  Moreover, in an architecture where potentially thousands of services are regularly undertaking federated scheduled activity it is hard to reason on their interdependencies.  It is these two problems that prompted this experiment.

Dis-intermediation (i.e. it has an API)
---------------------------------------

Maintaining the bounded context is simplified within horae by exposing its scheduling capability via an API allowing services to define and manage their tasks and scheduled activity through a simple http interface. It is expected that any service utilising horae will create/manage their tasks/queues on startup to maintain the configuration within their code base. The horae model is based on queues and tasks.

Full API documentation can be found at [horae.kieranbroadfoot.com](http://horae.kieranbroadfoot.com)

Tasks
-----

Tasks are the core execution model within the scheduler. horae is only able to request activity on remote systems via API call and hence removes much of the baggage of enterprise class solutions in order to focus purely on scheduling.  Tasks may define an execution time when executed within the context of an async queue (see below), otherwise they are executed in a FIFO manner when a sync queue is open.  If no queue is specified the task is assigned to the root queue which is defined as async.

Tasks reference two discrete actions, the execution context and promise.  The execution context is always required and specifies the primary action to be executed by horae.  The promise is optional and provides a mechanism by which a callback to the originating service (or something else) may be executed on completion of the task.  

Tasks when operating within a synchronous context MUST signal their completion before the next task may be executed.  Hence the completion API call must be called.  Tasks executing within a sync context may specify an optional priority value.  The default is 0.  Choosing a higher value will re-order the FIFO queue with priorities ordered accordingly.  Use carefully.

Actions
-------

Actions are re-usable objects which describe a remote activity.  In both the execution and promise contexts the action may include:

* URI: where the call should be made
* Operation: http verb required for the executing call (GET, POST, HEAD, DELETE)
* Payload: an optional blob of data (likely json) which is sent to the remote service (dependent on operation type)

The payload mechanism provides a very simple templating mechanism to enable horae to include information which may be relevant to the receiving system.  For example if a payload is of the form `'{"taskUuid":"<<HORAE_TASK_UUID>>","status":"<<HORAE_TASK_STATUS>>"}'` horae will resolve these tags before POSTing to the action URI.

The template tags are:

* HORAE_API_URI
* HORAE_COMPLETION_URI
* HORAE_TASK_UUID
* HORAE_TASK_STATUS

Queues
------

Queues act as a window of operation in which tasks may be actioned.  Queues have two interesting properties: windows and paths.  Windows define when the queue is "open".  Open means the tasks associated to the queue may be executed.  Windows are defined using a simple NLP mechanism.  Examples include:

* always
* always on except 2 - 4am every sunday
* never
* 4.34am - 17:00 every day
* 1am to 11pm every thursday
* 1am to 2am every 1st of the month
* 4am to 6am every 01/01 yearly
* 10pm to 4am every saturday where timezone = GMT

Queues will expose the notion of a path.  A path is an arbitrary / delimited string which represents a logical or physical context for the queue.  Examples might include:

* /datacenters/dc2/east/rack10
* /apps/my_application/middle_tier

The purpose of the path is to define "containment".  What this means is that when horae attempts to open a queue for operation it will determine if all queues within its path are also open.  If *any* queue within the path is closed the queue will not open until these conditions are met.  Let's talk through the second example: /apps/my_application/middle_tier.  If the middle_tier queue is defined as "2am - 4am every saturday" horae will attempt to open the queue at 2am on saturday.  When it does so it will check for the existence of the following queues and ensure they are actively open and operating:

* /apps/my_application
* /apps
* root

Why might this be useful?  Here's some reasons:

1. Adding a queue with the path /apps which is defined as "always on except 00:00 - 11:59 on saturday" disables all scheduled tasks across all business applications on this coming saturday (e.g. a scheduled downtime activity)
2. Signalling availability and activity of an entire application by managing a queue at /apps/my_application

It should be noted that horae will enable a default queue named "root" which is always available, configured in async operation and cannot be modified.

Queues may be defined as sync or async.  Synchronous queues are serial in operation using FIFO with a simple prioritisation capability.  This means tasks placed in a synchronous queue will be executed in order when the queue is open.  However, greater flexibility is afforded with async queues where horae will execute tasks at a specific point in time (as defined by the task) during the queues open window.  As noted in the task section sync queues expect the action to be "completed" via callback from the executing service.

Finally we should mention backpressure for sync orientated queues.  These queues may define a callback action which is executed when the queue depth reaches a given number.  At this point these callbacks only occur when the queue is open but can be used to signal potential downstream issues, or the potential need to scale the associated services to handle the load.

Examples
--------

Create a new synchronous queue within horae:

    curl -X PUT -d '{"name":"My First Sync Queue","queueType":"sync","windowOfOperation":"always"}' http://horae.dev:8015/v1/queue
    {"uuid":"b7eac25a-dc46-11e4-a977-14109fd1718f","name":"My First Sync Queue","queueType":"sync","windowOfOperation":"always"}
    
Next, we'll create an action (with inline templating). When this action is run we're tasking horae with calling it's own completion URI hence it's NO-OP (unless you're watching the console output):

    curl -X PUT -d '{"name":"My First Action","operation":"GET","uri":"<<HORAE_COMPLETION_URI>>"}' http://horae.dev:8015/v1/action
    {"uuid":"c49f14b2-dc46-11e4-a978-14109fd1718f","operation":"GET","uri":"\u003c\u003cHORAE_COMPLETION_URI\u003e\u003e"}

Finally, join the action and queue via a task:

    curl -X PUT -d '{"name":"My First Task","queue":"b7eac25a-dc46-11e4-a977-14109fd1718f","execution":"c49f14b2-dc46-11e4-a978-14109fd1718f","priority":10}' http://horae.dev:8015/v1/task
    {"uuid":"3ee1ea5e-dc47-11e4-a979-14109fd1718f","name":"My First Task","priority":10,"queue":"b7eac25a-dc46-11e4-a977-14109fd1718f","when":"1974-12-31T00:00:00Z","execution":"c49f14b2-dc46-11e4-a978-14109fd1718f","status":"Pending"}

By this point horae will have started the queue hence this task will be immediately executed.

Architecture
------------

horae is built using golang and is expected to be deployed with both an etcd and cassandra cluster.  horae utilises cassandra as its permanent store but the majority of its running state is maintained via etcd.  The maintenance of keys within etcd enables each node within the cluster to claim and maintain ownership of the API endpoint or a given queue in order to balance workloads across the cluster.  Only a single node within the cluster will be successful in becoming the API endpoint for the cluster, this can be discovered by reviewing /horae/clusters/_name_/nodes or enabling vulcand support.

Setup
-----

In it's simplest form horae may be deployed with a single instance of etcd and cassandra.  After starting both etcd and cassandra a simple keyspace must be defined within cassandra.  Use schema.cql to load a standard data model.  Note: if you wish to run multiple horae clusters it is advised to change the name of the keyspace to match your preferred cluster name.  The default keyspace is horae_default.

Once cassandra has been configured any number of instances of horae may be started.  Configuration options are as follows:

* -static-port (or env: HORAE_USE_STATIC_PORT): specify whether you want horae to use static ports, use if you are running each node on a dedicated address (such as via docker).
* -clustername (or env: HORAE_CLUSTERNAME): the name of your horae cluster, do not include horae_
* -cassandra-address (or env: HORAE_CASSANDRA_ADDRESS): the host address of your cassandra cluster
* -etcd-address (or env: HORAE_ETCD_ADDRESS): the host/port combination for your etcd cluster