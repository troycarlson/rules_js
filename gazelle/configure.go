package gazelle

import (
	"flag"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/rule"

	"aspect.build/rules_js/gazelle/tsconfig"
)

// Configurer satisfies the config.Configurer interface. It's the
// language-specific configuration extension.
type Configurer struct{}

// RegisterFlags registers command-line flags used by the extension. This
// method is called once with the root configuration when Gazelle
// starts. RegisterFlags may set an initial values in Config.Exts. When flags
// are set, they should modify these values.
func (ts *Configurer) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {}

// CheckFlags validates the configuration after command line flags are parsed.
// This is called once with the root configuration when Gazelle starts.
// CheckFlags may set default values in flags or make implied changes.
func (ts *Configurer) CheckFlags(fs *flag.FlagSet, c *config.Config) error {
	return nil
}

// KnownDirectives returns a list of directive keys that this Configurer can
// interpret. Gazelle prints errors for directives that are not recoginized by
// any Configurer.
func (ts *Configurer) KnownDirectives() []string {
	return []string{
		tsconfig.TypeScriptGenerationDirective,
		tsconfig.TypeScriptRootDirective,
		tsconfig.IgnoreDependenciesDirective,
		tsconfig.ValidateImportStatementsDirective,
		tsconfig.GenerationMode,
		tsconfig.LibraryNamingConvention,
		tsconfig.TestNamingConvention,
	}
}

// Configure modifies the configuration using directives and other information
// extracted from a build file. Configure is called in each directory.
//
// c is the configuration for the current directory. It starts out as a copy
// of the configuration for the parent directory.
//
// rel is the slash-separated relative path from the repository root to
// the current directory. It is "" for the root directory itself.
//
// f is the build file for the current directory or nil if there is no
// existing build file.
func (ts *Configurer) Configure(c *config.Config, rel string, f *rule.File) {
	// Create the root config.
	if _, exists := c.Exts[languageName]; !exists {
		rootConfig := tsconfig.New(c.RepoRoot, "")
		c.Exts[languageName] = tsconfig.Configs{"": rootConfig}
	}

	configs := c.Exts[languageName].(tsconfig.Configs)

	config, exists := configs[rel]
	if !exists {
		parent := configs.ParentForPackage(rel)
		config = parent.NewChild()
		configs[rel] = config
	}

	if f == nil {
		return
	}

	for _, d := range f.Directives {
		switch d.Key {
		case "exclude":
			// We record the exclude directive for coarse-grained packages
			// since we do manual tree traversal in this mode.
			config.AddExcludedPattern(strings.TrimSpace(d.Value))
		case tsconfig.TypeScriptGenerationDirective:
			switch d.Value {
			case "enabled":
				config.SetGenerationEnabled(true)
			case "disabled":
				config.SetGenerationEnabled(false)
			default:
				err := fmt.Errorf("invalid value for directive %q: %s: possible values are enabled/disabled",
					tsconfig.TypeScriptGenerationDirective, d.Value)
				log.Fatal(err)
			}
		case tsconfig.TypeScriptRootDirective:
			config.SetTypeScriptProjectRoot(rel)
		case tsconfig.IgnoreDependenciesDirective:
			for _, ignoreDependency := range strings.Split(d.Value, ",") {
				config.AddIgnoreDependency(ignoreDependency)
			}
		case tsconfig.ValidateImportStatementsDirective:
			v, err := strconv.ParseBool(strings.TrimSpace(d.Value))
			if err != nil {
				log.Fatal(err)
			}
			config.SetValidateImportStatements(v)
		case tsconfig.GenerationMode:
			switch tsconfig.GenerationModeType(strings.TrimSpace(d.Value)) {
			case tsconfig.GenerationModePackage:
				config.SetCoarseGrainedGeneration(false)
			case tsconfig.GenerationModeProject:
				config.SetCoarseGrainedGeneration(true)
			default:
				err := fmt.Errorf("invalid value for directive %q: %s",
					tsconfig.GenerationMode, d.Value)
				log.Fatal(err)
			}
		case tsconfig.LibraryNamingConvention:
			config.SetLibraryNamingConvention(strings.TrimSpace(d.Value))
		case tsconfig.TestNamingConvention:
			config.SetTestNamingConvention(strings.TrimSpace(d.Value))
		}
	}
}
