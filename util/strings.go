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
	"fmt"
	"github.com/ctdk/goiardi/gerror"
	"golang.org/x/exp/utf8string"
	"regexp"
	"strings"
)

// StringSlice makes it possible to scan Postgres arrays directly into a golang
// slice. Borrowed from https://gist.github.com/adharris/4163702.
type StringSlice []string

// Scan implements sql.Scanner for the StringSlice type.
func (s *StringSlice) Scan(src interface{}) error {
	asBytes, ok := src.([]byte)
	if !ok {
		return error(gerror.New("Scan source was not []bytes"))
	}

	asString := string(asBytes)
	parsed := parseArray(asString)
	(*s) = StringSlice(parsed)

	return nil
}

// construct a regexp to extract values:
var (
	// unquoted array values must not contain: (" , \ { } whitespace NULL)
	// and must be at least one char
	unquotedChar  = `[^",\\{}\s(NULL)]`
	unquotedValue = fmt.Sprintf("(%s)+", unquotedChar)

	// quoted array values are surrounded by double quotes, can be any
	// character except " or \, which must be backslash escaped:
	quotedChar  = `[^"\\]|\\"|\\\\`
	quotedValue = fmt.Sprintf("\"(%s)*\"", quotedChar)

	// an array value may be either quoted or unquoted:
	arrayValue = fmt.Sprintf("(?P<value>(%s|%s))", unquotedValue, quotedValue)

	// Array values are separated with a comma IF there is more than one value:
	arrayExp = regexp.MustCompile(fmt.Sprintf("((%s)(,)?)", arrayValue))

	valueIndex int
)

// Find the index of the 'value' named expression
func init() {
	for i, subexp := range arrayExp.SubexpNames() {
		if subexp == "value" {
			valueIndex = i
			break
		}
	}
}

// Parse the output string from the array type.
// Regex used: (((?P<value>(([^",\\{}\s(NULL)])+|"([^"\\]|\\"|\\\\)*")))(,)?)
func parseArray(array string) []string {
	results := make([]string, 0)
	matches := arrayExp.FindAllStringSubmatch(array, -1)
	for _, match := range matches {
		s := match[valueIndex]
		// the string _might_ be wrapped in quotes, so trim them:
		s = strings.Trim(s, "\"")
		results = append(results, s)
	}
	return results
}

// TrimStringMax trims a string down if its length is over a certain amount
func TrimStringMax(s string, strLength int) string {
	if strLength <= 0 {
		return s
	}
	r := utf8string.NewString(s)
	if r.RuneCount() > strLength {
		return r.Slice(0, strLength)
	}
	return s
}

// RemoveDupStrings removes duplicates from a slice of strings. The slice of
// strings must be sorted before it's used with this function.
func RemoveDupStrings(strs []string) []string {
	for i, v := range strs {
		// catches the case where we've sliced off all the duplicates,
		// but if we don't break here checking the last element will
		// needlessly keep marching down the remainder of the slice for
		// no effect
		if i > len(strs) {
			break
		}
		j := 1
		s := 0
		for {
			if i+j >= len(strs) {
				break
			}
			if v == strs[i+j] {
				j++
				s++
			} else {
				break
			}
		}
		if s == 0 {
			continue
		}
		strs = delTwoPosElements(i+1, s, strs)
	}
	return strs
}

// DelSliceElement removes an element from a slice of strings.
func DelSliceElement(pos int, strs []string) []string {
	return delTwoPosElements(pos, 1, strs)
}

func delTwoPosElements(pos int, skip int, strs []string) []string {
	strs = append(strs[:pos], strs[pos+skip:]...)
	return strs
}

// take the easy way out at least for now
func StringPresentInSlice(str string, chking []string) bool {
	for _, s := range chking {
		if str == s {
			return true
		}
	}
	return false
}
