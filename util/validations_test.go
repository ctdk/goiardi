/*
 * Copyright (c) 2013-2017, Jeremy Bingham (<jeremy@goiardi.gl>)
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

package util

import (
	"testing"
)

func TestValidateNameFull(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{"pedant_node", true},
		{"PEDANT_NODE", true},
		{"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_", true},
		{"foo-bar.baz", true},
		{"this+ is bad!!!", false},
		{"I-do-not-like!!!", false},
		{"node@127.0.0.1", false},
		{"", true}, // empty string passes (no bad chars)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateName(tt.name)
			if got != tt.valid {
				t.Errorf("ValidateName(%q) = %v, want %v", tt.name, got, tt.valid)
			}
		})
	}
}

func TestValidateUserNameFull(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{"foo", true},
		{"foo-bar.baz", true},
		{"foo_bar", true},
		{"USERNAME", false},
		{"USER", false},
		{"HasUpper", false},
		{"foo!bar", false},
		{"foo bar", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateUserName(tt.name)
			if got != tt.valid {
				t.Errorf("ValidateUserName(%q) = %v, want %v", tt.name, got, tt.valid)
			}
		})
	}
}

func TestValidateDBagNameFull(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{"pedant", true},
		{"pedant-bag", true},
		{"pedant_bag", true},
		{"pedant_bag-foo", true},
		{"1234567890", true},
		{"pedant99", true},
		{"pedant:with:colons", true},
		{"pedant.with.dots", true},
		{"pedant_badName!!$$$$_oh_very+bad", false},
		{"pedant-does-not-like-punctuation!!!!", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateDBagName(tt.name)
			if got != tt.valid {
				t.Errorf("ValidateDBagName(%q) = %v, want %v", tt.name, got, tt.valid)
			}
		})
	}
}

func TestValidateEnvNameFull(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{"pedant_environment", true},
		{"ABCDEFGHIJKLMNOPQRSTUVWXYZ_abcdefghijklmnopqrstuvwxyz-0123456789", true},
		{"_default", true},
		{"abc!123", false},
		{"abc 123", false},
		{"大爆発", false},
		{"abc.def", false},
		{"abc:def", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateEnvName(tt.name)
			if got != tt.valid {
				t.Errorf("ValidateEnvName(%q) = %v, want %v", tt.name, got, tt.valid)
			}
		})
	}
}

func TestValidateAsString(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		valid bool
	}{
		{"string", "hello", true},
		{"nil", nil, false},
		{"int", 42, false},
		{"bool", true, false},
		{"array", []string{}, false},
		{"map", map[string]string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateAsString(tt.input)
			if tt.valid && err != nil {
				t.Errorf("ValidateAsString(%v) should have succeeded, got: %v", tt.input, err)
			}
			if !tt.valid && err == nil {
				t.Errorf("ValidateAsString(%v) should have failed", tt.input)
			}
		})
	}
}

func TestValidateAsBool(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		valid bool
	}{
		{"true", true, true},
		{"false", false, true},
		{"string", "true", false},
		{"int", 1, false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateAsBool(tt.input)
			if tt.valid && err != nil {
				t.Errorf("ValidateAsBool(%v) should have succeeded, got: %v", tt.input, err)
			}
			if !tt.valid && err == nil {
				t.Errorf("ValidateAsBool(%v) should have failed", tt.input)
			}
		})
	}
}

func TestValidateAsFieldString(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		valid bool
	}{
		{"string", "hello", true},
		{"nil", nil, false},
		{"int", 42, false},
		{"bool", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateAsFieldString(tt.input)
			if tt.valid && err != nil {
				t.Errorf("ValidateAsFieldString(%v) should have succeeded, got: %v", tt.input, err)
			}
			if !tt.valid && err == nil {
				t.Errorf("ValidateAsFieldString(%v) should have failed", tt.input)
			}
		})
	}
}

func TestValidateAsVersionFull(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		valid   bool
		expect  string
	}{
		{"1.0.0", "1.0.0", true, "1.0.0"},
		{"1.0", "1.0", true, "1.0"},
		{"0.0", "0.0", false, ""},
		{"1", "1", false, ""},
		{"foo", "foo", false, ""},
		{"nil", nil, true, "0.0.0"},
		{"int", 42, false, ""},
		{"1.2.3", "1.2.3", true, "1.2.3"},
		{"1.2.2147483647", "1.2.2147483647", true, "1.2.2147483647"},
		{"1.2.2147483669", "1.2.2147483669", true, "1.2.2147483669"},
		{"1.2.9223372036854775849", "1.2.9223372036854775849", false, ""},
		{"1.-2.3", "1.-2.3", false, ""},
		{"1.2.20130730201745", "1.2.20130730201745", true, "1.2.20130730201745"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateAsVersion(tt.input)
			if tt.valid {
				if err != nil {
					t.Errorf("ValidateAsVersion(%v) should have succeeded, got: %v", tt.input, err)
				}
				if got != tt.expect {
					t.Errorf("ValidateAsVersion(%v) = %q, want %q", tt.input, got, tt.expect)
				}
			} else {
				if err == nil {
					t.Errorf("ValidateAsVersion(%v) should have failed, got %q", tt.input, got)
				}
			}
		})
	}
}

func TestValidateAttributes(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		input interface{}
		valid bool
	}{
		{"valid_map", "normal", map[string]interface{}{"key": "value"}, true},
		{"nil", "normal", nil, true},
		{"string", "normal", "string", false},
		{"int", "normal", 123, false},
		{"array", "normal", []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateAttributes(tt.key, tt.input)
			if tt.valid && err != nil {
				t.Errorf("ValidateAttributes(%q, %v) should have succeeded, got: %v", tt.key, tt.input, err)
			}
			if !tt.valid && err == nil {
				t.Errorf("ValidateAttributes(%q, %v) should have failed", tt.key, tt.input)
			}
		})
	}
}

func TestValidateAsConstraint(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		valid bool
	}{
		{">= 1.0.0", ">= 1.0.0", true},
		{"<= 1.0.0", "<= 1.0.0", true},
		{"> 1.0.0", "> 1.0.0", true},
		{"< 1.0.0", "< 1.0.0", true},
		{"= 1.0.0", "= 1.0.0", true},
		{"~> 1.0.0", "~> 1.0.0", true},
		{"1.0.0", "1.0.0", false},
		{">= 1.0", ">= 1.0", true},
		{">=1.0.0", ">=1.0.0", false},
		{" >= 1.0.0", " >= 1.0.0", false},
		{">=  1.0.0", ">=  1.0.0", false},
		{">= 1.a.b", ">= 1.a.b", false},
		{">= 1.0rc1", ">= 1.0rc1", false},
		{">= 1.0.0.0", ">= 1.0.0.0", false},
		{">= 1,0,0", ">= 1,0,0", false},
		{"nil", nil, false},
		{"int", 42, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateAsConstraint(tt.input)
			if tt.valid {
				if err != nil {
					t.Errorf("ValidateAsConstraint(%v) should have succeeded, got: %v", tt.input, err)
				}
				if !got {
					t.Errorf("ValidateAsConstraint(%v) returned false", tt.input)
				}
			} else {
				if err == nil {
					t.Errorf("ValidateAsConstraint(%v) should have failed", tt.input)
				}
			}
		})
	}
}

func TestValidateRunList(t *testing.T) {
	tests := []struct {
		name   string
		input  interface{}
		valid  bool
		expect []string
	}{
		{"bare_recipes", []string{"foo", "bar"}, true, []string{"recipe[foo]", "recipe[bar]"}},
		{"recipe_format", []string{"recipe[foo]", "recipe[bar::baz]"}, true, []string{"recipe[foo]", "recipe[bar::baz]"}},
		{"role_format", []string{"role[prod]"}, true, []string{"role[prod]"}},
		{"mixed", []string{"foo", "recipe[web]", "role[prod]"}, true, []string{"recipe[foo]", "recipe[web]", "role[prod]"}},
		{"with_version", []string{"bar::baz@1.0.0"}, true, []string{"recipe[bar::baz@1.0.0]"}},
		{"duplicates", []string{"webserver", "recipe[webserver]", "role[prod]", "role[prod]"}, true, []string{"recipe[webserver]", "role[prod]"}},
		{"empty", []string{}, true, []string{}},
		{"nil", nil, true, nil},
		{"int_array", []interface{}{1, 2, 3}, false, nil},
		{"nested_array", []interface{}{[]interface{}{}}, false, nil},
		{"string", "string", false, nil},
		{"int", 42, false, nil},
		{"map", map[string]interface{}{}, false, nil},
		{"invalid_chars", []string{"this+ is bad!!!"}, false, nil},
		{"invalid_chars2", []string{"I-do-not-like!!!"}, false, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateRunList(tt.input)
			if tt.valid {
				if err != nil {
					t.Errorf("ValidateRunList(%v) should have succeeded, got: %v", tt.input, err)
				}
				if len(got) != len(tt.expect) {
					t.Errorf("ValidateRunList(%v) = %v (len=%d), want %v (len=%d)", tt.input, got, len(got), tt.expect, len(tt.expect))
				} else {
					for i := range got {
						if got[i] != tt.expect[i] {
							t.Errorf("ValidateRunList(%v)[%d] = %q, want %q", tt.input, i, got[i], tt.expect[i])
						}
					}
				}
			} else {
				if err == nil {
					t.Errorf("ValidateRunList(%v) should have failed, got %v", tt.input, got)
				}
			}
		})
	}
}

func TestValidateNumVersions(t *testing.T) {
	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{"all", "all", true},
		{"number", "5", true},
		{"zero", "0", true},
		{"empty", "", false},
		{"negative", "-1", false},
		{"invalid", "foo", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNumVersions(tt.input)
			if tt.valid && err != nil {
				t.Errorf("ValidateNumVersions(%q) should have succeeded, got: %v", tt.input, err)
			}
			if !tt.valid && err == nil {
				t.Errorf("ValidateNumVersions(%q) should have failed", tt.input)
			}
		})
	}
}

func TestCheckAdminPlusValidator(t *testing.T) {
	tests := []struct {
		name  string
		input map[string]interface{}
		valid bool
	}{
		{"admin_only", map[string]interface{}{"admin": true}, true},
		{"validator_only", map[string]interface{}{"validator": true}, true},
		{"neither", map[string]interface{}{}, true},
		{"both", map[string]interface{}{"admin": true, "validator": true}, false},
		{"admin_false_validator_true", map[string]interface{}{"admin": false, "validator": true}, true},
		{"admin_true_validator_false", map[string]interface{}{"admin": true, "validator": false}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckAdminPlusValidator(tt.input)
			if tt.valid && err != nil {
				t.Errorf("CheckAdminPlusValidator(%v) should have succeeded, got: %v", tt.input, err)
			}
			if !tt.valid && err == nil {
				t.Errorf("CheckAdminPlusValidator(%v) should have failed", tt.input)
			}
		})
	}
}
