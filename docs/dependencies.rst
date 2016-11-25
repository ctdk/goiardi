.. _dependencies:

Dependencies
============

As of version 0.11.0, goiardi now includes its dependencies in the ``vendor`` directory. This saves the headache of having to download various sources and possibly finding that they don't work.

If, for whatever reason, you are building goiardi with vendoring disabled, the dependencies will be installed when you ``go get`` it.

If you would like to modify the search grammar, you'll need the ``peg`` package. To install that, run:

.. code-block:: bash

   go get github.com/pointlander/peg


In the ``search/`` directory, run ``peg -switch -inline search-parse.peg`` to generate the new grammar. If you don't plan on editing the search grammar, though, you won't need that.
