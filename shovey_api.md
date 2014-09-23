The Shovey API
--------------

Documentation of the goiardi shovey HTTP and Serf APIs. As a work in progress, be aware that anything in this document is subject to change until shovey is officially released. This documentation covers all the endpoints shovey uses, both on the goiardi side and the schob (the shovey client that executes jobs) side, but at the moment is a little sparse. This will change as the documentation fills out.

### HTTP API

The Chef Pushy API located at http://docs.getchef.com/push_jobs.html#api-push-jobs is also relevant, but the Shovey HTTP API is not exactly the same as the Pushy API, for various reasons.

#### Shovey job control

`/shovey/jobs`

Methods: GET, PUT

Method: GET

List all jobs on the server. Returns a list of uuids of jobs.

Response body format:

```
[
  "036d8b61-da10-439b-ba1f-40f5f866c6b1",
  "04226cc8-0c9b-47e5-adaa-158ccc36f0b1",
  "1204692a-8e4c-4adb-960a-089d59c10fbf",
  "242957ce-10c5-4f7a-89d8-ffb478fd1ef9"
]
```

Method: POST

Create a new shovey job.

Request body format:

```
{
	"command": "foo",
	"quorum": "75%",
	"nodes": [ "foo.local", "bar.local" ]
}
```

Response body format:

```
{
	"id": "76b745eb-45d6-4856-94f9-7830e79cb8cd",
	"uri": "http://your.chef-server.local:4545/shovey/jobs/76b745eb-45d6-4856-94f9-7830e79cb8cd"
}
```


`/shovey/jobs/<JOB ID>`

Method: GET

Information about a shovey job's status, both overall and each node's status.

Response body format:

```
{
  "command": "ls",
  "created_at": "2014-08-26T21:44:24.636242093-07:00",
  "id": "76b745eb-45d6-4856-94f9-7830e79cb8cd",
  "nodes": {
    "succeeded": [
      "nineveh.local"
    ]
  },
  "run_timeout": 300,
  "status": "complete",
  "updated_at": "2014-08-26T21:44:25.079010129-07:00"
}
```

	
`/shovey/jobs/<JOB ID>/<NODENAME>`

Methods: GET, PUT

Method: GET

Provides detailed information about a shovey run on a specific node.

Response body format:

```
{
  "run_id": "76b745eb-45d6-4856-94f9-7830e79cb8cd",
  "node_name": "nineveh.local",
  "status": "succeeded",
  "ack_time": "2014-08-26T21:44:24.645047317-07:00",
  "end_time": "2014-08-26T21:44:25.078800724-07:00",
  "output": "Applications\nLibrary\nNetwork\nSystem\nUser Information\nUsers\nVolumes\nbin\ncores\ndev\netc\nhome\nmach_kernel\nnet\nopt\nprivate\nsbin\ntmp\nusr\nvar\n",
  "error": "",
  "stderr": "",
  "exit_status": 0
}
```

Method: PUT

Update a node's shovey run information on the server.

Request body format:

```
{
  "run_id": "76b745eb-45d6-4856-94f9-7830e79cb8cd",
  "node_name": "nineveh.local",
  "status": "succeeded",
  "ack_time": "2014-08-26T21:44:24.645047317-07:00",
  "end_time": "2014-08-26T21:44:25.078800724-07:00",
  "error": "",
  "exit_status": 0,
  "protocol_major": 0,
  "protocol_minor": 1
}
```

Response body format:

```
{
  "id": "76b745eb-45d6-4856-94f9-7830e79cb8cd",
  "node": "nineveh.local",
  "response": "ok"
}

```


`/shovey/jobs/cancel`

Methods: PUT

Cancels a job. The "nodes" option can either be a list of nodes to cancel the job on, or use an empty array to cancel the job on all nodes running this job.

Request body format:

```
{
  "run_id": "76b745eb-45d6-4856-94f9-7830e79cb8cd",
  "nodes": [ "foomer.local", "noober.snerber.com" ]
}
```

Response body format:

```
{
  "command"=>"sleepy", 
  "created_at"=>"2014-08-26T21:55:07.751851335-07:00",
  "id"=>"188d457e-2e07-40ef-954c-ab936af615b6",
  "nodes"=>{"cancelled"=>["nineveh.local"]},
  "run_timeout"=>300,
  "status"=>"cancelled",
  "updated_at"=>"2014-08-26T21:55:25.161713014-07:00"
}
```

#### Streaming output

`/shovey/stream/<JOB ID>/<NODE>`

Methods: GET, PUT

Method: GET

Streams the output from a job running on a node. Takes two query parameters: `sequence` and `output_type`. The `sequence` parameter is the the sequence record to start fetching from, while `output_type` sets the sort of output you'd like to receive. Acceptable values are 'stdout', 'stderr', and 'both'. The default value for `sequence` if none is given is 0, while the default for `output_type` is 'stdout'.

Response body format:

```
{
  "run_id": "188d457e-2e07-40ef-954c-ab936af615b6",
  "node_name": "foomer.local",
  "last_seq": 123,
  "is_last": false,
  "output_type": "stdout",
  "output": "foo"
}
```

Method: PUT

Add a chunk of output from a shovey job on a node to the log on the server for the job and node.

Request body format:

```
{
  "run_id": "188d457e-2e07-40ef-954c-ab936af615b6",
  "node_name": "foomer.local",
  "seq": 1,
  "is_last": false,
  "output_type": "stdout",
  "output": "foo"
}
```

Response body format:

```
{
  "response":"ok"
}
```

#### Node status

`/status/all/nodes`

Methods: GET

Get the latest status from every node on the server.

Response Body format:

```
[
  {
    "node_name": "nineveh.local",
    "status": "up",
    "updated_at": "2014-08-26T21:49:58-07:00",
    "url": "http://nineveh.local:4545/status/node/nineveh.local/latest"
  },
  {
    "node_name": "fooper.local",
    "status": "down",
    "updated_at": "2014-08-26T21:47:48-07:00",
    "url": "http://nineveh.local:4545/status/node/fooper.local/latest"
  }
]
```


`/status/node/<NODENAME>/all`

Methods: GET

Get a list of all statuses a particular node has had.
		
Response body format:

```
[
  {
    "node_name": "nineveh.local",
    "status": "up",
    "updated_at": "2014-08-26T21:51:28-07:00"
  },
  {
    "node_name": "nineveh.local",
    "status": "up",
    "updated_at": "2014-08-26T21:50:58-07:00"
  },
  {
    "node_name": "nineveh.local",
    "status": "up",
    "updated_at": "2014-08-26T21:50:28-07:00"
  },
  {
    "node_name": "nineveh.local",
    "status": "up",
    "updated_at": "2014-08-26T21:49:58-07:00"
  }
]

```


`/status/node/<NODENAME>/latest`

Methods: GET

Get the latest status of this particular node.

Response body format:

```
{
  "node_name": "nineveh.local",
  "status": "up",
  "updated_at": "2014-08-26T21:50:58-07:00"
}
```

### serf API

#### Node status

Sent by schob to goiardi over serf as a heartbeat message.

Serf parameters:

Name: node_status

Payload: JSON described below

RespCh: goiardi will respond to the heartbeat message over this response channel.

JSON payload parameters:

	node: name of the chef client/node.
	status: "up"

#### Shovey command

Sent by goiardi to schob over serf to start a shovey run on a node.

Serf parameters:

Name: "shovey"

Payload: JSON described below

FilterNodes: Limit the serf query to the given nodes

RequestAck: request an acknowledgement from schob

AckCh, RespCh: acknowledgement and response channels from schob to goiardi.

JSON payload parameters:

	run_id: the uuid of the shovey run
	action: the action to perform on the node. May be "start" or "cancel".
	command: the name of the command to run. Only required when action is "start".
	time: RFC3339 formatted current timestamp
	timeout: Time, in seconds, to kill the process if it hasn't finished by the time the timeout expires.
	signature: assembled from the JSON payload by joining the elements of the JSON payload that aren't the signature, separated by newlines, in alphabetical order. The goiardi server must be given an RSA private key to sign the request with, and schob must have the public key matching that private key to verify the request.


The block to sign will look something like this:

```
action: start
command: foo
run_id: b5a6ee64-67ca-4a4f-94ad-6c18eb1c6a32
time: 2014-09-05T23:00:00Z
timeout: 300
```
