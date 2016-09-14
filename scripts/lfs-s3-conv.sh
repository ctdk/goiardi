#!/bin/bash

# Upload your files in the local file store to s3, when switching from using
# said local file store to s3.
#
# Requires s3cmd (http://s3tools.org/s3cmd) to be installed. Also the given 
# bucket needs to have been created and set up already.

while getopts ":a:s:r:b:d:h" opt; do
	case $opt in
		a) AMZ_ACCESS_ID="$OPTARG"
		;;
		s) AMZ_SECRET="$OPTARG"
		;;
		r) REGION="$OPTARG"
		;;
		b) BUCKET="$OPTARG"
		;;
		d) DIR="$OPTARG"
		;;
		h) echo "Usage: -a <AWS access id> -s <AWS secret> -r <region> -b <bucket> -d <local filestore directory>"
		exit
		;;
		\?) echo "invalid option -$OPTARG" >&2
		;;
	esac
done

if ! [ -z ${AMZ_SECRET+x} ]; then
	AMZ_ARGS="--secret_key=$AMZ_SECRET"
fi
if ! [ -z ${AMZ_ACCESS_ID+x} ]; then
	AMZ_ARGS="$AMZ_ARGS --access_key=$AMZ_ACCESS_ID"
fi

for FILE in `ls $DIR`; do
	prefix=`echo $FILE | cut -c 1-2`
	s3cmd --region=$REGION $AMZ_ARGS put $DIR/$FILE s3://${BUCKET}/default/file_store/$prefix/$FILE
done
