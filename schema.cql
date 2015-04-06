create keyspace horae_default with replication = { 'class' : 'SimpleStrategy', 'replication_factor' : 1 } and durable_writes = true;

use horae_default;

// queue

create table queues (
    queue_uuid uuid,
    status varchar,
    name varchar,
    queue_type varchar,
    window_of_operation varchar,
    should_drain boolean,
    backpressure_action uuid,
    backpressure_definition bigint,
    primary key (queue_uuid, status)
);

// task

create table tasks (
    task_uuid uuid primary key,
    queue_uuid uuid,
    name varchar,
    priority bigint,
    when timestamp,
    promise_action uuid,
    execution_action uuid,
    status varchar
);

// async tasks are executed in FIFO order (based on priority)
create table async_tasks (
    queue_uuid uuid,
    task_uuid uuid,
    when timestamp,
    status varchar,
    primary key ((queue_uuid, status), when, task_uuid)
) with clustering order by (when desc);

// sync tasks are executed based on a timestamp (if the window of operation is open)
create table sync_tasks (
    queue_uuid uuid,
    task_uuid uuid,
    priority bigint,
    status varchar,
    primary key ((queue_uuid, status), priority, task_uuid)
) with clustering order by (priority desc);

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
insert into queues (queue_uuid, name, queue_type, window_of_operation, should_drain, status) values (11111111-1111-1111-1111-111111111111, 'root', 'async', 'always', false, 'Active');
insert into paths (queue_uuid, path) values(11111111-1111-1111-1111-111111111111, '/');

// Test Data
insert into actions (action_uuid, operation, uri, payload) values (575d6070-a3f0-11e4-89d3-123b93f75cba, 'GET', 'http://www.test1.com', '');
insert into actions (action_uuid, operation, uri, payload) values (28941e26-a479-11e4-89d3-123b93f75cba, 'GET', 'http://www.l1fe.co.uk', '');
insert into actions (action_uuid, operation, uri, payload) values (28942088-a479-11e4-89d3-123b93f75cba, 'GET', 'http://www.verdacc.io', '');
insert into actions (action_uuid, operation, uri, payload) values (290f4671-4793-48d9-b332-1acd66d0c996, 'GET', '<<HORAE_COMPLETION_URI>>', '{"UUID":"<<HORAE_TASK_UUID>>", "STATUS": "<<HORAE_TASK_STATUS>>"}');

insert into queues (queue_uuid, name, queue_type, window_of_operation, should_drain, backpressure_action, backpressure_definition, status) values (2f8ed604-daba-11e4-b9d6-1681e6b88ec1, 'test async queue', 'async', 'always', false, 575d6070-a3f0-11e4-89d3-123b93f75cba, 10, 'Active');
insert into paths (queue_uuid, path) values(2f8ed604-daba-11e4-b9d6-1681e6b88ec1, '/datacenter/dc2/west/rack1');
insert into tags (object_uuid, type, tag) values (2f8ed604-daba-11e4-b9d6-1681e6b88ec1, 'queue', 'dc2_west_rack1');

insert into queues (queue_uuid, name, queue_type, window_of_operation, should_drain, backpressure_action, backpressure_definition, status) values (cfd66ccc-d857-4e90-b1e5-df98a3d40cd6, 'test sync queue', 'sync', 'always', false, 575d6070-a3f0-11e4-89d3-123b93f75cba, 20, 'Active');
insert into paths (queue_uuid, path) values(cfd66ccc-d857-4e90-b1e5-df98a3d40cd6, '/datacenter/dc1/east/rack42');
insert into tags (object_uuid, type, tag) values (cfd66ccc-d857-4e90-b1e5-df98a3d40cd6, 'queue', 'dc2_east_rack42');

insert into tasks (task_uuid, queue_uuid, when, status, execution_action) values (d9c4b820-13b2-43a2-aac8-26ee4a55bb14, 2f8ed604-daba-11e4-b9d6-1681e6b88ec1, '2015-03-15 15:50', 'Pending', 575d6070-a3f0-11e4-89d3-123b93f75cba) ;
insert into tasks (task_uuid, queue_uuid, when, status, execution_action) values (eed6116e-54d1-40fc-9513-18d35f2018d1, 2f8ed604-daba-11e4-b9d6-1681e6b88ec1, '2015-03-15 15:30', 'Pending', 28941e26-a479-11e4-89d3-123b93f75cba) ;
insert into tasks (task_uuid, queue_uuid, when, status, execution_action) values (e233c3b6-faf6-4297-b12c-6c0467e53d7b, 2f8ed604-daba-11e4-b9d6-1681e6b88ec1, '2015-03-15 16:02', 'Pending', 575d6070-a3f0-11e4-89d3-123b93f75cba) ;
insert into tasks (task_uuid, queue_uuid, when, status, execution_action) values (d7841a23-c5e5-479b-a36f-7b3b8da5e19a, 2f8ed604-daba-11e4-b9d6-1681e6b88ec1, '2015-03-15 16:00', 'Pending', 28942088-a479-11e4-89d3-123b93f75cba) ;
insert into tasks (task_uuid, queue_uuid, when, status, execution_action) values (7091fc0f-14f0-41d2-8851-cbdf04a74f87, 2f8ed604-daba-11e4-b9d6-1681e6b88ec1, '2015-03-16 16:00', 'Pending', 575d6070-a3f0-11e4-89d3-123b93f75cba) ;
insert into tasks (task_uuid, queue_uuid, when, status, execution_action) values (d8fca102-a8a4-4163-a6c6-4bead0de80d8, 2f8ed604-daba-11e4-b9d6-1681e6b88ec1, '2015-03-15 16:45', 'Pending', 28941e26-a479-11e4-89d3-123b93f75cba) ;
insert into tasks (task_uuid, queue_uuid, when, status, execution_action) values (67e1d924-b512-4d5e-8212-20be51ac08db, 2f8ed604-daba-11e4-b9d6-1681e6b88ec1, '2015-03-15 17:32', 'Pending', 28942088-a479-11e4-89d3-123b93f75cba) ;

insert into async_tasks (queue_uuid, task_uuid, when, status) values (2f8ed604-daba-11e4-b9d6-1681e6b88ec1, d9c4b820-13b2-43a2-aac8-26ee4a55bb14, '2015-03-15 15:50', 'Pending') ;
insert into async_tasks (queue_uuid, task_uuid, when, status) values (2f8ed604-daba-11e4-b9d6-1681e6b88ec1, eed6116e-54d1-40fc-9513-18d35f2018d1, '2015-03-15 15:30', 'Pending') ;
insert into async_tasks (queue_uuid, task_uuid, when, status) values (2f8ed604-daba-11e4-b9d6-1681e6b88ec1, e233c3b6-faf6-4297-b12c-6c0467e53d7b, '2015-03-15 16:02', 'Pending') ;
insert into async_tasks (queue_uuid, task_uuid, when, status) values (2f8ed604-daba-11e4-b9d6-1681e6b88ec1, d7841a23-c5e5-479b-a36f-7b3b8da5e19a, '2015-03-15 16:00', 'Pending') ;
insert into async_tasks (queue_uuid, task_uuid, when, status) values (2f8ed604-daba-11e4-b9d6-1681e6b88ec1, 7091fc0f-14f0-41d2-8851-cbdf04a74f87, '2015-03-16 16:00', 'Pending') ;
insert into async_tasks (queue_uuid, task_uuid, when, status) values (2f8ed604-daba-11e4-b9d6-1681e6b88ec1, d8fca102-a8a4-4163-a6c6-4bead0de80d8, '2015-03-15 16:45', 'Pending') ;
insert into async_tasks (queue_uuid, task_uuid, when, status) values (2f8ed604-daba-11e4-b9d6-1681e6b88ec1, 67e1d924-b512-4d5e-8212-20be51ac08db, '2015-03-15 17:32', 'Pending') ;

insert into tasks (task_uuid, queue_uuid, status, priority, execution_action) values (9adfe006-aef3-4d91-bf70-709057624d73, cfd66ccc-d857-4e90-b1e5-df98a3d40cd6, 'Pending', 0, 575d6070-a3f0-11e4-89d3-123b93f75cba) ;
insert into tasks (task_uuid, queue_uuid, status, priority, execution_action) values (bdb1806b-94e4-43fd-94c5-1ba7aa0e42c7, cfd66ccc-d857-4e90-b1e5-df98a3d40cd6, 'Pending', 10, 28941e26-a479-11e4-89d3-123b93f75cba) ;
insert into tasks (task_uuid, queue_uuid, status, priority, execution_action) values (e41274a7-6491-45f7-80d3-685118972378, cfd66ccc-d857-4e90-b1e5-df98a3d40cd6, 'Pending', 20, 575d6070-a3f0-11e4-89d3-123b93f75cba) ;
insert into tasks (task_uuid, queue_uuid, status, priority, execution_action) values (7c1933e4-898c-4b6e-b175-4c55714384d9, cfd66ccc-d857-4e90-b1e5-df98a3d40cd6, 'Pending', 0, 28942088-a479-11e4-89d3-123b93f75cba) ;
insert into tasks (task_uuid, queue_uuid, status, priority, execution_action) values (ab358ba8-4816-4b4f-b70e-d687d6ec4145, cfd66ccc-d857-4e90-b1e5-df98a3d40cd6, 'Pending', 0, 575d6070-a3f0-11e4-89d3-123b93f75cba) ;
insert into tasks (task_uuid, queue_uuid, status, priority, execution_action, promise_action) values (69a2a26d-bdfa-4fe2-bd67-43185fefc15c, cfd66ccc-d857-4e90-b1e5-df98a3d40cd6, 'Pending', 50, 28942088-a479-11e4-89d3-123b93f75cba, 28942088-a479-11e4-89d3-123b93f75cba) ;
insert into tasks (task_uuid, queue_uuid, status, priority, execution_action, promise_action) values (ab69938a-2a86-416e-bff7-47dab014b732, cfd66ccc-d857-4e90-b1e5-df98a3d40cd6, 'Pending', 100, 575d6070-a3f0-11e4-89d3-123b93f75cba, 28942088-a479-11e4-89d3-123b93f75cba) ;

insert into sync_tasks (queue_uuid, task_uuid, priority, status) values (cfd66ccc-d857-4e90-b1e5-df98a3d40cd6, 9adfe006-aef3-4d91-bf70-709057624d73, 0, 'Pending') ;
insert into sync_tasks (queue_uuid, task_uuid, priority, status) values (cfd66ccc-d857-4e90-b1e5-df98a3d40cd6, bdb1806b-94e4-43fd-94c5-1ba7aa0e42c7, 10, 'Pending') ;
insert into sync_tasks (queue_uuid, task_uuid, priority, status) values (cfd66ccc-d857-4e90-b1e5-df98a3d40cd6, e41274a7-6491-45f7-80d3-685118972378, 20, 'Pending') ;
insert into sync_tasks (queue_uuid, task_uuid, priority, status) values (cfd66ccc-d857-4e90-b1e5-df98a3d40cd6, 7c1933e4-898c-4b6e-b175-4c55714384d9, 0, 'Pending') ;
insert into sync_tasks (queue_uuid, task_uuid, priority, status) values (cfd66ccc-d857-4e90-b1e5-df98a3d40cd6, ab358ba8-4816-4b4f-b70e-d687d6ec4145, 0, 'Pending') ;
insert into sync_tasks (queue_uuid, task_uuid, priority, status) values (cfd66ccc-d857-4e90-b1e5-df98a3d40cd6, 69a2a26d-bdfa-4fe2-bd67-43185fefc15c, 50, 'Pending') ;
insert into sync_tasks (queue_uuid, task_uuid, priority, status) values (cfd66ccc-d857-4e90-b1e5-df98a3d40cd6, ab69938a-2a86-416e-bff7-47dab014b732, 100, 'Pending') ;
