package bolt

import (
	"fmt"
	"omnethdb/internal/memory"
	"slices"

	bbolt "go.etcd.io/bbolt"
)

func relationBucketKey(fromID string, kind memory.RelationType) []byte {
	return []byte(fmt.Sprintf("%s:%s", fromID, kind))
}

func relationReverseBucketKey(toID string, kind memory.RelationType) []byte {
	return []byte(fmt.Sprintf("%s:%s", toID, kind))
}

func loadIndexedRelatedIDs(tx *bbolt.Tx, memoryID string, kind memory.RelationType) ([]string, error) {
	outgoing, err := loadRelationIDsFromBucket(tx.Bucket(bucketRelations), relationBucketKey(memoryID, kind))
	if err != nil {
		return nil, err
	}
	inbound, err := loadInboundRelationIDs(tx, memoryID, kind)
	if err != nil {
		return nil, err
	}

	combined := append(slices.Clone(outgoing), inbound...)
	if len(combined) == 0 {
		return nil, nil
	}

	seen := make(map[string]struct{}, len(combined))
	deduped := make([]string, 0, len(combined))
	for _, id := range combined {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		deduped = append(deduped, id)
	}
	return deduped, nil
}

func loadInboundRelationIDs(tx *bbolt.Tx, memoryID string, kind memory.RelationType) ([]string, error) {
	return loadRelationIDsFromBucket(tx.Bucket(bucketRelationRefs), relationReverseBucketKey(memoryID, kind))
}
