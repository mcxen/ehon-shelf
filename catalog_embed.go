package main

import _ "embed"

// builtinCatalog lists all 470 works so the shelf can browse offline.
//
//go:embed catalog.json
var builtinCatalog []byte
