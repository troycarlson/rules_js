package gazelle

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/emirpasic/gods/lists/singlylinkedlist"
)

type EnvironmentType string

const (
	EnvironmentNode    EnvironmentType = "node"
	EnvironmentBrowser EnvironmentType = "browser"
	EnvironmentOther   EnvironmentType = "other"
)

// Directives
const (
	// TypeScriptGenerationDirective represents the directive that controls whether
	// this TypeScript generation is enabled or not. Sub-packages inherit this value.
	// Can be either "enabled" or "disabled". Defaults to "enabled".
	TypeScriptGenerationDirective = "ts_generation"
	// IgnoreDependenciesDirective represents the directive that controls the
	// ignored dependencies from the generated targets.
	IgnoreDependenciesDirective = "ts_ignore_dependencies"
	// ValidateImportStatementsDirective represents the directive that controls
	// whether the TypeScript import statements should be validated.
	ValidateImportStatementsDirective = "ts_validate_import_statements"
	// EnvironmentDirective represents the runtime environment such as in a browser, node etc.
	// and effects which native imports are available.
	EnvironmentDirective = "ts_environment"
	// LibraryNamingConvention represents the directive that controls the
	// ts_project naming convention. It interpolates $package_name$ with the
	// Bazel package name. E.g. if the Bazel package name is `foo`, setting this
	// to `$package_name$_my_lib` would render to `foo_my_lib`.
	LibraryNamingConvention = "ts_project_naming_convention"
	// TestNamingConvention represents the directive that controls the ts_project test
	// naming convention. See ts_project_naming_convention for more info on
	// the package name interpolation.
	TestNamingConvention = "ts_test_naming_convention"
)

const (
	packageNameNamingConventionSubstitution = "$package_name$"
)

// Configs is an extension of map[string]*TypeScriptConfig. It provides finding methods
// on top of the mapping.
type Configs map[string]*TypeScriptConfig

// ParentForPackage returns the parent TypeScriptConfig for the given Bazel package.
func (c *Configs) ParentForPackage(pkg string) *TypeScriptConfig {
	dir := filepath.Dir(pkg)
	if dir == "." {
		dir = ""
	}
	parent := (map[string]*TypeScriptConfig)(*c)[dir]
	return parent
}

// TypeScriptConfig represents a config extension for a specific Bazel package.
type TypeScriptConfig struct {
	parent *TypeScriptConfig

	generationEnabled bool
	repoRoot          string
	environmentType   EnvironmentType

	excludedPatterns         *singlylinkedlist.List
	ignoreDependencies       map[string]struct{}
	validateImportStatements bool
	libraryNamingConvention  string
	testNamingConvention     string
}

// New creates a new TypeScriptConfig.
func NewTypeScriptConfig(
	repoRoot string,
) *TypeScriptConfig {
	return &TypeScriptConfig{
		generationEnabled:        true,
		repoRoot:                 repoRoot,
		environmentType:          EnvironmentOther,
		excludedPatterns:         singlylinkedlist.New(),
		ignoreDependencies:       make(map[string]struct{}),
		validateImportStatements: true,
		libraryNamingConvention:  packageNameNamingConventionSubstitution,
		testNamingConvention:     fmt.Sprintf("%s_test", packageNameNamingConventionSubstitution),
	}
}

// Parent returns the parent config.
func (c *TypeScriptConfig) Parent() *TypeScriptConfig {
	return c.parent
}

// NewChild creates a new child TypeScriptConfig. It inherits desired values from the
// current TypeScriptConfig and sets itself as the parent to the child.
func (c *TypeScriptConfig) NewChild() *TypeScriptConfig {
	return &TypeScriptConfig{
		parent:                   c,
		generationEnabled:        c.generationEnabled,
		repoRoot:                 c.repoRoot,
		environmentType:          c.environmentType,
		excludedPatterns:         c.excludedPatterns,
		ignoreDependencies:       make(map[string]struct{}),
		validateImportStatements: c.validateImportStatements,
		libraryNamingConvention:  c.libraryNamingConvention,
		testNamingConvention:     c.testNamingConvention,
	}
}

// AddExcludedPattern adds a glob pattern parsed from the standard
// gazelle:exclude directive.
func (c *TypeScriptConfig) AddExcludedPattern(pattern string) {
	c.excludedPatterns.Add(pattern)
}

// ExcludedPatterns returns the excluded patterns list.
func (c *TypeScriptConfig) ExcludedPatterns() *singlylinkedlist.List {
	return c.excludedPatterns
}

// SetGenerationEnabled sets whether the extension is enabled or not.
func (c *TypeScriptConfig) SetGenerationEnabled(enabled bool) {
	c.generationEnabled = enabled
}

// GenerationEnabled returns whether the extension is enabled or not.
func (c *TypeScriptConfig) GenerationEnabled() bool {
	return c.generationEnabled
}

// FindThirdPartyDependency finds third-party dependencies which fulfill the given import path.
func (c *TypeScriptConfig) FindThirdPartyDependency(imp string) (string, bool) {
	// TODO(jbedard)

	return "", false
}

// AddIgnoreDependency adds a dependency to the list of ignored dependencies for
// a given package. Adding an ignored dependency to a package also makes it
// ignored on a subpackage.
func (c *TypeScriptConfig) AddIgnoreDependency(dep string) {
	c.ignoreDependencies[strings.TrimSpace(dep)] = struct{}{}
}

// IgnoresDependency checks if a dependency is ignored in the given package or
// in one of the parent packages up to the workspace root.
func (c *TypeScriptConfig) IgnoresDependency(dep string) bool {
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
func (c *TypeScriptConfig) SetValidateImportStatements(validate bool) {
	c.validateImportStatements = validate
}

// ValidateImportStatements returns whether the TypeScript import statements should
// be validated or not. If this option was not explicitly specified by the user,
// it defaults to true.
func (c *TypeScriptConfig) ValidateImportStatements() bool {
	return c.validateImportStatements
}

// SetEnvironment sets the TypeScirpt/JS environment type
func (c *TypeScriptConfig) SetEnvironmentType(envType EnvironmentType) {
	c.environmentType = envType
}

// Environment returns the TypeScirpt/JS environment type
func (c *TypeScriptConfig) EnvironmentType() EnvironmentType {
	return c.environmentType
}

// SetLibraryNamingConvention sets the ts_project target naming convention.
func (c *TypeScriptConfig) SetLibraryNamingConvention(libraryNamingConvention string) {
	c.libraryNamingConvention = libraryNamingConvention
}

// RenderLibraryName returns the ts_project target name by performing all
// substitutions.
func (c *TypeScriptConfig) RenderLibraryName(packageName string) string {
	return strings.ReplaceAll(c.libraryNamingConvention, packageNameNamingConventionSubstitution, packageName)
}

// SetTestNamingConvention sets the ts_project test target naming convention.
func (c *TypeScriptConfig) SetTestNamingConvention(testNamingConvention string) {
	c.testNamingConvention = testNamingConvention
}

// RenderTestName returns the ts_project test target name by performing all
// substitutions.
func (c *TypeScriptConfig) RenderTestName(packageName string) string {
	return strings.ReplaceAll(c.testNamingConvention, packageNameNamingConventionSubstitution, packageName)
}
