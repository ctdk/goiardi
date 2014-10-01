.. _data:

Import and Export of Data
=========================

Goiardi can now import and export its data in a JSON file. This can help both when upgrading, when the on-disk data format changes between releases, and to convert your goiardi installation from in-memory to MySQL (or vice versa). The JSON file has a version number set (currently 1.0), so that in the future if there is some sort of incompatible change to the JSON file format the importer will be able to handle it.

Before importing data, you should back up any existing data and index files (and take a snapshot of the SQL db, if applicable) if there's any reason you might want it around later. After exporting, you may wish to hold on to the old installation data until you're satisfied that the import went well.

Remember that the JSON export file contains the client and user public keys (which for the purposes of goiardi and chef are private) and the user hashed passwords and password salts. The export file should be guarded closely.

The ``-x/--export`` and ``-m/--import`` flags control importing and exporting data. To export data, stop goiardi, then run it again with the same options as before but adding ``-x <filename>`` to the command. This will export all the data to the given filename, and goiardi will exit.

Importing is ever so slightly trickier. You should remove any existing data store and index files, and if using an SQL database use sqitch to revert and deploy all of the SQL files to set up a completely clean schema for goiardi. Then run goiardi with the new options like you normally would, but add ``-m <filename>``. Goiardi will run, import the new data, and exit. Assuming it went well, the data will be all imported. The export dump does not contain the user and client .pem files, so those will need to be saved and moved as needed.

Theoretically a properly crafted export file could be used to do bulk loading of data into goiardi, thus goiardi does not wipe out the existing data on its own but rather leaves that task to the administrator. This functionality is merely theoretical and completely untested. If you try it, you should back your data up first.
