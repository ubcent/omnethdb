package grpcapi

import (
	"context"
	"net"
	"path/filepath"
	"reflect"
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

func TestGRPCAdvisoryCurationMatchesStore(t *testing.T) {
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
	ctx := context.Background()

	if _, err := client.InitSpace(ctx, &omnethdbv1.InitSpaceRequest{
		SpaceId: "repo:company/app",
	}); err != nil {
		t.Fatalf("InitSpace returned unexpected error: %v", err)
	}

	for _, req := range []*omnethdbv1.RememberRequest{
		{
			SpaceId:    "repo:company/app",
			Content:    "repository requires reviewed migrations",
			Kind:       omnethdbv1.MemoryKind_MEMORY_KIND_EPISODIC,
			Actor:      &omnethdbv1.Actor{Id: "agent:one", Kind: omnethdbv1.ActorKind_ACTOR_KIND_AGENT},
			Confidence: 0.8,
			Relations:  &omnethdbv1.MemoryRelations{},
		},
		{
			SpaceId:    "repo:company/app",
			Content:    "team requires reviewed migrations",
			Kind:       omnethdbv1.MemoryKind_MEMORY_KIND_EPISODIC,
			Actor:      &omnethdbv1.Actor{Id: "agent:two", Kind: omnethdbv1.ActorKind_ACTOR_KIND_AGENT},
			Confidence: 0.82,
			Relations:  &omnethdbv1.MemoryRelations{},
		},
		{
			SpaceId:    "repo:company/app",
			Content:    "migration policy requires review approval",
			Kind:       omnethdbv1.MemoryKind_MEMORY_KIND_EPISODIC,
			Actor:      &omnethdbv1.Actor{Id: "user:alice", Kind: omnethdbv1.ActorKind_ACTOR_KIND_HUMAN},
			Confidence: 0.95,
			Relations:  &omnethdbv1.MemoryRelations{},
		},
		{
			SpaceId:    "repo:company/app",
			Content:    "payments rollback investigation remained noisy",
			Kind:       omnethdbv1.MemoryKind_MEMORY_KIND_EPISODIC,
			Actor:      &omnethdbv1.Actor{Id: "agent:three", Kind: omnethdbv1.ActorKind_ACTOR_KIND_AGENT},
			Confidence: 0.6,
			Relations:  &omnethdbv1.MemoryRelations{},
		},
	} {
		if _, err := client.Remember(ctx, req); err != nil {
			t.Fatalf("Remember returned unexpected error: %v", err)
		}
	}

	storeSynthesis, err := store.GetSynthesisCandidates(omnethdb.SynthesisCandidatesRequest{
		SpaceID: "repo:company/app",
	})
	if err != nil {
		t.Fatalf("GetSynthesisCandidates returned unexpected error: %v", err)
	}
	grpcSynthesis, err := client.SynthesisCandidates(ctx, &omnethdbv1.SynthesisCandidatesRequest{
		SpaceId: "repo:company/app",
	})
	if err != nil {
		t.Fatalf("SynthesisCandidates returned unexpected error: %v", err)
	}
	if int(grpcSynthesis.GetLiveEpisodicCount()) != storeSynthesis.LiveEpisodicCount {
		t.Fatalf("expected matching live episodic count, grpc=%d store=%d", grpcSynthesis.GetLiveEpisodicCount(), storeSynthesis.LiveEpisodicCount)
	}
	if !reflect.DeepEqual(synthesisSignatureFromProto(grpcSynthesis.GetCandidates()), synthesisSignatureFromStore(storeSynthesis.Candidates)) {
		t.Fatalf("expected grpc synthesis candidates to match store, grpc=%v store=%v", synthesisSignatureFromProto(grpcSynthesis.GetCandidates()), synthesisSignatureFromStore(storeSynthesis.Candidates))
	}

	storePromotion, err := store.GetPromotionSuggestions(omnethdb.PromotionSuggestionsRequest{
		SpaceID:             "repo:company/app",
		MinObservationCount: 2,
		MinDistinctActors:   2,
		MinDistinctWindows:  1,
		MinCumulativeScore:  2.5,
	})
	if err != nil {
		t.Fatalf("GetPromotionSuggestions returned unexpected error: %v", err)
	}
	grpcPromotion, err := client.PromotionSuggestions(ctx, &omnethdbv1.PromotionSuggestionsRequest{
		SpaceId:             "repo:company/app",
		MinObservationCount: 2,
		MinDistinctActors:   2,
		MinDistinctWindows:  1,
		MinCumulativeScore:  2.5,
	})
	if err != nil {
		t.Fatalf("PromotionSuggestions returned unexpected error: %v", err)
	}
	if int(grpcPromotion.GetLiveEpisodicCount()) != storePromotion.LiveEpisodicCount {
		t.Fatalf("expected matching live episodic count, grpc=%d store=%d", grpcPromotion.GetLiveEpisodicCount(), storePromotion.LiveEpisodicCount)
	}
	if !reflect.DeepEqual(promotionSignatureFromProto(grpcPromotion.GetSuggestions()), promotionSignatureFromStore(storePromotion.Suggestions)) {
		t.Fatalf("expected grpc promotion suggestions to match store, grpc=%v store=%v", promotionSignatureFromProto(grpcPromotion.GetSuggestions()), promotionSignatureFromStore(storePromotion.Suggestions))
	}
}

func synthesisSignatureFromStore(items []omnethdb.SynthesisCandidate) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.CandidateID+"|"+item.SuggestedAction+"|"+joinReasonCodes(item.ReasonCodes)+"|"+firstMemberIDFromStore(item.Members))
	}
	return out
}

func synthesisSignatureFromProto(items []*omnethdbv1.SynthesisCandidate) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.GetCandidateId()+"|"+item.GetSuggestedAction()+"|"+joinReasonCodes(item.GetReasonCodes())+"|"+firstMemberIDFromProto(item.GetMembers()))
	}
	return out
}

func promotionSignatureFromStore(items []omnethdb.PromotionSuggestion) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.MemoryID+"|"+item.SuggestedAction+"|"+joinReasonCodes(item.ReasonCodes)+"|"+item.Memory.MemoryID)
	}
	return out
}

func promotionSignatureFromProto(items []*omnethdbv1.PromotionSuggestion) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		memoryID := ""
		if item.GetMemory() != nil {
			memoryID = item.GetMemory().GetMemoryId()
		}
		out = append(out, item.GetMemoryId()+"|"+item.GetSuggestedAction()+"|"+joinReasonCodes(item.GetReasonCodes())+"|"+memoryID)
	}
	return out
}

func joinReasonCodes(items []string) string {
	if len(items) == 0 {
		return ""
	}
	out := items[0]
	for _, item := range items[1:] {
		out += "," + item
	}
	return out
}

func firstMemberIDFromStore(items []omnethdb.SynthesisCandidateMember) string {
	if len(items) == 0 {
		return ""
	}
	return items[0].MemoryID
}

func firstMemberIDFromProto(items []*omnethdbv1.SynthesisCandidateMember) string {
	if len(items) == 0 || items[0] == nil {
		return ""
	}
	return items[0].GetMemoryId()
}
