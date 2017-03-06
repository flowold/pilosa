package server

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/pilosa/pilosa"
)

// Version and BuildTime hold the version/build time information passed in at compile time.
var (
	Version   string
	BuildTime string
)

func init() {
	if Version == "" {
		Version = "v0.0.0"
	}
	if BuildTime == "" {
		BuildTime = "not recorded"
	}

	rand.Seed(time.Now().UTC().UnixNano())
}

const (
	// DefaultDataDir is the default data directory.
	DefaultDataDir = "~/.pilosa"
)

// Main represents the main program execution.
type Main struct {
	Server *pilosa.Server

	// Configuration options.
	ConfigPath string
	Config     *pilosa.Config

	// Profiling options.
	CPUProfile string
	CPUTime    time.Duration

	// Standard input/output
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// NewMain returns a new instance of Main.
func NewMain() *Main {
	return &Main{
		Server: pilosa.NewServer(),
		Config: pilosa.NewConfig(),

		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

// Run executes the main program execution.
func (m *Main) Run(args ...string) error {
	// Notify user of config file.
	if m.ConfigPath != "" {
		fmt.Fprintf(m.Stdout, "Using config: %s\n", m.ConfigPath)
	}

	// Setup logging output.
	m.Server.LogOutput = m.Stderr

	// Configure index.
	fmt.Fprintf(m.Stderr, "Using data from: %s\n", m.Config.DataDir)
	m.Server.Index.Path = m.Config.DataDir
	m.Server.Index.Stats = pilosa.NewExpvarStatsClient()

	// Build cluster from config file.
	m.Server.Host = m.Config.Host
	m.Server.Cluster = m.Config.PilosaCluster()

	// Set configuration options.
	m.Server.AntiEntropyInterval = time.Duration(m.Config.AntiEntropy.Interval)

	// Initialize server.
	if err := m.Server.Open(); err != nil {
		return err
	}

	fmt.Fprintf(m.Stderr, "Listening as http://%s\n", m.Server.Host)

	return nil
}

// Close shuts down the server.
func (m *Main) Close() error {
	return m.Server.Close()
}

// ParseFlags parses command line flags from args.
func (m *Main) SetupConfig(args []string) error {
	// Load config, if specified.
	if m.ConfigPath != "" {
		if _, err := toml.DecodeFile(m.ConfigPath, &m.Config); err != nil {
			return err
		}
	}

	// Use default data directory if one is not specified.
	if m.Config.DataDir == "" {
		m.Config.DataDir = DefaultDataDir
	}

	// Expand home directory.
	prefix := "~" + string(filepath.Separator)
	if strings.HasPrefix(m.Config.DataDir, prefix) {
		//	u, err := user.Current()
		HomeDir := os.Getenv("HOME")
		/*if err != nil {
			return err
		} else*/if HomeDir == "" {
			return errors.New("data directory not specified and no home dir available")
		}
		m.Config.DataDir = filepath.Join(HomeDir, strings.TrimPrefix(m.Config.DataDir, prefix))
	}

	return nil
}
