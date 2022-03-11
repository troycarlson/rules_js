package tsconfig

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/emirpasic/gods/lists/singlylinkedlist"
)

// Directives
const (
	// TypeScriptGenerationDirective represents the directive that controls whether
	// this TypeScript generation is enabled or not. Sub-packages inherit this value.
	// Can be either "enabled" or "disabled". Defaults to "enabled".
	TypeScriptGenerationDirective = "ts_generation"
	// TypeScriptRootDirective represents the directive that sets a Bazel package as
	// a TypeScript root. This is used on monorepos with multiple TypeScript projects
	// that don't share the top-level of the workspace as the root.
	TypeScriptRootDirective = "ts_root"
	// IgnoreDependenciesDirective represents the directive that controls the
	// ignored dependencies from the generated targets.
	IgnoreDependenciesDirective = "ts_ignore_dependencies"
	// ValidateImportStatementsDirective represents the directive that controls
	// whether the TypeScript import statements should be validated.
	ValidateImportStatementsDirective = "ts_validate_import_statements"
	// GenerationMode represents the directive that controls the target generation
	// mode. See below for the GenerationModeType constants.
	GenerationMode = "ts_generation_mode"
	// LibraryNamingConvention represents the directive that controls the
	// ts_project naming convention. It interpolates $package_name$ with the
	// Bazel package name. E.g. if the Bazel package name is `foo`, setting this
	// to `$package_name$_my_lib` would render to `foo_my_lib`.
	LibraryNamingConvention = "ts_project_naming_convention"
	// TestNamingConvention represents the directive that controls the py_test
	// naming convention. See ts_project_naming_convention for more info on
	// the package name interpolation.
	TestNamingConvention = "ts_test_naming_convention"
)

// GenerationModeType represents one of the generation modes for the TypeScript
// extension.
type GenerationModeType string

// Generation modes
const (
	// GenerationModePackage defines the mode in which targets will be generated
	// for each __init__.py, or when an existing BUILD or BUILD.bazel file already
	// determines a Bazel package.
	GenerationModePackage GenerationModeType = "package"
	// GenerationModeProject defines the mode in which a coarse-grained target will
	// be generated englobing sub-directories containing TypeScript files.
	GenerationModeProject GenerationModeType = "project"
)

const (
	packageNameNamingConventionSubstitution = "$package_name$"
)

// defaultIgnoreFiles is the list of default values used in the
// ts_ignore_files option.
var defaultIgnoreFiles = map[string]struct{}{}

// Configs is an extension of map[string]*Config. It provides finding methods
// on top of the mapping.
type Configs map[string]*Config

// ParentForPackage returns the parent Config for the given Bazel package.
func (c *Configs) ParentForPackage(pkg string) *Config {
	dir := filepath.Dir(pkg)
	if dir == "." {
		dir = ""
	}
	parent := (map[string]*Config)(*c)[dir]
	return parent
}

// Config represents a config extension for a specific Bazel package.
type Config struct {
	parent *Config

	generationEnabled bool
	repoRoot          string
	tsProjectRoot     string

	excludedPatterns         *singlylinkedlist.List
	ignoreFiles              map[string]struct{}
	ignoreDependencies       map[string]struct{}
	validateImportStatements bool
	coarseGrainedGeneration  bool
	libraryNamingConvention  string
	binaryNamingConvention   string
	testNamingConvention     string
}

// New creates a new Config.
func New(
	repoRoot string,
	tsProjectRoot string,
) *Config {
	return &Config{
		generationEnabled:        true,
		repoRoot:                 repoRoot,
		tsProjectRoot:            tsProjectRoot,
		excludedPatterns:         singlylinkedlist.New(),
		ignoreFiles:              make(map[string]struct{}),
		ignoreDependencies:       make(map[string]struct{}),
		validateImportStatements: true,
		coarseGrainedGeneration:  false,
		libraryNamingConvention:  packageNameNamingConventionSubstitution,
		binaryNamingConvention:   fmt.Sprintf("%s_bin", packageNameNamingConventionSubstitution),
		testNamingConvention:     fmt.Sprintf("%s_test", packageNameNamingConventionSubstitution),
	}
}

// Parent returns the parent config.
func (c *Config) Parent() *Config {
	return c.parent
}

// NewChild creates a new child Config. It inherits desired values from the
// current Config and sets itself as the parent to the child.
func (c *Config) NewChild() *Config {
	return &Config{
		parent:                   c,
		generationEnabled:        c.generationEnabled,
		repoRoot:                 c.repoRoot,
		tsProjectRoot:            c.tsProjectRoot,
		excludedPatterns:         c.excludedPatterns,
		ignoreFiles:              make(map[string]struct{}),
		ignoreDependencies:       make(map[string]struct{}),
		validateImportStatements: c.validateImportStatements,
		coarseGrainedGeneration:  c.coarseGrainedGeneration,
		libraryNamingConvention:  c.libraryNamingConvention,
		binaryNamingConvention:   c.binaryNamingConvention,
		testNamingConvention:     c.testNamingConvention,
	}
}

// AddExcludedPattern adds a glob pattern parsed from the standard
// gazelle:exclude directive.
func (c *Config) AddExcludedPattern(pattern string) {
	c.excludedPatterns.Add(pattern)
}

// ExcludedPatterns returns the excluded patterns list.
func (c *Config) ExcludedPatterns() *singlylinkedlist.List {
	return c.excludedPatterns
}

// SetGenerationEnabled sets whether the extension is enabled or not.
func (c *Config) SetGenerationEnabled(enabled bool) {
	c.generationEnabled = enabled
}

// GenerationEnabled returns whether the extension is enabled or not.
func (c *Config) GenerationEnabled() bool {
	return c.generationEnabled
}

// SetTypeScriptProjectRoot sets the TypeScript project root.
func (c *Config) SetTypeScriptProjectRoot(tsProjectRoot string) {
	c.tsProjectRoot = tsProjectRoot
}

// TypeScriptProjectRoot returns the TypeScript project root.
func (c *Config) TypeScriptProjectRoot() string {
	return c.tsProjectRoot
}

// FindThirdPartyDependency scans the gazelle manifests for the current config
// and the parent configs up to the root finding if it can resolve the module
// name.
func (c *Config) FindThirdPartyDependency(modName string) (string, bool) {
	// TODO
	return "", false
}

// AddIgnoreFile adds a file to the list of ignored files for a given package.
// Adding an ignored file to a package also makes it ignored on a subpackage.
func (c *Config) AddIgnoreFile(file string) {
	c.ignoreFiles[strings.TrimSpace(file)] = struct{}{}
}

// IgnoresFile checks if a file is ignored in the given package or in one of the
// parent packages up to the workspace root.
func (c *Config) IgnoresFile(file string) bool {
	trimmedFile := strings.TrimSpace(file)

	if _, ignores := defaultIgnoreFiles[trimmedFile]; ignores {
		return true
	}

	if _, ignores := c.ignoreFiles[trimmedFile]; ignores {
		return true
	}

	parent := c.parent
	for parent != nil {
		if _, ignores := parent.ignoreFiles[trimmedFile]; ignores {
			return true
		}
		parent = parent.parent
	}

	return false
}

// AddIgnoreDependency adds a dependency to the list of ignored dependencies for
// a given package. Adding an ignored dependency to a package also makes it
// ignored on a subpackage.
func (c *Config) AddIgnoreDependency(dep string) {
	c.ignoreDependencies[strings.TrimSpace(dep)] = struct{}{}
}

// IgnoresDependency checks if a dependency is ignored in the given package or
// in one of the parent packages up to the workspace root.
func (c *Config) IgnoresDependency(dep string) bool {
	trimmedDep := strings.TrimSpace(dep)

	if _, ignores := c.ignoreDependencies[trimmedDep]; ignores {
		return true
	}

	parent := c.parent
	for parent != nil {
		if _, ignores := parent.ignoreDependencies[trimmedDep]; ignores {
			return true
		}
		parent = parent.parent
	}

	return false
}

// SetValidateImportStatements sets whether TypeScript import statements should be
// validated or not. It throws an error if this is set multiple times, i.e. if
// the directive is specified multiple times in the Bazel workspace.
func (c *Config) SetValidateImportStatements(validate bool) {
	c.validateImportStatements = validate
}

// ValidateImportStatements returns whether the TypeScript import statements should
// be validated or not. If this option was not explicitly specified by the user,
// it defaults to true.
func (c *Config) ValidateImportStatements() bool {
	return c.validateImportStatements
}

// SetCoarseGrainedGeneration sets whether coarse-grained targets should be
// generated or not.
func (c *Config) SetCoarseGrainedGeneration(coarseGrained bool) {
	c.coarseGrainedGeneration = coarseGrained
}

// CoarseGrainedGeneration returns whether coarse-grained targets should be
// generated or not.
func (c *Config) CoarseGrainedGeneration() bool {
	return c.coarseGrainedGeneration
}

// SetLibraryNamingConvention sets the ts_project target naming convention.
func (c *Config) SetLibraryNamingConvention(libraryNamingConvention string) {
	c.libraryNamingConvention = libraryNamingConvention
}

// RenderLibraryName returns the ts_project target name by performing all
// substitutions.
func (c *Config) RenderLibraryName(packageName string) string {
	return strings.ReplaceAll(c.libraryNamingConvention, packageNameNamingConventionSubstitution, packageName)
}

// SetBinaryNamingConvention sets the ts_project target naming convention.
func (c *Config) SetBinaryNamingConvention(binaryNamingConvention string) {
	c.binaryNamingConvention = binaryNamingConvention
}

// SetTestNamingConvention sets the py_test target naming convention.
func (c *Config) SetTestNamingConvention(testNamingConvention string) {
	c.testNamingConvention = testNamingConvention
}

// RenderTestName returns the py_test target name by performing all
// substitutions.
func (c *Config) RenderTestName(packageName string) string {
	return strings.ReplaceAll(c.testNamingConvention, packageNameNamingConventionSubstitution, packageName)
}
