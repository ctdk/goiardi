.. _berks:

Berks Universe Endpoint
=======================

Starting with version 0.6.1, goiardi supports the berks-api ``/universe`` endpoint. It returns a JSON list of all the cookbooks and their versions that have been uploaded to the server, along with the URL and dependencies of each version. The requester will need to be properly authenticated with the server to use the universe endpoint.

The universe endpoint works with all backends, but with a ridiculous number of cookbooks (like, loading all 6000+ cookbooks in the Chef Supermarket), the Postgres implementation is able to take advantage of some Postgres specific functionality to generate that page significantly faster than the in-mem or MySQL implementations. It's not too bad, but on my laptop at home goiardi could generate /universe against the full 6000+ cookbooks of the supermarket in ~350 milliseconds, while MySQL took about 1 second and in-mem took about 1.2 seconds. Normal functionality is OK, but if you have that many cookbooks and expect to use the universe endpoint often you may wish to consider using Postgres.
