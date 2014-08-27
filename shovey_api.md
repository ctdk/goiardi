The Shovey API
--------------

### HTTP API

#### Shovey job control

`/shovey/jobs`

	Method: GET
		List all jobs on the server

		Response body format:

	Method: POST
		Create a new shovey job.

		Request body format:

		Response body format:

`/shovey/jobs/<JOB ID>`

	Method: GET
		Information about a shovey jobs status, both overall and each
		node's status.

		Response body format:
	
`/shovey/jobs/<JOB ID>/<NODENAME>`

	Method: GET
		Provides detailed information about a shovey run on a specific
		node.

		Response body format:

`/shovey/jobs/cancel`

	Methods: PUT
		Cancels a job.

		Request body format:

		Response body format:

#### Node status

`/status/all/nodes`

	Methods: GET

		Response Body format:

`/status/node/<NODENAME>/all`

	Methods: GET
		
		Response body format:

`/status/node/<NODENAME>/latest`

	Methods: GET

		Response body format:

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
	signature: assembled from the JSON payload by joining the elements of the JSON payload that aren't the signature, separated by newlines, in alphabetical order. The goiardi server must be given an RSA private key to sign the request with, and schob must have the public key matching that private key to verify the request.

The block to sign will look something like this:

>action: start
>command: foo
>run_id: b5a6ee64-67ca-4a4f-94ad-6c18eb1c6a32
