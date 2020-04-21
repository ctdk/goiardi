.. _upgrading_1_0_0:

Upgrading to 1.0.0
==================

The upgrade process from 0.11.x to 1.0.0 is definitely beefier than previous upgrades, but should not be too painful. The process will be vastly easier for those folks running Postgres; in this case all you should need to do is run ``sqitch deploy`` to apply the various SQL patches, and your existing nodes, clients, cookbooks, etc. will be in the ``default`` organization. Internally, everything was in the ``default`` organization anyway, so the biggest change will be having to update ``knife.rb`` or ``client.rb`` accordingly to refer to the new URL.

Since MySQL has been removed from goiardi as of version 1.0.0, the whole upgrade process is rather trickier. One option is to stay on 0.11.x - it won't get the latest and greatest features, certainly, but the plan for the foreseeable future is for it to still receive updates when needed. Otherwise, MySQL users will have to resort to using the same upgrade process as the in-mem users. Presumably most MySQL users would either switch to Postgres or stay on 0.11.x, but they could move to the in-mem model if they really wanted to for some reason.

< NOTE/TODO: Some/most of these upgrade tasks will be or are documented on their respective pages. They should be described and linked to or partially (or totally) moved to this page. >

< TODO: Postgres upgrade >

< TODO: detail upgrade progress for in-mem/MySQL folks >

< TODO: if needed, knife plugin updates >

Moving files for the upgrade process
------------------------------------

Rearranging the file storage as part of the upgrade is pretty straightforward; all you need to do is move the files from the filestore root into a subdirectory named ``default`` under that filestore root. There's an awful lot of leeway in how you do it, but here are a couple of suggestions.

### Local file storage

Assume for this exercise that the local filestore is at ``/var/lib/goiardi/lfs``, and the destination is at ``/var/lib/goiardi/lfs/default``. The destination directory will need to be created first if it doesn't already exist.

Run this command to copy the filestore contents to the new organization specific location:

``$ find /var/lib/goiardi/lfs -type f -d 1 -exec cp -p '{}' /var/lib/goiardi/lfs/default \;``

This way is a bit safer, since in case something goes terribly wrong you still have the original files easily available. They can be deleted later at your leisure. If you're feeling confident, replace ``cp -p`` with ``mv``.

### S3 file storage

Assume for this exercise that the filestore is at ``s3://mah-bukkit``, and you're moving it to ``s3://mah-bukkit/default``. This requires installing ``aws-cli`` and a properly configured AWS account.

Running this command to copy the files to the new location:

``aws s3 cp s3://mah-bukkit s3://mah-bukkit/default --recursive --exclude="default"``

Again, if you feel confident and don't feel like having to go back and delete the original files, replace ``cp`` above with ``mv``.

< TODO: Moving secrets >

< TODO: Updating existing nodes >

< TODO: Updating existing shovey nodes >

< TODO: Updating knife.rb/client.rb >
