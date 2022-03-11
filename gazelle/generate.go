package gazelle

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/language"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/bmatcuk/doublestar"
	"github.com/emirpasic/gods/lists/singlylinkedlist"
	"github.com/emirpasic/gods/sets/treeset"
	godsutils "github.com/emirpasic/gods/utils"

	"aspect.build/rules_js/gazelle/tsconfig"
)

var (
	buildFilenames = []string{"BUILD", "BUILD.bazel"}
	// errHaltDigging is an error that signals whether the generator should halt
	// digging the source tree searching for modules in subdirectories.
	errHaltDigging = fmt.Errorf("halt digging")
)

// GenerateRules extracts build metadata from source files in a directory.
// GenerateRules is called in each directory where an update is requested
// in depth-first post-order.
func (ts *TypeScript) GenerateRules(args language.GenerateArgs) language.GenerateResult {
	cfgs := args.Config.Exts[languageName].(tsconfig.Configs)
	cfg := cfgs[args.Rel]

	// When we return empty, we mean that we don't generate anything, but this
	// still triggers the indexing for all the TypeScript targets in this
	// package.
	if !cfg.GenerationEnabled() {
		return language.GenerateResult{}
	}

	if !isBazelPackage(args.Dir) {
		if cfg.CoarseGrainedGeneration() {
			// Determine if the current directory is the root of the coarse-grained
			// generation. If not, return without generating anything.
			parent := cfg.Parent()
			if parent != nil && parent.CoarseGrainedGeneration() {
				return language.GenerateResult{}
			}
		} else {
			return language.GenerateResult{}
		}
	}

	tsProjectRoot := cfg.TypeScriptProjectRoot()

	packageName := filepath.Base(args.Dir)

	tsProjectFilenames := treeset.NewWith(godsutils.StringComparator)

	for _, f := range args.RegularFiles {
		if cfg.IgnoresFile(filepath.Base(f)) {
			continue
		}
		ext := filepath.Ext(f)
		// TODO: js, json, especially the eteceteras.
		// if !hasPyBinary && f == pyBinaryEntrypointFilename {
		// 	hasPyBinary = true
		// } else if !hasPyTestFile && f == pyTestEntrypointFilename {
		// 	hasPyTestFile = true
		// } else if strings.HasSuffix(f, "_test.ts") || (strings.HasPrefix(f, "test_") && ext == ".ts") {
		// 	pyTestFilenames.Add(f)
		// } else if ext == ".ts" {
		// 	tsProjectFilenames.Add(f)
		// }
		if ext == ".ts" {
			tsProjectFilenames.Add(f)
		}
	}

	// TODO: check args.OtherGen for .ts files?

	// Add files from subdirectories if they meet the criteria.
	for _, d := range args.Subdirs {
		// boundaryPackages represents child Bazel packages that are used as a
		// boundary to stop processing under that tree.
		boundaryPackages := make(map[string]struct{})
		err := filepath.Walk(
			filepath.Join(args.Dir, d),
			func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				// Ignore the path if it crosses any boundary package. Walking
				// the tree is still important because subsequent paths can
				// represent files that have not crossed any boundaries.
				for bp := range boundaryPackages {
					if strings.HasPrefix(path, bp) {
						return nil
					}
				}

				if info.IsDir() {
					// If we are visiting a directory we halt digging the tree if
					// the directory has a BUILD or BUILD.bazel.
					if isBazelPackage(path) {
						boundaryPackages[path] = struct{}{}
						return nil
					}

					if !cfg.CoarseGrainedGeneration() {
						return errHaltDigging
					}

					return nil
				}

				if filepath.Ext(path) == ".ts" {
					if cfg.CoarseGrainedGeneration() {
						f, _ := filepath.Rel(args.Dir, path)
						excludedPatterns := cfg.ExcludedPatterns()
						if excludedPatterns != nil {
							it := excludedPatterns.Iterator()
							for it.Next() {
								excludedPattern := it.Value().(string)
								isExcluded, err := doublestar.Match(excludedPattern, f)
								if err != nil {
									return err
								}
								if isExcluded {
									return nil
								}
							}
						}
						tsProjectFilenames.Add(f)
					}
				}
				return nil
			},
		)
		if err != nil && err != errHaltDigging {
			log.Printf("ERROR: %v\n", err)
			return language.GenerateResult{}
		}
	}

	visibility := fmt.Sprintf("//%s:__subpackages__", tsProjectRoot)

	var result language.GenerateResult
	result.Gen = make([]*rule.Rule, 0)

	collisionErrors := singlylinkedlist.New()

	var tsProject *rule.Rule
	if !tsProjectFilenames.Empty() {
		// TODO
		deps := treeset.NewWith(godsutils.StringComparator)

		tsProjectTargetName := cfg.RenderLibraryName(packageName)

		// Check if a target with the same name we are generating alredy exists,
		// and if it is of a different kind from the one we are generating. If
		// so, we have to throw an error since Gazelle won't generate it
		// correctly.
		if args.File != nil {
			for _, t := range args.File.Rules {
				if t.Name() == tsProjectTargetName && t.Kind() != tsProjectKind {
					fqTarget := label.New("", args.Rel, tsProjectTargetName)
					err := fmt.Errorf("failed to generate target %q of kind %q: "+
						"a target of kind %q with the same name already exists. "+
						"Use the '# gazelle:%s' directive to change the naming convention.",
						fqTarget.String(), tsProjectKind, t.Kind(), tsconfig.LibraryNamingConvention)
					collisionErrors.Add(err)
				}
			}
		}

		tsProject = newTargetBuilder(tsProjectKind, tsProjectTargetName, tsProjectRoot, args.Rel).
			addVisibility(visibility).
			addSrcs(tsProjectFilenames).
			addModuleDependencies(deps).
			build()

		result.Gen = append(result.Gen, tsProject)
		result.Imports = append(result.Imports, tsProject.PrivateAttr(config.GazelleImportsKey))
	}

	if !collisionErrors.Empty() {
		it := collisionErrors.Iterator()
		for it.Next() {
			log.Printf("ERROR: %v\n", it.Value())
		}
		os.Exit(1)
	}

	return result
}

// isBazelPackage determines if the directory is a Bazel package by probing for
// the existence of a known BUILD file name.
func isBazelPackage(dir string) bool {
	for _, buildFilename := range buildFilenames {
		path := filepath.Join(dir, buildFilename)
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
}

// hasEntrypointFile determines if the directory has any of the established
// entrypoint filenames.
func hasEntrypointFile(dir string) bool {
	// TODO?
	return false
}

// isEntrypointFile returns whether the given path is an entrypoint file. The
// given path can be absolute or relative.
func isEntrypointFile(path string) bool {
	// TODO?
	return false
}
