package gazelle

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
)

var (
	tsStdin  io.Writer
	tsStdout io.Reader
	tsMutex  sync.Mutex
)

const TS_API_BIN = "ts_api"
const TS_API_RUNFILES_PATH = "gazelle/_" + TS_API_BIN + "_launcher.sh"

// init starts the typescript api server as a background subprocess
func init() {
	runfile, err := bazel.Runfile(TS_API_RUNFILES_PATH)
	if err != nil {
		log.Printf("failed to initialize typescript: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	ctx, parserCancel := context.WithTimeout(ctx, time.Minute*5)
	cmd := exec.CommandContext(ctx, runfile)

	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Printf("failed to initialize typescript: %v\n", err)
		os.Exit(1)
	}
	tsStdin = stdin

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("failed to initialize typescript: %v\n", err)
		os.Exit(1)
	}
	tsStdout = stdout

	if err := cmd.Start(); err != nil {
		log.Printf("failed to initialize typescript: %v\n", err)
		os.Exit(1)
	}

	go func() {
		defer parserCancel()
		if err := cmd.Wait(); err != nil {
			log.Printf("failed to wait for typescript: %v\n", err)
			os.Exit(1)
		}
	}()
}

type parserResponse struct {
	Options   TsCompilerOptions `json:"options"`
	Filenames []string          `json:"fileNames"`
}
type TsCompilerOptions struct {
	BaseUrl   string            `json:"baseUrl"`
	RootDir   string            `json:"rootDir"`
}

// ParseOptions asks TypeScript to load a tsconfig.json file and return the compilerOptions config
func ParseOptions(tsconfigPath string) (*TsCompilerOptions, error) {
	// Ensure no concurrent requests to the node subprocess
	tsMutex.Lock()
	defer tsMutex.Unlock()

	req := map[string]interface{}{
		"command":        "parseOptions",
		"path": 	tsconfigPath,
	}

	encoder := json.NewEncoder(tsStdin)
	if err := encoder.Encode(&req); err != nil {
		return nil, fmt.Errorf("failed to parse: %w", err)
	}

	reader := bufio.NewReader(tsStdout)
	data, err := reader.ReadBytes(0)
	if err != nil {
		return nil, fmt.Errorf("failed to parse: %w", err)
	}
	data = data[:len(data)-1]
	var allRes parserResponse
	if err := json.Unmarshal(data, &allRes); err != nil {
		return nil, fmt.Errorf("failed to parse: %w", err)
	}
	return &allRes.Options, nil
}
