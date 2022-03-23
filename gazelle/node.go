package gazelle

import (
	"github.com/emirpasic/gods/sets/treeset"
)

// TODO(jbedard): launch node to determine all native modules
// See https://nodejs.org/api/module.html#modulebuiltinmodules

var NODE_LIBS = treeset.NewWithStringComparator(
	"fs",
	"path",
)

func isNodeImport(imp string) bool {
	return NODE_LIBS.Contains(imp)
}
