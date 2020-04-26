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

package policies

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/indexer"
	"github.com/ctdk/goiardi/organization"
	"testing"
)

var revJSON map[string]interface{}

// Borrowed wholesale from the Chef Server API docs.
var policyName = "aar"
var revRevId = "37f9b658cdd1d9319bac8920581723efcc2014304b5f3827ee0779e10ffbdcc9"
var revJSONBase = `
{
  "revision_id": "37f9b658cdd1d9319bac8920581723efcc2014304b5f3827ee0779e10ffbdcc9",
  "name": "aar",
  "run_list": [
    "recipe[aar::default]"
  ],
  "cookbook_locks": {
    "aar": {
      "version": "0.1.0",
      "identifier": "29648fe36333f573d5fe038a53256e23733618aa",
      "dotted_decimal_identifier": "11651043203167221.32604909279531813.121098535835818",
      "source": "cookbooks/aar",
      "cache_key": null,
      "scm_info": {
        "scm": "git",
        "remote": null,
        "revision": "a2c8cbb24a08625921d753cde36e8320465116c3",
        "working_tree_clean": false,
        "published": false,
        "synchronized_remote_branches": [

        ]
      },
      "source_options": {
        "path": "cookbooks/aar"
      }
    },
    "apt": {
      "version": "2.7.0",
      "identifier": "16c57abbd056543f7d5a15dabbb03261024a9c5e",
      "dotted_decimal_identifier": "6409580415309396.17870749399956400.55392231660638",
      "cache_key": "apt-2.7.0-supermarket.chef.io",
      "origin": "https://supermarket.chef.io/api/v1/cookbooks/apt/versions/2.7.0/download",
      "source_options": {
        "artifactserver": "https://supermarket.chef.io/api/v1/cookbooks/apt/versions/2.7.0/download",
        "version": "2.7.0"
      }
    }
  },
  "default_attributes": {

  },
  "override_attributes": {

  },
  "solution_dependencies": {
    "Policyfile": [
      [
        "aar",
        ">= 0.0.0"
      ],
      [
        "apt",
        "= 2.7.0"
      ]
    ],
    "dependencies": {
      "apt (2.7.0)": [

      ],
      "aar (0.1.0)": [
        [
          "apt",
          ">= 0.0.0"
        ]
      ]
    }
  }
}
`

func init() {
	indexer.Initialize(config.Config, indexer.DefaultDummyOrg)

	gob.Register(new(organization.Organization))
	gob.Register(new(Policy))
	gob.Register(new(PolicyRevision))
	gob.Register(new(PolicyGroup))

	revJSON = makeJSONObj(revJSONBase)
}

func makeJSONObj(raw string) map[string]interface{} {
	buf := bytes.NewBufferString(raw)
	obj := make(map[string]interface{})
	dec := json.NewDecoder(buf)
	dec.UseNumber()
	if err := dec.Decode(&obj); err != nil {
		panic(err)
	}
	orl, _ := obj["run_list"].([]interface{})

	rl := make([]string, len(orl))
	for i, v := range orl {
		rl[i], _ = v.(string)
	}
	obj["run_list"] = rl
	return obj
}

func TestNewPolicyRevision(t *testing.T) {
	polName := "aar"
	p, _ := New(org, polName)
	p.Save()

	pr, err := p.NewPolicyRevisionFromJSON(revJSON)
	if err != nil {
		t.Error(err)
	}
	if pr.PolicyName() != policyName {
		t.Errorf("policy revision name: expected %s, got %s", policyName, pr.PolicyName())
	}
	if pr.RevisionId != revRevId {
		t.Errorf("policy revision id: expected '%s', got '%s'", revRevId, pr.RevisionId)
	}

	if err = pr.Save(); err != nil {
		t.Errorf("saving policy revision failed: %v", err)
	}

	pr2, err := p.GetPolicyRevision(revRevId)
	if err != nil {
		t.Errorf("GetPolicyRevision failed: %v", err)
	}
	if pr2 == nil {
		t.Error("GetPolicyRevision returned a nil object")
	}
	if pr.PolicyName() != pr2.PolicyName() {
		t.Errorf("somehow got mismatched policy names: '%v' vs. '%v'", pr.PolicyName(), pr2.PolicyName())
	}
	if pr.RevisionId != pr2.RevisionId {
		t.Errorf("revision ids didn't match: '%v' vs. '%v'", pr.RevisionId, pr2.RevisionId)
	}
}
