This module is ONLY here for running an authentication test, and has been
imported into the goiardi source tree solely to make packaging goiardi for
debian easier (because go-chef has a dependency on goiardi, and goiardi uses
go-chef for a test, the packages have trouble building with the usual debian
go package procedure).

Also, most of the files and functionality here have been stripped out - the only
part remaining is the minimum needed to fake a request to test authentication.

You really shouldn't use it anywhere else.

## AUTHORS

|               |                                                |
|:--------------|:-----------------------------------------------|
|Jesse Nelson   |[@spheromak](https://github.com/spheromak)
|AJ Christensen |[@fujin](https://github.com/fujin)
|Brad Beam      |[@bradbeam](https://github.com/bradbeam)
|Kraig Amador   |[@bigkraig](https://github.com/bigkraig)

## COPYRIGHT

Copyright 2013-2014, Jesse Nelson

## LICENSE

Like many Chef ecosystem programs, go-chef/chef is licensed under the Apache 2.0
License. See the LICENSE file for details.

Chef is copyright (c) 2008-2014 Chef, Inc. and its various contributors.

Thanks go out to the fine folks of Opscode and the Chef community for all their
hard work.
