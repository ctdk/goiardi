.. _metrics:

Metrics
=======

Starting with goiardi v0.10.4, goiardi can export metrics about itself via statsd. In turn, statsd can feed these metrics into a time series database like graphite. Once in graphite, one could visualize the data with something like `grafana https://grafana.org`_, or set up alerts with that data in `bosun http://bosun.org`_.

At this time, goiardi exports via statsd metrics covering the runtime (memory usage, garbage collection, goroutines), API timing, information about chef-client runs, the number of nodes, and search timing.

The available metrics via statsd currently are:

* ``node.count`` - number of nodes currently in the system
* ``runtime.goroutines`` - number of goroutines running
* ``runtime.memory.allocated`` - allocated memory in bytes
* ``runtime.memory.mallocs`` - number of mallocs
* ``runtime.memory.frees`` - number of times memory's been freed
* ``runtime.memory.heap`` - size of heap memory in bytes
* ``runtime.memory.stack`` - size of stack memory in bytes
* ``runtime.gc.total_pause`` - how many nanoseconds goiardi has paused for garbage collection the whole time the process has been running.
* ``runtime.gc.pause_per_sec`` - pauses per second
* ``runtime.gc.pause_per_tick`` - pauses per interval sending metrics to statsd (currently 10 seconds)
* ``runtime.gc.num_gc`` - number of garbage collections
* ``runtime.gc.gc_per_sec`` - gc per second
* ``runtime.gc.gc_per_tick`` - gc per statsd tick (as above, every 10 secodns)
* ``runtime.gc.pause`` - timing of how long each gc pause lasts
* ``api.timing.%s.%s``, where "``%s.%s``" is the first part of the api endpoint path and the HTTP method (so, for example, a PUT to cookbooks would be ``api.timing.cookbooks.put``) - timing of API endpoint requests
* ``client.run.started`` - Count of started chef-client runs
* ``client.run.success`` - Count of successful chef-client runs
* ``client.run.failure`` - Count of failed chef-client runs
* ``client.run.run_time`` - Timing of how long
* ``client.run.total_resource_count`` - Total resources in a run
* ``client.run.updated_resources`` - Total updated resources in a run
* ``search.in_mem`` - timing of in-memory searches
* ``search.pg`` - timing of Postgres-based searches
