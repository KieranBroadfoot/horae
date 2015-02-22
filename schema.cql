create keyspace horae_default with replication = { 'class' : 'SimpleStrategy', 'replication_factor' : 1 } and durable_writes = true;

use horae_default;

// queue

create table queues (
    queue_uuid uuid primary key,
    name varchar,
    queue_type varchar,
    window_of_operation varchar,
    should_drain boolean,
    backpressure_action uuid,
    backpressure_definition varchar
);

// task

create table tasks (
    task_uuid uuid,
    name varchar,
    priority bigint,
    queue_uuid uuid,
    when timestamp,
    promise_action uuid,
    execution_action uuid,
    status varchar,
    primary key (queue_uuid, task_uuid, when)
) with clustering order by (task_uuid asc, when asc);

// action

create table actions (
    action_uuid uuid primary key,
    operation varchar,
    uri varchar,
    payload varchar,
    status varchar,
    failure varchar,
);

// tags
// primary query: find tags for uuid
// secondary query: find uuids for tag (requires 'allow filtering')
create table tags (
    object_uuid uuid,
    type varchar,
    tag varchar,
    primary key (object_uuid, type, tag)
);

// paths

create table paths (
    queue_uuid uuid,
    path varchar,
    primary key (queue_uuid, path)
);

// Root Queue
insert into queues (queue_uuid, name, queue_type, window_of_operation, should_drain) values (11111111-1111-1111-1111-111111111111, 'root', 'async', '', false);

// Test Data
insert into actions (action_uuid, operation, uri, payload) values (575d6070-a3f0-11e4-89d3-123b93f75cba, 'GET', 'http://www.test1.com', '');
insert into actions (action_uuid, operation, uri, payload) values (28941e26-a479-11e4-89d3-123b93f75cba, 'GET', 'http://www.l1fe.co.uk', '');
insert into actions (action_uuid, operation, uri, payload) values (28942088-a479-11e4-89d3-123b93f75cba, 'GET', 'http://www.verdacc.io', '');
insert into queues (queue_uuid, name, queue_type, window_of_operation, should_drain, backpressure_action, backpressure_definition) values (cfd66ccc-d857-4e90-b1e5-df98a3d40cd6, 'test queue', 'async', '', false, 575d6070-a3f0-11e4-89d3-123b93f75cba, '');
insert into paths (queue_uuid, path) values(cfd66ccc-d857-4e90-b1e5-df98a3d40cd6, '/datacenter/dc2/west/rack1');
insert into tags (object_uuid, type, tag) values (cfd66ccc-d857-4e90-b1e5-df98a3d40cd6, 'queue', 'tag1');
insert into tasks (task_uuid, queue_uuid, execution_action, when) values (289421e6-a479-11e4-89d3-123b93f75cba, cfd66ccc-d857-4e90-b1e5-df98a3d40cd6, 28941e26-a479-11e4-89d3-123b93f75cba, '2015-06-01 12:00:00');
insert into tasks (task_uuid, queue_uuid, execution_action, promise_action, when) values (28942326-a479-11e4-89d3-123b93f75cba, cfd66ccc-d857-4e90-b1e5-df98a3d40cd6, 28942088-a479-11e4-89d3-123b93f75cba, 575d6070-a3f0-11e4-89d3-123b93f75cba, '2015-06-01 12:00:00');

