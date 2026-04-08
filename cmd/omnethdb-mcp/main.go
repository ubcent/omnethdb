package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	omnethdb "omnethdb"
	hashembedder "omnethdb/embedders/hash"
	"omnethdb/internal/mcp"
)

const (
	defaultWorkspace = "."
	defaultModelID   = "builtin/hash-embedder-v1"
	defaultDimension = 256
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if len(os.Args) == 2 {
		switch os.Args[1] {
		case "version", "-v", "--version":
			printVersion()
			return
		}
	}

	fs := flag.NewFlagSet("omnethdb-mcp", flag.ExitOnError)
	workspace := fs.String("workspace", defaultWorkspace, "workspace root")
	versionFlag := fs.Bool("version", false, "print build version information")
	if err := fs.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if *versionFlag {
		printVersion()
		return
	}

	open := func() (*omnethdb.Store, *omnethdb.RuntimeConfig, error) {
		store, layout, err := omnethdb.OpenWorkspace(*workspace)
		if err != nil {
			return nil, nil, err
		}
		cfg, err := omnethdb.LoadRuntimeConfig(layout.ConfigPath)
		if err != nil {
			_ = store.Close()
			return nil, nil, err
		}
		for spaceID, settings := range cfg.Spaces {
			if settings.Embedder.ModelID == "" || settings.Embedder.Dimensions <= 0 {
				continue
			}
			_ = spaceID
			store.RegisterEmbedder(hashembedder.New(settings.Embedder.ModelID, settings.Embedder.Dimensions))
		}
		return store, cfg, nil
	}

	server := mcp.NewServer("omnethdb-mcp", version, mcp.NewOmnethDBTools(open))
	if err := server.Serve(context.Background(), os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func printVersion() {
	fmt.Printf("omnethdb-mcp version=%s commit=%s date=%s\n", version, commit, date)
}
