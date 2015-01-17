.. _authentication:

Authentication
==============

If goiardi is not running in use-auth mode, it does not actually care about .pem files at all. You still need to have one to keep knife and chef-client happy. It's like chef-zero in that regard.

If goiardi is running in use-auth mode, then proper keys are needed. When goiardi is started, if the chef-webui and chef-validator clients, and the admin user, are not present, it will create new keys in the ``--conf-root`` directory. Use them as you would normally for validating clients, performing tasks with the admin user, or using chef-webui if webui will run in front of goiardi.

In auth mode, goiardi supports versions 1.0, 1.1, and 1.2 of the Chef authentication protocol.

*Note:* The admin user, when created on startup, does not have a password. This prevents logging in to the webui with the admin user, so a password will have to be set for admin before doing so.

Fresh start
-----------

If you have not started the server without authentication and a persistent data store configured, just start it with authentication enabled and a conf-root directory. On the first start the admin, chef-webui, chef-validator keys will be saved to the directory given as conf-root. The admin client is not created yet, but you can use the admin.pem as 'admin' client with knife to create it (knife client create admin). This will give you a new private key for the user, you'll have to replace the previous with it.

Server saved data without authentication enabled
------------------------------------------------

This means that the clients were created in the database but no private keys were generated for them. You need to start the server with authentication disabled, then knife client create admin and save the private key. Now you can enable authentication and use this key for the admin client. You'll have to recreate the chef-webui and chef-validator clients too using the same knife command, but you don't have to disable authentication anymore, since you are authenticated with your new primary key.
