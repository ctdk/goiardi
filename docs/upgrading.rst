.. _upgrading:

Upgrading
============

Upgrading goiardi is generally a straightforward process. Usually all you should need to do is get the new sources and rebuild (using the ``-u`` flag when running ``go get`` to update goiardi is a good idea to ensure the dependencies are up to date), or download the appropriate new binary. However, sometimes a little more work is involved. Check the release notes for the new release in question for any extra steps that may need to be done. If you're running one of the SQL backends, you may need to apply database patches (either with sqitch or by hand), and in-memory mode especially may require using the data import/export functionality to dump and load your chef data between upgrades if the binary save file compatibility breaks between releases. However, while it should not happen often, occasionally more serious preparation will be needed before upgrading. It won't happen without a good reason, and the needed steps will be clearly outlined to make the process as painless as possible.

Upgrading to 1.0.0 from earlier versions
----------------------------------------

The upgrade process from 0.11.x to 1.0.0 is definitely beefier than previous upgrades, but again should not be too painful. The process will be vastly easier for those folks running Postgres; in this case all you should need to do is run ``sqitch deploy`` to apply the various SQL patches, and your existing nodes, clients, cookbooks, etc. will be in the ``default`` organization. Internally, everything was in the ``default`` organization anyway, so the biggest change will be having to update ``knife.rb`` or ``client.rb`` accordingly to refer to the new URL.

Since MySQL has been removed from goiardi as of version 1.0.0, the whole upgrade process is rather trickier. One option is to stay on 0.11.x - it won't get the latest and greatest features, certainly, but the plan for the foreseeable future is for it to still receive updates when needed. Otherwise, MySQL users will have to resort to using the same upgrade process as the in-mem users. Presumably most MySQL users would either switch to Postgres or stay on 0.11.x, but they could move to the in-mem model if they really wanted to for some reason.

< TODO: detail upgrade progress for in-mem/MySQL folks >

Upgrading to the latest 0.11.x from earlier versions
----------------------------------------------------

As a special note, if you are upgrading from any release prior to 0.6.1-pre1 to 0.7.0 and are using one of the SQL backends, the upgrade is one of the special cases. Between those releases the way the complex data structures associated with cookbook versions, nodes, etc. changed from using gob encoding to json encoding. It turns out that while gob encoding is indeed faster than json (and was in all the tests I had thrown at it) in the usual case, in this case json is actually significantly faster, at least once there are a few thousand coobkooks in the database. In-memory datastore (including file-backed in-memory datastore) users are advised to dump and reload their data between upgrading from <= 0.6.1-pre1 and 0.7.0, but people using either MySQL or Postgres *have* to do these things:

* Export their goiardi server's data with the ``-x`` flag.
* Either revert all changes to the db with sqitch, then redeploy, or drop the database manually and recreate it from either the sqitch patches or the full table dump of the release (provided starting with 0.7.0)
* Reload the goiardi data with the ``-m`` flag.

It's a fairly quick process (a goiardi dump with the ``-x`` flag took 15 minutes or so to load with over 6200 cookbooks) at least, but if you don't do it very little of your goiardi environment will work correctly. The above steps will take care of it.

If you're upgrading from version 0.8.2 (or before) to version 0.9.0, you will need to remove the search index save file before starting the new goiardi for the first time. After that's been done, run ``knife index rebuild`` to rebuild the search index.

One thing that's not always necessary, but is often good practice when running the in-memory trie based index, is to rebuild the search index with ``knife index rebuild``. If anything has changed with the search index between releases, even if it's a minor one not worth making a new minor point release, rebuilding can help avoid any potential gotchas. Postgres index users should only need to reindex when it's specifically noted that they should, although rebuilding won't hurt in that case.


