.. goiardi documentation master file, created by
   sphinx-quickstart on Wed Oct  1 04:21:38 2014.
   You can adapt this file completely to your liking, but it should at least
   contain the root `toctree` directive.

Welcome to goiardi's documentation!
===================================

Goiardi is an implementation of the Chef server (now Chef Infrastructure) (http://www.chef.io) written in Go. It can either run entirely in memory with the option to save and load the in-memory data and search indexes to and from disk, drawing inspiration from  chef-zero, or it can use PostgreSQL as its storage backend. Cookbooks can either be stored locally, or optionally in Amazon S3 (or a compatible service).

Like all software, it is a work in progress. Goiardi now, though, should have all the functionality of older versions of Chef Server 12, plus some extras like reporting, event logging, and a Chef Push-like feature called "shovey". When used, knife works, and chef-client runs complete successfully. Almost all chef-pendant tests successfully successfully  run, with a few disagreements about error messages that don't impact the clients. It does pretty well against the official chef-pedant, but because goiardi handles some authentication matters a little differently than the official chef-server, there are also forks of chef-pedant and oc-chef-pedant located at https://github.com/ctdk/chef-pedant and https://github.com/ctdk/oc-chef-pedant that are more custom tailored to goiardi.

Many go tests are present as well in different goiardi subdirectories.

The goiardi manual is licensed under a Creative Commons Attribution 4.0 License (http://creativecommons.org/licenses/by/4.0/).

.. toctree::
   :maxdepth: 3

   dependencies
   installation
   upgrading
   platforms
   features/authentication
   features/persistence
   features/data
   features/search
   features/event_logging
   features/reporting
   features/berks
   features/serf_and_shovey
   features/shovey_api
   features/logging
   features/webui
   features/metrics
   features/s3
   features/secrets
   changelog

Indices and tables
==================

* :ref:`genindex`
* :ref:`modindex`
* :ref:`search`

