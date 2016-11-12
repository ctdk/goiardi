This module is ONLY here for running an authentication test, and has been
imported into the goiardi source tree solely to make packaging goiardi for
debian easier (because go-chef has a dependency on goiardi, and goiardi uses
go-chef for a test, the packages have trouble building with the usual debian
go package procedure).

You really shouldn't use it anywhere else.
