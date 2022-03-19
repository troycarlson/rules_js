package gazelle

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/repo"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
	bzl "github.com/bazelbuild/buildtools/build"
	"github.com/emirpasic/gods/sets/treeset"
)

const (
	// resolvedDepsKey is the attribute key used to pass dependencies that don't
	// need to be resolved by the dependency resolver in the Resolver step.
	resolvedDepsKey = "_gazelle_ts_resolved_deps"
)

var EXPLAIN_DEPENDENCY = os.Getenv("EXPLAIN_DEPENDENCY")

// Resolver satisfies the resolve.Resolver interface. It resolves dependencies
// in rules generated by this extension.
type Resolver struct{}

// Name returns the name of the language. This is the prefix of the kinds of
// rules generated. E.g. ts_project
func (*Resolver) Name() string { return languageName }

// Determine what rule (r) outputs which can be imported.
// For TypeScript this is all the import-paths pointing to files within the rule.
func (ts *Resolver) Imports(c *config.Config, r *rule.Rule, f *rule.File) []resolve.ImportSpec {
	srcs := r.AttrStrings("srcs")
	provides := make([]resolve.ImportSpec, 0, len(srcs)+1)

	for _, src := range srcs {
		spec := stripImportExtensions(filepath.Clean(filepath.Join(f.Pkg, src)))

		provides = append(provides, resolve.ImportSpec{
			Lang: languageName,
			Imp:  spec,
		})

		// Index files can also be imported using only the directory
		if isIndexFile(src) {
			provides = append(provides, resolve.ImportSpec{
				Lang: languageName,
				Imp:  strings.TrimRight(filepath.Dir(spec), indexFileName),
			})
		}
	}

	if len(provides) == 0 {
		return nil
	}

	DEBUG("PROVIDES(%s): %s\n", r.Name(), provides)

	return provides
}

// Embeds returns a list of labels of rules that the given rule embeds. If
// a rule is embedded by another importable rule of the same language, only
// the embedding rule will be indexed. The embedding rule will inherit
// the imports of the embedded rule.
func (ts *Resolver) Embeds(r *rule.Rule, from label.Label) []label.Label {
	// TODO(jbedard): implement.
	return make([]label.Label, 0)
}

// Resolve translates imported libraries for a given rule into Bazel
// dependencies. Information about imported libraries is returned for each
// rule generated by language.GenerateRules in
// language.GenerateResult.Imports. Resolve generates a "deps" attribute (or
// the appropriate language-specific equivalent) for each import according to
// language-specific rules and heuristics.
func (ts *Resolver) Resolve(
	c *config.Config,
	ix *resolve.RuleIndex,
	rc *repo.RemoteCache,
	r *rule.Rule,
	modulesRaw interface{},
	from label.Label,
) {
	deps := treeset.NewWithStringComparator()

	// Pre-resolved deps
	resolvedDepsIt := r.PrivateAttr(resolvedDepsKey).(*treeset.Set).Iterator()
	for resolvedDepsIt.Next() {
		deps.Add(resolvedDepsIt.Value())
	}

	if modulesRaw != nil {
		ResolveModuleDeps(c, ix, modulesRaw.(*treeset.Set), from, deps)
	}

	if !deps.Empty() {
		r.SetAttr("deps", convertDependencySetToExpr(deps))
	}
}

func ResolveModuleDeps(
	c *config.Config,
	ix *resolve.RuleIndex,
	modules *treeset.Set,
	from label.Label,
	deps *treeset.Set,
) {
	cfgs := c.Exts[languageName].(Configs)
	cfg := cfgs[from.Pkg]
	hasFatalError := false

	it := modules.Iterator()
	for it.Next() {
		mod := it.Value().(ImportStatement)
		imp := resolve.ImportSpec{
			Lang: languageName,
			Imp:  toWorkspaceImportPath(from.Pkg, mod.SourcePath, mod.Path),
		}

		DEBUG("FIND(%s): %s\n", from.Name, imp.Imp)

		if override, ok := resolve.FindRuleWithOverride(c, imp, languageName); ok {
			if override.Repo == "" {
				override.Repo = from.Repo
			}
			if !override.Equal(from) {
				if override.Repo == from.Repo {
					override.Repo = ""
				}
				dep := override.String()
				deps.Add(dep)
				if EXPLAIN_DEPENDENCY == dep {
					log.Printf("Explaining dependency (%s): "+
						"in the target %q, the file %q imports %q at line %d, "+
						"which resolves using the \"gazelle:resolve\" directive.\n",
						EXPLAIN_DEPENDENCY, from.String(), mod.SourcePath, mod.Path, mod.SourceLineNumber)
				}
			}
		} else if matches := ix.FindRulesByImportWithConfig(c, imp, languageName); len(matches) > 0 {
			filteredMatches := make([]resolve.FindResult, 0, len(matches))
			for _, match := range matches {
				// Prevent from adding itself as a dependency.
				if !match.IsSelfImport(from) {
					filteredMatches = append(filteredMatches, match)
				}
			}

			DEBUG("MATCHES(%s): %s\n", from.Name, filteredMatches)

			if len(filteredMatches) == 1 {
				matchLabel := filteredMatches[0].Label.Rel(from.Repo, from.Pkg)
				dep := matchLabel.String()
				deps.Add(dep)
				if EXPLAIN_DEPENDENCY == dep {
					log.Printf("Explaining dependency (%s): "+
						"in the target %q, the file %q imports %q at line %d, "+
						"which resolves from the first-party indexed labels.\n",
						EXPLAIN_DEPENDENCY, from.String(), mod.SourcePath, mod.Path, mod.SourceLineNumber)
				}
			} else if len(filteredMatches) > 1 {
				err := fmt.Errorf(
					"multiple targets (%s) may be imported with %q at line %d in %q "+
						"- this must be fixed using the \"gazelle:resolve\" directive",
					targetListFromResults(filteredMatches), mod.Path, mod.SourceLineNumber, mod.SourcePath)
				log.Println("ERROR: ", err)
				hasFatalError = true
			}
		} else if dep, ok := cfg.FindThirdPartyDependency(mod.Path); ok {
			deps.Add(dep)
			if EXPLAIN_DEPENDENCY == dep {
				log.Printf("Explaining dependency (%s): "+
					"in the target %q, the file %q imports %q at line %d, "+
					"which resolves from the third-party package %q.\n",
					EXPLAIN_DEPENDENCY, from.String(), mod.SourcePath, mod.Path, mod.SourceLineNumber, dep)
			}
		} else if cfg.ValidateImportStatements() {
			err := fmt.Errorf(
				"%[1]q at line %[2]d from %[3]q is an invalid dependency: possible solutions:\n"+
					"\t1. Add it as a dependency in the requirements.txt file.\n"+
					"\t2. Instruct Gazelle to resolve to a known dependency using the gazelle:resolve directive.\n"+
					"\t3. Ignore it with a comment '# gazelle:ignore %[1]s' in the TypeScript file.\n",
				mod.Path, mod.SourceLineNumber, mod.SourcePath,
			)
			log.Printf("ERROR: failed to validate dependencies for target %q: %v\n", from.String(), err)
			hasFatalError = true
		}
	}
	if hasFatalError {
		os.Exit(1)
	}
}

// Normalize the given import statement from a relative path
// to a path relative to the workspace.
func toWorkspaceImportPath(pkg, src, impt string) string {
	// Convert relative to workspace-relative
	if impt[0] == '.' {
		impt = filepath.Join(pkg, filepath.Dir(src), impt)
	}

	// Clean any extra . / .. etc
	impt = filepath.Clean(impt)

	// Trim supported TS extensions
	ext := filepath.Ext(impt)
	if len(ext) > 0 {
		for _, tsExt := range typescriptSourceExtensions {
			if ext == tsExt {
				return strings.TrimSuffix(impt, tsExt)
			}
		}
	}

	return impt
}

// targetListFromResults returns a string with the human-readable list of
// targets contained in the given results.
func targetListFromResults(results []resolve.FindResult) string {
	list := make([]string, len(results))
	for i, result := range results {
		list[i] = result.Label.String()
	}
	return strings.Join(list, ", ")
}

// convertDependencySetToExpr converts the given set of dependencies to an
// expression to be used in the deps attribute.
func convertDependencySetToExpr(set *treeset.Set) bzl.Expr {
	deps := make([]bzl.Expr, set.Size())
	it := set.Iterator()
	for it.Next() {
		dep := it.Value().(string)
		deps[it.Index()] = &bzl.StringExpr{Value: dep}
	}
	return &bzl.ListExpr{List: deps}
}
