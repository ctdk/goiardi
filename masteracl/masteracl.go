/*
 * Copyright (c) 2013-2019, Jeremy Bingham (<jeremy@goiardi.gl>)
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

// Package masteracl is for perm checks outside of the scope of one
// organization.
package masteracl

import (
	"errors"
	"github.com/casbin/casbin"
	"github.com/casbin/casbin/persist"
	"github.com/casbin/casbin/persist/file-adapter"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/util"
	"github.com/tideland/golib/logger"
	"net/http"
	"os"
	"path"
)

const masterPolicyFilename = "master-policy.csv"

// A subset of the Actor interface for checking master perms.
type Actor interface {
	GetName() string
	IsUser() bool
	IsClient() bool
	GetId() int64
}

type MasterACLItem uint8

const (
	Organizations MasterACLItem = iota
	Reindex
)

var aclLookup = map[MasterACLItem]string{
	Organizations: "organizations",
	Reindex: "reindex",
}

// masterACL lets us easily do perm checks that affect goiardi as a whole,
// rather than specific to an organization.
type masterACL struct {
	*casbin.SyncedEnforcer
}

var ClientErr = util.Errorf("clients are ineligible to have permissions to perform this action")

// For now, don't load the master policy file into memory. This may change down
// the road.

func MasterCheckPerm(doer Actor, item MasterACLItem, perm string) (bool, util.Gerror) {
	if doer.IsClient() {
		return false, ClientErr
	}
	masterChecker, err := loadMasterACL()
	if err != nil {
		gerr := util.CastErr(err)
		gerr.SetStatus(http.StatusInternalServerError)
		return false, gerr
	}

	cond := []interface{}{doer.GetName(), aclLookup[item], perm}
	chk := masterChecker.Enforce(cond...)
	return chk, nil
}

func loadMasterACL() (*masterACL, error) {
	m := casbin.NewModel(modelDefinition)
	if !masterPolicyExists() {
		if err := initializeMasterPolicy(); err != nil {
			return nil, err
		}
	}
	adp, err := loadMasterPolicyAdapter() 
	if err != nil {
		return nil, err
	}
	e := casbin.NewSyncedEnforcer(m, adp, config.Config.PolicyLogging)
	mc := &masterACL{e}
	return mc, nil
}

func getMasterPolicyFile() string {
	return path.Join(config.Config.PolicyRoot, masterPolicyFilename)
}

func masterPolicyExists() bool {
	_, err := os.Stat(getMasterPolicyFile())
	return !os.IsNotExist(err) // bit heavy handed, but eh
}

func loadMasterPolicyAdapter() (persist.Adapter, error) {
	if !masterPolicyExists() {
		err := errors.New("Cannot load master policy file: file does not exist.")
		return nil, err
	}
	adp := fileadapter.NewAdapter(getMasterPolicyFile())
	return adp, nil
}

func initializeMasterPolicy() error {
	logger.Debugf("initializing master policy")
	if masterPolicyExists() {
		err := errors.New("master policy file already exists, cannot initialize!")
		return err
	}
	masterPol := getMasterPolicyFile()
	p, err := os.OpenFile(masterPol, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer p.Close()
	if _, err = p.WriteString(masterPolicySkel); err != nil {
		return err
	}
	return nil
}
