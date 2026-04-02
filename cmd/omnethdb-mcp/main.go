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

func main() {
	fs := flag.NewFlagSet("omnethdb-mcp", flag.ExitOnError)
	workspace := fs.String("workspace", defaultWorkspace, "workspace root")
	if err := fs.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
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

	server := mcp.NewServer("omnethdb-mcp", "0.1.0", mcp.NewOmnethDBTools(open))
	if err := server.Serve(context.Background(), os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
