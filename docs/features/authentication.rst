.. _berks:authentication

Authentication
==============

Fresh start
-----------

If you have not started the server without authentication and a persistent data store configured, just start it with authentication enabled and a conf-root directory. On the first start the admin, chef-webui, chef-validator keys will be saved to the directory given as conf-root. The admin client is not created yet, but you can use the admin.pem as 'admin' client with knife to create it (knife client create admin). This will give you a new private key for the user, you'll have to replace the previous with it.

Server saved data without authentication enabled
------------------------------------------------

This means that the clients were created in the database but no private keys were generated for them. You need to start the server with authentication disabled, then knife client create admin and save the private key. Now you can enable authentication and use this key for the admin client. You'll have to recreate the chef-webui and chef-validator clients too using the same knife command, but you don't have to disable authentication anymore, since you are authenticated with your new primary key.
