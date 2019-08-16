.. _authentication:

Authentication
==============

If goiardi is not running in use-auth mode, it does not actually care about .pem files at all. You still need to have one to keep knife and chef-client happy. It's like chef-zero in that regard.

If goiardi is running in use-auth mode, then proper keys are needed. When goiardi is started, if the chef-webui and chef-validator clients, and the admin user, are not present, it will create new keys in the ``--conf-root`` directory. Use them as you would normally for validating clients, performing tasks with the admin user, or using chef-webui if webui will run in front of goiardi.

In auth mode, goiardi supports versions 1.0, 1.1, 1.2, and 1.3 of the Chef authentication protocol.

*Note:* The admin user, when created on startup, does not have a password. This prevents logging in to the webui with the admin user, so a password will have to be set for admin before doing so.

Fresh start
-----------

If you have not started the server without authentication and a persistent data store configured, just start it with authentication enabled and a conf-root directory. On the first start the admin, chef-webui, chef-validator keys will be saved to the directory given with the conf-root option.

Server saved data without authentication enabled
------------------------------------------------

This means that the clients were created in the database but you don't have private keys for them. You need to start the server with authentication disabled, then use knife to regenerate the admin user's private key with ``knife user reregister admin``. Save this key, and you can now enable authentication and use this key for the admin user. You'll have to recreate the chef-webui and chef-validator keys as well using a similar knife command (``knife client reregister <name>``), but you don't have to have authentication authentication disabled anymore, since you are authenticated with your new primary key.
