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

/* Package international is a module that provides translations of goiardi error messages and other output where possible. If a particular message cannot be found in a particular language, it will fall back to the English version.

IMPORTANT NOTE: If this message is still in the source file, it means that this feature is not yet implemented. There's a non-trivial amount of work to do to get it ready just for English, totally leaving aside any translations to other languages.
*/
package international

import (
	"github.com/ctdk/goiardi/config"
)

const defaultLang = "english"

// Text is a struct of translations of output strings with a consistent
// identifier.
type Text struct {
	Name         string            // a consistent identifier for the message
	translations map[string]string // map of message translations, keyed by language
}

// GetString, unsurprisingly, gets the translation of a string for a particular
// language goiardi's been configured to use with fallback to English.
func (t *Text) GetString() string {
	s, ok := t.translations[config.Config.Language]
	if !ok {
		s = t.translations[defaultLang]
	}
	return s
}
