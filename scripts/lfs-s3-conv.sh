#!/bin/bash

# Upload your files in the local file store to s3, when switching from using
# said local file store to s3.
#
# Requires aws-cli (LINK) to be installed. Also the given bucket needs to have
# been created and set up already.

while getopts ":a:s:r:b:d:h" opt; do
	case $opt in
		b) BUCKET="$OPTARG"
		;;
		d) DIR="$OPTARG"
		;;
		h) echo "Usage: -b <bucket> -d <local filestore directory>"
		exit
		;;
		\?) echo "invalid option -$OPTARG" >&2
		;;
	esac
done

for ORGDIR in $DIR/*; do
	ORGNAME=$(basename $ORGDIR)
	pushd $ORGDIR > /dev/null
	for FILE in *; do
		prefix=`echo $FILE | cut -c 1-2`
		aws s3 put $ORGDIR/$FILE s3://${BUCKET}/${ORGNAME}/file_store/${prefix}/$FILE
	done
	popd > /dev/null
done
