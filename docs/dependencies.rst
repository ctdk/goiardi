.. _dependencies:

Dependencies
============

Goiardi currently has nine dependencies: go-flags, go-cache, go-trie, toml, the mysql driver from go-sql-driver, the postgres driver, logger, go-uuid, and serf.

To install them, run:

.. code-block:: bash

    go get github.com/jessevdk/go-flags
    go get github.com/pmylund/go-cache
    go get github.com/ctdk/go-trie/gtrie
    go get github.com/BurntSushi/toml
    go get github.com/go-sql-driver/mysql
    go get github.com/lib/pq
    go get github.com/ctdk/goas/v2/logger
    go get github.com/codeskyblue/go-uuid
    go get github.com/hashicorp/serf/client

from your ``$GOROOT``, or just use the ``-t`` flag when you go get goiardi.

If you would like to modify the search grammar, you'll need the ``peg`` package. To install that, run:

.. code-block:: bash

   go get github.com/pointlander/peg


In the ``search/`` directory, run ``peg -switch -inline search-parse.peg`` to generate the new grammar. If you don't plan on editing the search grammar, though, you won't need that.
