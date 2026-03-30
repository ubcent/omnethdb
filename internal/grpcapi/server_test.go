package grpcapi

import (
	"context"
	"net"
	"path/filepath"
	"testing"

	omnethdb "omnethdb"
	omnethdbv1 "omnethdb/gen/omnethdb/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

type testEmbedder struct {
	modelID    string
	dimensions int
}

func (e testEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	out := make([]float32, e.dimensions)
	for i := range out {
		out[i] = float32(len(text)+i+1) / 100
	}
	return out, nil
}

func (e testEmbedder) Dimensions() int { return e.dimensions }
func (e testEmbedder) ModelID() string { return e.modelID }

func TestGRPCAPIEndToEnd(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "memory.db")
	store, err := omnethdb.Open(path)
	if err != nil {
		t.Fatalf("Open returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	cfg := &omnethdb.RuntimeConfig{
		Spaces: map[string]omnethdb.RuntimeSpaceSettings{
			"repo:company/app": {
				Embedder: omnethdb.RuntimeEmbedderConfig{
					ModelID:    "builtin/hash-embedder-v1",
					Dimensions: 256,
				},
			},
		},
	}
	store.RegisterEmbedder(testEmbedder{modelID: "builtin/hash-embedder-v1", dimensions: 256})

	listener := bufconn.Listen(1 << 20)
	server := grpc.NewServer()
	omnethdbv1.RegisterOmnethDBServer(server, NewServer(store, cfg))
	t.Cleanup(server.Stop)
	go func() { _ = server.Serve(listener) }()

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("NewClient returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	client := omnethdbv1.NewOmnethDBClient(conn)

	if _, err := client.InitSpace(context.Background(), &omnethdbv1.InitSpaceRequest{
		SpaceId: "repo:company/app",
	}); err != nil {
		t.Fatalf("InitSpace returned unexpected error: %v", err)
	}

	root, err := client.Remember(context.Background(), &omnethdbv1.RememberRequest{
		SpaceId:    "repo:company/app",
		Content:    "payments use cursor pagination",
		Kind:       omnethdbv1.MemoryKind_MEMORY_KIND_STATIC,
		Actor:      &omnethdbv1.Actor{Id: "user:alice", Kind: omnethdbv1.ActorKind_ACTOR_KIND_HUMAN},
		Confidence: 1.0,
		Relations:  &omnethdbv1.MemoryRelations{},
	})
	if err != nil {
		t.Fatalf("Remember returned unexpected error: %v", err)
	}

	updated, err := client.Remember(context.Background(), &omnethdbv1.RememberRequest{
		SpaceId:    "repo:company/app",
		Content:    "payments use signed cursor pagination",
		Kind:       omnethdbv1.MemoryKind_MEMORY_KIND_STATIC,
		Actor:      &omnethdbv1.Actor{Id: "user:alice", Kind: omnethdbv1.ActorKind_ACTOR_KIND_HUMAN},
		Confidence: 1.0,
		IfLatestId: root.GetId(),
		Relations: &omnethdbv1.MemoryRelations{
			Updates: []string{root.GetId()},
		},
	})
	if err != nil {
		t.Fatalf("updated Remember returned unexpected error: %v", err)
	}

	recall, err := client.Recall(context.Background(), &omnethdbv1.RecallRequest{
		SpaceIds: []string{"repo:company/app"},
		Query:    "signed cursor pagination",
		TopK:     5,
	})
	if err != nil {
		t.Fatalf("Recall returned unexpected error: %v", err)
	}
	if len(recall.GetMemories()) != 1 || recall.GetMemories()[0].GetMemory().GetId() != updated.GetId() {
		t.Fatalf("unexpected recall result: %#v", recall)
	}

	lineage, err := client.GetLineage(context.Background(), &omnethdbv1.GetLineageRequest{RootId: root.GetId()})
	if err != nil {
		t.Fatalf("GetLineage returned unexpected error: %v", err)
	}
	if len(lineage.GetMemories()) != 2 {
		t.Fatalf("expected two lineage versions, got %#v", lineage)
	}
}
