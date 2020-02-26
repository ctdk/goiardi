package main

import (
	"io/ioutil"
	"log"
	"os"
	"strings"
)

const (
	defLen = 7
	kindField = 3
	subKindField = 2
)

func main() {
	// get the default csv definitions from the first arg. Not going to use
	// flags or some such, so look up how to do that.
	var defPath string // define for real of course

	containers := make(map[string]bool)

	rawDefs, err := io.ReadFile(defPath)
	if err != nil {
		log.Fatal(err)
	}

	defLines := strings.Split(string(rawDefs), "\n")
	for _, l := range defLines {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}

		f := strings.Split(l, ", ")
		if len(f) != defLen {
			log.Fatalf("Def field length mismatch: should have been %d, but got %d instead. Offending definition: '%s'", defLen, len(f), l)
		}

		kind := f[kindField]
		subKind := f[subKindField]

		// all containers are located before the groups. Keep it that
		// way, yo.
		if kind == "groups" {
			break
		}

		// take the easy way out. Not checking if it's already been set
		// since that might actually slow it down slightly
		containers[subKind] = true
	}

	cnt := len(containers)
}
