.. _upgrading_1_0_0:

Upgrading to 1.0.0
==================

The upgrade process from 0.11.x to 1.0.0 is definitely beefier than previous upgrades, but should not be too painful. The process will be vastly easier for those folks running Postgres; in this case all you should need to do is run ``sqitch deploy`` to apply the various SQL patches, and your existing nodes, clients, cookbooks, etc. will be in the ``default`` organization. Internally, everything was in the ``default`` organization anyway, so the biggest change will be having to update ``knife.rb`` or ``client.rb`` accordingly to refer to the new URL.

Since MySQL has been removed from goiardi as of version 1.0.0, the whole upgrade process is rather trickier. One option is to stay on 0.11.x - it won't get the latest and greatest features, certainly, but the plan for the foreseeable future is for it to still receive updates when needed. Otherwise, MySQL users will have to resort to using the same upgrade process as the in-mem users. Presumably most MySQL users would either switch to Postgres or stay on 0.11.x, but they could move to the in-mem model if they really wanted to for some reason.

< NOTE/TODO: Some/most of these upgrade tasks will be or are documented on their respective pages. They should be described and linked to or partially (or totally) moved to this page. >

< TODO: Postgres upgrade >

< TODO: detail upgrade progress for in-mem/MySQL folks >

< TODO: if needed, knife plugin updates >

< TODO: Moving files in the local file store or s3 >

< TODO: Moving secrets >

< TODO: Updating existing nodes >

< TODO: Updating existing shovey nodes >

< TODO: Updating knife.rb/client.rb >
