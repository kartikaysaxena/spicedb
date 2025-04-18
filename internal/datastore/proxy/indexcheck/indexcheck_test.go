package indexcheck

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/authzed/spicedb/pkg/datastore"
	"github.com/authzed/spicedb/pkg/datastore/options"
	"github.com/authzed/spicedb/pkg/datastore/queryshape"
)

func TestIndexCheckingMissingIndex(t *testing.T) {
	ds := fakeDatastore{indexesUsed: []string{"anotherindex"}}
	wrapped := WrapWithIndexCheckingDatastoreProxyIfApplicable(ds)
	require.NotNil(t, wrapped.(*indexcheckingProxy).delegate)

	headRev, err := ds.HeadRevision(context.Background())
	require.NoError(t, err)

	reader := wrapped.SnapshotReader(headRev)
	it, err := reader.QueryRelationships(context.Background(), datastore.RelationshipsFilter{
		OptionalResourceType:     "document",
		OptionalResourceIds:      []string{"somedoc"},
		OptionalResourceRelation: "viewer",
		OptionalSubjectsSelectors: []datastore.SubjectsSelector{
			{
				OptionalSubjectType: "user",
				OptionalSubjectIds:  []string{"tom"},
				RelationFilter:      datastore.SubjectRelationFilter{}.WithEllipsisRelation(),
			},
		},
	}, options.WithQueryShape(queryshape.CheckPermissionSelectDirectSubjects))
	require.NoError(t, err)

	for _, err := range it {
		require.ErrorContains(t, err, "expected index(es) [anotherindex] for query shape check-permission-select-direct-subjects not used: EXPLAIN IS FAKE")
	}
}

func TestIndexCheckingFoundIndex(t *testing.T) {
	ds := fakeDatastore{indexesUsed: []string{"testindex"}}
	wrapped := WrapWithIndexCheckingDatastoreProxyIfApplicable(ds)
	require.NotNil(t, wrapped.(*indexcheckingProxy).delegate)

	headRev, err := ds.HeadRevision(context.Background())
	require.NoError(t, err)

	reader := wrapped.SnapshotReader(headRev)
	it, err := reader.QueryRelationships(context.Background(), datastore.RelationshipsFilter{
		OptionalResourceType:     "document",
		OptionalResourceIds:      []string{"somedoc"},
		OptionalResourceRelation: "viewer",
		OptionalSubjectsSelectors: []datastore.SubjectsSelector{
			{
				OptionalSubjectType: "user",
				OptionalSubjectIds:  []string{"tom"},
				RelationFilter:      datastore.SubjectRelationFilter{}.WithEllipsisRelation(),
			},
		},
	}, options.WithQueryShape(queryshape.CheckPermissionSelectDirectSubjects))
	require.NoError(t, err)

	for _, err := range it {
		require.NoError(t, err)
	}
}
