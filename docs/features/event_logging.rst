.. _event_logging:

Event Logging
=============

Goiardi has optional event logging. When enabled with the ``--log-events`` command line option, or with the ``"log-events"`` option in the config file, changes to clients, users, cookbooks, data bags, environments, nodes, and roles will be tracked. The event log can be viewed through the /events API endpoint.

If the ``-K``/``--log-event-keep`` option is set, then once a minute the event log will be automatically purged, leaving that many events in the log. This is particularly recommended when using the event log in in-memory mode.

If the ``--skip-log-extended`` option is set, then the JSON encoded blob of the object being logged will not be stored.

The easiest way to use the event log is with the knife-goiardi-event-log knife plugin. It's available on rubygems, or at github at https://github.com/ctdk/knife-goiardi-event-log.

The event API endpoints work as follows:

* ``GET /events`` - optionally taking ``offset``, ``limit``, ``from``, ``until``, ``object_type``, ``object_name``, and ``doer`` query parameters.

  List the logged events, starting with the most recent. Use the ``offset`` and ``limit`` query parameters to view smaller chunks of the event log at one time. The ``from``, ``until``, ``object_type``, ``object_name``, and ``doer`` query parameters can be used to narrow the results returned further, by time range (for ``from`` and ``until``), the type of object and the name of the object (for ``object_type`` and ``object_name``) and the name of the performer of the action (for ``doer``). These options may be used in singly or in concert.

* ``DELETE /events?purge=1234`` - purge logged events older than the given id from the event log.

* ``GET /events/1234`` - get a single logged event with the given id.

* ``DELETE /events/1234`` - delete a single logged event from the event log.

A user or client must be an administrator account to use the ``/events`` endpoint.

The data returned from the event log should look something like this:

.. code-block:: javascript

    {
      "actor_info": "{\"username\":\"admin\",\"name\":\"admin\",\"email\":\"\",\"admin\":true}\n",
      "actor_type": "user",
      "time": "2014-05-06T07:40:12Z",
      "action": "delete",
      "object_type": "*client.Client",
      "object_name": "pedant_testclient_1399361999-483981000-42305",
      "extended_info": "{\"name\":\"pedant_testclient_1399361999-483981000-42305\",\"node_name\":\"pedant_testclient_1399361999-483981000-42305\",\"json_class\":\"Chef::ApiClient\",\"chef_type\":\"client\",\"validator\":false,\"orgname\":\"default\",\"admin\":true,\"certificate\":\"\"}\n",
      "id": 22
    }
