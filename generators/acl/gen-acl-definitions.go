// build +ignore

/*
 * Copyright (c) 2013-2020, Jeremy Bingham (<jeremy@goiardi.gl>)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

const aclDefTemplate = `// Code generated by generators/acl/gen-acl-definitions.go, DO NOT EDIT.

/*
 * Copyright (c) 2013-2020, Jeremy Bingham (<jeremy@goiardi.gl>)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package acl

import ()

// **NB:** This autogenerated file does contain some comments taken from the
// hand-crafted version of this file that preceded this one. They are only for
// reference, because sometimes it's extremely useful to be able to look up what
// on earth a given ACL policy field actually means.

// Define the casbin RBAC model and the skeletal $$default$$ policy.

const modelDefinition = %s

// NOTE: Postgres implementations of this may require some mild heroics to get
// this to a form suitable to put in the DB. We'll see what ends up happening.

// group, subkind, kind, name, perm, effect

const defaultPolicySkel = %s
`

const argLen = 7

func main() {
	if len(os.Args) < argLen {
		log.Fatal("not enough arguments!")
	}

	var modelBaseFile string
	var policyBaseFile string
	var outPath string

	// don't feel like the headache of properly using 'flags'
	for i := 1; i < argLen; i++ {
		switch os.Args[i] {
		case "-m":
			modelBaseFile = os.Args[i+1]
		case "-p":
			policyBaseFile = os.Args[i+1]
		case "-o":
			outPath = os.Args[i+1]
		}
	}

	mRaw, err := ioutil.ReadFile(modelBaseFile)
	if err != nil {
		log.Fatal(err)
	}
	model := string(mRaw)

	pRaw, err := ioutil.ReadFile(policyBaseFile)
	if err != nil {
		log.Fatal(err)
	}
	policy := string(pRaw)

	// grrrrrrr
	model = fmt.Sprintf("`%s\n`", strings.TrimSpace(model))
	policy = fmt.Sprintf("`%s\n`", strings.TrimSpace(policy))

	outFile, err := os.Create(outPath)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Fprintf(outFile, aclDefTemplate, model, policy)
}