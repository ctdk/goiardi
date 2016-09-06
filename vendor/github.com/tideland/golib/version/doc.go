// Tideland Go Library - Version
//
// Copyright (C) 2014-2015 Frank Mueller / Tideland / Oldenburg / Germany
//
// All rights reserved. Use of this source code is governed
// by the new BSD license.

// The Tideland Go Library version package helps other packages to
// provide information about their current version and compare it
// to others. It follows the idea of semantic versioning.
package version

//--------------------
// VERSION
//--------------------

// PackageVersion returns the version of the version package.
func PackageVersion() Version {
	return New(2, 0, 0)
}

// EOF
