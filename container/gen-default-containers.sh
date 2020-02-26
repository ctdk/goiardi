#!/bin/sh

CONTAINERS=$(grep containers ../acl/definitions.go | \
	grep -v "=" | \
	grep -v root | \
	cut -d' ' -f 3 | \
	sort | \
	uniq | \
	sed -e 's/,$//' | \
	awk '{ printf("\"%s\",\n", $0) }')

CNT=$(echo "$CONTAINERS" | wc -l)
# heredoc occurs, ugh
cat <<-EOC
	// generated by containers.go, do not edit
	package container

	import ()

	var DefaultContainers = [$CNT]string{
	$CONTAINERS
	}

EOC
