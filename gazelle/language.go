package gazelle

import (
	"github.com/bazelbuild/bazel-gazelle/language"
)

const languageName = "TypeScript"

// TypeScript satisfies the language.Language interface. It is the Gazelle
// extension for TypeScript rules.
type TypeScript struct {
	Configurer
	Resolver
}

// NewLanguage initializes a new TypeScript that satisfies the language.Language
// interface. This is the entrypoint for the extension initialization.
func NewLanguage() language.Language {
	return &TypeScript{}
}
