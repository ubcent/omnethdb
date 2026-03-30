package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	omnethdb "omnethdb"
	hashembedder "omnethdb/embedders/hash"
)

func main() {
	workspace, err := os.MkdirTemp("", "omnethdb-basic-example-*")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(workspace)

	store, layout, err := omnethdb.OpenWorkspace(workspace)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	embedder := hashembedder.New("builtin/hash-embedder-v1", 256)
	store.RegisterEmbedder(embedder)

	spaceID := "repo:company/app"
	init := omnethdb.SpaceInit{
		DefaultWeight: 1.0,
		HalfLifeDays:  30,
		WritePolicy:   omnethdb.DefaultSpaceWritePolicy(),
	}

	if _, err := store.EnsureSpace(spaceID, embedder, init); err != nil {
		log.Fatal(err)
	}

	v1, err := store.Remember(omnethdb.MemoryInput{
		SpaceID:    spaceID,
		Content:    "payments use cursor pagination",
		Kind:       omnethdb.KindStatic,
		Actor:      omnethdb.Actor{ID: "user:alice", Kind: omnethdb.ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		log.Fatal(err)
	}

	v2, err := store.Remember(omnethdb.MemoryInput{
		SpaceID:    spaceID,
		Content:    "payments use signed cursor pagination",
		Kind:       omnethdb.KindStatic,
		Actor:      omnethdb.Actor{ID: "user:alice", Kind: omnethdb.ActorHuman},
		Confidence: 1.0,
		IfLatestID: &v1.ID,
		Relations:  omnethdb.MemoryRelations{Updates: []string{v1.ID}},
	})
	if err != nil {
		log.Fatal(err)
	}

	results, err := store.Recall(omnethdb.RecallRequest{
		SpaceIDs: []string{spaceID},
		Query:    "signed cursor pagination",
		TopK:     5,
	})
	if err != nil {
		log.Fatal(err)
	}

	lineage, err := store.GetLineage(v1.ID)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("workspace:", workspace)
	fmt.Println("config:", layout.ConfigPath)
	fmt.Println("data:", layout.DataPath)
	fmt.Println("latest memory id:", v2.ID)
	fmt.Println("recall results:", len(results))
	fmt.Println("lineage versions:", len(lineage))

	if len(results) > 0 {
		fmt.Println("top recall content:", results[0].Content)
	}
	fmt.Println("example complete")
	fmt.Println("tip: inspect the workspace before process exit by copying", filepath.Clean(workspace))
}
