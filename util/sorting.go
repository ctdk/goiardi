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

package util

type Int64Sort []int64

func (i6 Int64Sort) Len() int           { return len(i6) }
func (i6 Int64Sort) Swap(i, j int)      { i6[i], i6[j] = i6[j], i6[i] }
func (i6 Int64Sort) Less(i, j int) bool { return i6[i] < i6[j] }

// RemoveDupInt64s removes duplicates from a slice of int64s. The slice must
// already be sorted before using this function.
func RemoveDupInt64s(i6s []int64) []int64 {
	for i, v := range i6s {
		if i > len(i6s) {
			break
		}
		j := 1
		s := 0
		for {
			if i+j >= len(i6s) {
				break
			}
			if v == i6s[i+j] {
				j++
				s++
			} else {
				break
			}
		}
		if s == 0 {
			continue
		}
		// don't really need the overhead of yet another little function
		pos := i + 1
		i6s = append(i6s[:pos], i6s[pos+s:]...)
	}
	return i6s
}
