.. _secrets:

Secret Handling
===============

Starting with version 0.11.1, goiardi can use external services to store secrets like public keys, the signing key for shovey, and user password hashes. As of this writing, only `Hashicorp's vault <https://www.vaultproject.io/>`_ is supported. This is very new functionality, so be aware.

Configuration
-------------

The relevant options for secret configuration on goiardi's end are:

* ``--use-external-secrets``: Turns on using an external secret store.

* ``--vault-addr=<address>``: Address of vault server. Defaults to the value the ``VAULT_ADDR`` environment variable, but can be specified here. Optional.
* ``--vault-shovey-key=<path>``: Optional path for where shovey's signing key will be stored in vault. Defaults to "keys/shovey/signing". Only meaningful, unsurprisingly, if shovey is enabled.

Each of the above command-line flags may also be set in the configuration file, with the ``--`` removed.

Additionally, the ``VAULT_TOKEN`` environment variable needs to be set. This can either be set in the configuration file in the ``env-vars`` stanza in the configuration file, or exported to goiardi in one of the many other ways that's possible.

To set up vault itself, see the `intro <https://www.vaultproject.io/intro/index.html>`_ and the `general documentation <https://www.vaultproject.io/docs/index.html>`_ for that program. For goiardi to work right with vault, there will need to be a backend mounted with ``-path=keys`` before goiardi is started.

Populating
----------

A new goiardi installation won't need to do anything special to use vault for secrets - assuming everything's set up properly, new clients and users will work as expected.

Existing goiardi installations will need to transfer their various secrets into vault. A persistent but not DB backed goiardi installation will need to export and import all of goiardi's data. With MySQL or Postgres, it's much simpler.

For each secret, get the key or password hash from the database for each object and make a JSON file like this: ::

        {
                "secretType": "secret-data\nwith\nescaped\nnew\nlines\nif-any"
        }

(Once everything looks good with the secrets being stored in vault, those columns in the database should be cleared.)

The "secretType" is "pubKey" for public keys, "passwd" for password hashes, and "RSAKey" for the shovey signing key.

Optionally, you can add a ``ttl`` (with values like "60s", "30m", etc) field to that JSON, so that goiardi will refetch the secret after that much time has passed.

Now this JSON needs to be written to the vault. For client and user public keys, the path is "keys/clients/<name>" for clients and "keys/users/<name>" for users. User password hashes are "keys/passwd/users/<name>". The shovey signing key is more flexible, but defaults to "keys/shovey/signing". If you save the shovey key to some other path, set ``--vault-shovey-key`` appropriately.
