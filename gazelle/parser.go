package gazelle

import (
	"io/ioutil"
	"log"

	"path/filepath"

	"github.com/evanw/esbuild/pkg/api"
)

// Scanner reads a file into a string.
type Scanner struct {
}

// Scan reads a file named name in directory dir into a string.
// The contents of the file are stored in fileInfo.Content.
func (s *Scanner) Scan(dir string, name string) *FileInfo {
	fpath := filepath.Join(dir, name)
	filename := name
	content, err := ioutil.ReadFile(fpath)
	if err != nil {
		log.Printf("%s: error reading ts file: %v", fpath, err)
		return nil
	}
	return &FileInfo{
		Path:     fpath,
		Filename: filename,
		Content:  string(content),
	}
}

func NewScanner() *Scanner {
	return &Scanner{}
}

type FileInfo struct {
	Path     string
	Filename string
	Content  string
}

type Parser struct {
}

func NewParser() *Parser {
	p := &Parser{}
	return p
}

// filenameToLoader takes in a filename, e.g. myFile.ts,
// and returns the appropriate esbuild loader for that file.
func filenameToLoader(filename string) api.Loader {
	ext := filepath.Ext(filename)
	switch ext {
	case ".ts":
		return api.LoaderTS
	case ".tsx":
		return api.LoaderTSX
	case ".js":
		return api.LoaderJSX
	case ".jsx":
		return api.LoaderJSX
	default:
		return api.LoaderTS
	}
}

// ParseImports returns all the imports from a file
// after parsing it.
func (p *Parser) ParseImports(fileInfo *FileInfo) []string {
	imports := []string{}
	if filepath.Ext(fileInfo.Filename) == ".css" {
		// No need to try to parse CSS.
		return imports
	}
	// Construct an esbuild plugin that pulls out all the imports.
	plugin := api.Plugin{
		Name: "GetImports",
		Setup: func(pluginBuild api.PluginBuild) {
			// callback is a handler for esbuild resolutions. This is how
			// we'll get access to every import in the file.
			callback := func(args api.OnResolveArgs) (api.OnResolveResult, error) {
				// Add the imported string to our list of imports.
				imports = append(imports, args.Path)
				return api.OnResolveResult{
					// Mark the import as external so esbuild doesn't complain
					// about not being able to find the import.
					External: true,
				}, nil
			}

			// pluginBuild.OnResolve sets the plugin's onResolve callback to our custom callback.
			// Make sure to set Filter: ".*" so that our plugin runs on all imports.
			pluginBuild.OnResolve(api.OnResolveOptions{Filter: ".*", Namespace: ""}, callback)
		},
	}
	options := api.BuildOptions{
		Stdin: &api.StdinOptions{
			Contents:   fileInfo.Content,
			Sourcefile: fileInfo.Filename,
			// The Loader determines how esbuild will parse the file.
			// We want to parse .ts files as typescript, .tsx files as .tsx, etc.
			Loader: filenameToLoader(fileInfo.Filename),
		},
		Plugins: []api.Plugin{
			plugin,
		},
		// Must set bundle to true so that esbuild actually does resolutions.
		Bundle: true,
	}
	result := api.Build(options)
	if len(result.Errors) > 0 {
		// Inform users that some files couldn't be fully parsed.
		// No need to crash the program though.
		log.Printf("Encountered errors parsing source %v: %v\n", fileInfo.Filename, result.Errors)
	}

	return imports
}
