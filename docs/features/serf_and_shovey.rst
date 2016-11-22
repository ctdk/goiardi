.. _serf_and_shovey:

Serf
====

As of version 0.8.0, goiardi has some serf integration. At the moment it's mainly used for shovey (see below), but it will also announce that it's started up and joined a serf cluster.

If the ``--serf-event-announce`` flag is set, goiardi will announce logged events from the event log and starting up and joining the serf cluster over serf as serf user events. Be aware that if this is enabled, something will need to read these events from serf. Otherwise, the logged events will pile up and eventually take up all the space in the event queue and prevent any new events from being added.

Shovey
======

Shovey is a facility for sending jobs to nodes independently of a chef-client run, like Chef Push but serf based.

Shovey requirements
-------------------

To use shovey, you will need:

* Serf installed on the server goiardi is running on.
* Serf installed on the node(s) running jobs.
* ``schob``, the shovey client, must be installed on the node(s) running jobs.
* The ``knife-shove`` plugin must be installed on the workstation used to manage
  shovey jobs.

The client can be found at https://github.com/ctdk/schob, and a cookbook for installing the shovey client on a node is at https://github.com/ctdk/shovey-jobs. The ``knife-shove`` plugin can be found at https://github.com/ctdk/knife-shove or on rubygems.

Shovey Installation
-------------------

Setting goiardi up to use shovey is pretty straightforward.

* Once goiardi is installed or updated, install serf and run it with
  ``serf agent``. Make sure that the serf agent is using the same name for its
  node name that goiardi is using for its server name.
* Generate an RSA public/private keypair. Goiardi will use this to sign its
  requests to the client, and schob will verify the requests with it.::

      openssl genrsa -out shovey.pem 2048 # generate 2048 bit private key
      openssl rsa -in shovey.pem -pubout -out shovey.key # public key

  Obviously, save these keys.
* If you're using an external service (like vault) to store secrets, please see   :ref:`secrets` for how to set up shovey's signing key with that. 
* Run goiardi like you usually would, but add these options:
  ``--use-serf --use-shovey --sign-priv-key=/path/to/shovey.pem``
* Install serf and schob on a chef node. Ensure that the serf agent on the node
  is using the same name as the chef node. The ``shovey-jobs`` cookbook makes
  installing schob easier, but it's not too hard to do by hand by running
  ``go get github.com/ctdk/schob`` and ``go install github.com/ctdk/schob``.
* If you didn't use the shovey-jobs cookbook, make sure that the public key you
  generated earlier is uploaded to the node somewhere.
* Shovey uses a whitelist to allow jobs to run on nodes. The shovey whitelist is
  a simple JSON hash, with job names as the keys and the commands to run as the
  values. There's a sample whitelist file in the schob repo at
  ``test/whitelist.json``, and the shovey-jobs cookbook will create a whitelist
  file from Chef node attributes using the usual precedence rules. The whitelist
  is drawn from ``node["schob"]["whitelist"]``.
* If you used the shovey-jobs cookbook schob should be running already. If not,
  start it with something like ``schob -VVVV -e http://chef-server.local:4545 -n
  node-name.local -k /path/to/node.key -w /path/to/schob/test/whitelist.json -p
  /path/to/public.key --serf-addr=127.0.0.1:7373``. Within a minute, goiardi
  should be aware that the node is up and ready to accept jobs.

At this point you should be able to submit jobs and have them run. The knife-shove documentation goes into detail on what actions you can take with shovey, but to start try ``knife goiardi job start ls <node name>``. To list jobs, run ``knife goiardi job list``. You can also get information on a shovey job, detailed information of a shovey job's run on one node, cancel jobs, query node status, and stream job output from a node with the knife-shove plugin. See the plugin's documentation for more information.

See the serf docs at http://www.serfdom.io/docs/index.html for more information on setting up serf. One serf option you may want to use, once you're satisfied that shovey is working properly, is to use encryption with your serf cluster.

Shovey In More Detail
---------------------

Every thirty seconds, schob sends a heartbeat back to goiardi over serf to let goiardi know that the node is up. Once a minute, goiardi pulls up a list of nodes that it hasn't seen in the last 10 minutes and marks them as being down. If a node that is down comes back up and sends a heartbeat back to goiardi, it is marked as being up again. The node statuses are tracked over time as well, so a motivated user could track node availability over time.

When a shovey run is submitted, goiardi determines which nodes are to be included in the run, either via the search function or from being listed on the command line. It then sees how many of the nodes are believed to be up, and compares that number with the job's quorum. If there aren't enough nodes up to satisfy the quorum, the job fails.

If the quorum is satisfied, goiardi sends out a serf query with the job's parameters to the nodes that will run the shovey job, signed with the shovey private key. The nodes verify the job's signature and compare the job's command to the whitelist, and if it checks out begin running the job.

As the job runs, schob will stream the command's output back to goiardi. This output can in turn be streamed to the workstation managing the shovey jobs, or viewed at a later time. Meanwhile, schob also watches for the job to complete, receiving a cancellation command from goiardi, or to timeout because it was running too long. Once the job finishes or is cancelled or killed, schob sends a report back to goiardi detailing the job's run on that node.
