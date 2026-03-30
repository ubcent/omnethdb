package hash

import (
	"context"
	"hash/fnv"
)

type Embedder struct {
	modelID    string
	dimensions int
}

func New(modelID string, dimensions int) Embedder {
	return Embedder{
		modelID:    modelID,
		dimensions: dimensions,
	}
}

func (e Embedder) Embed(_ context.Context, text string) ([]float32, error) {
	vec := make([]float32, e.dimensions)
	if e.dimensions == 0 {
		return vec, nil
	}

	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(e.modelID + "::" + text))
	hash := hasher.Sum64()
	for i := 0; i < e.dimensions; i++ {
		shift := uint((i % 8) * 8)
		value := float32((hash>>shift)&0xff) + 1
		vec[i] = value / 255
	}
	return vec, nil
}

func (e Embedder) Dimensions() int { return e.dimensions }

func (e Embedder) ModelID() string { return e.modelID }
