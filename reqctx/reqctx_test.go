/*
 * Copyright (c) 2013-2026, Jeremy Bingham (<jeremy@goiardi.gl>)
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

package reqctx

import (
	"context"
	"testing"

	"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/util"
)

type fakeActor struct{}

func (fakeActor) GetName() string                                        { return "fake" }
func (fakeActor) IsUser() bool                                         { return true }
func (fakeActor) IsClient() bool                                       { return false }
func (fakeActor) GetId() int64                                         { return 1 }
func (fakeActor) IsAdmin() bool                                       { return true }
func (fakeActor) IsValidator() bool                                   { return false }
func (fakeActor) IsSelf(interface{}) bool                             { return false }
func (fakeActor) PublicKey() string                                  { return "" }
func (fakeActor) SetPublicKey(interface{}) error                      { return nil }
func (fakeActor) CheckPermEdit(map[string]interface{}, string) util.Gerror { return nil }
func (fakeActor) OrgName() string                                     { return "default" }
func (fakeActor) ACLName() string                                     { return "user:fake" }
func (fakeActor) Authz() string                                       { return "" }
func (fakeActor) IsACLRole() bool                                    { return false }

func TestCtxReqUser(t *testing.T) {
	a := actor.Actor(fakeActor{})
	ctx := context.WithValue(context.Background(), OpUserKey, a)
	got, err := CtxReqUser(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}
	if got.GetName() != "fake" {
		t.Errorf("expected fake, got %s", got.GetName())
	}
}

func TestCtxReqUserMissing(t *testing.T) {
	if _, err := CtxReqUser(context.Background()); err == nil {
		t.Error("expected error when opUser is missing")
	}
}

func TestCtxOrg(t *testing.T) {
	org := &organization.Organization{Name: "test"}
	ctx := context.WithValue(context.Background(), OrgKey, org)
	got, err := CtxOrg(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}
	if got.Name != "test" {
		t.Errorf("expected test, got %s", got.Name)
	}
}

func TestCtxOrgMissing(t *testing.T) {
	if _, err := CtxOrg(context.Background()); err == nil {
		t.Error("expected error when org is missing")
	}
}
