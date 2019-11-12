.. _goiardi_knife_plugins:

Goiardi ``knife`` Plugins
=========================

There is a small collection of knife plugins for goiardi. None of them are strictly required for the general use case, but ``knife-goiardi-index`` is *strongly* suggested for administrators, and the other three are needed for some specific functionality.

The Plugins
-----------

* ``knife-goiardi-index`` - rebuild search indexes. This restores the ``knife index rebuild`` functionality from older versions of knife.
* ``knife-goiardi-event-log`` - view events in the goiardi event log.
* ``knife-goiardi-reporting`` - view reports about chef-client runs.
* ``knife-shove`` - the shovey plugin. This is used to issue jobs, cancel them, view lists and statuses of those jobs, and so on.

Documentation for these plugins can be found in their respective repositories.

Requirements
------------

### knife-goiardi-index

None. This plugin is recommended if you're using a version of knife that has removed the ``index`` command, though.

### knife-goiardi-event-log

Event logging must be enabled for this to be useful in any way. See < LINK TO EVENT LOG > for more information.

### knife-goiardi-reporting

Similarly, reporting must be enabled and the clients need to return that information to the server for this to be useful. See < LINK TO REPORTING > for more.

### knife-shove

Shovey must be enabled and configured, and there need to be nodes running the schob daemon. The < LINK TO SHOVEY DOCS > has all the information one would need to set that up.
