package proxy

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/authzed/spicedb/pkg/datastore"
	"github.com/authzed/spicedb/pkg/datastore/options"
	"github.com/authzed/spicedb/pkg/datastore/revisionparsing"
	corev1 "github.com/authzed/spicedb/pkg/proto/core/v1"
)

func TestReplicatedReaderFallsbackToPrimary(t *testing.T) {
	primary := fakeDatastore{true, revisionparsing.MustParseRevisionForTest("2")}
	replica := fakeDatastore{false, revisionparsing.MustParseRevisionForTest("1")}

	replicated, err := NewReplicatedDatastore(primary, replica)
	require.NoError(t, err)

	// Try at revision 1, which should use the replica.
	reader := replicated.SnapshotReader(revisionparsing.MustParseRevisionForTest("1"))
	ns, err := reader.ListAllNamespaces(context.Background())
	require.NoError(t, err)
	require.Equal(t, 0, len(ns))

	require.False(t, reader.(*replicatedReader).chosePrimary)

	// Try at revision 2, which should use the primary.
	reader = replicated.SnapshotReader(revisionparsing.MustParseRevisionForTest("2"))
	ns, err = reader.ListAllNamespaces(context.Background())
	require.NoError(t, err)
	require.Equal(t, 0, len(ns))

	require.True(t, reader.(*replicatedReader).chosePrimary)
}

type fakeDatastore struct {
	isPrimary bool
	revision  datastore.Revision
}

func (f fakeDatastore) SnapshotReader(_ datastore.Revision) datastore.Reader {
	return fakeSnapshotReader{}
}

func (f fakeDatastore) ReadWriteTx(_ context.Context, _ datastore.TxUserFunc, _ ...options.RWTOptionsOption) (datastore.Revision, error) {
	return nil, nil
}

func (f fakeDatastore) OptimizedRevision(_ context.Context) (datastore.Revision, error) {
	return nil, nil
}

func (f fakeDatastore) HeadRevision(_ context.Context) (datastore.Revision, error) {
	return nil, nil
}

func (f fakeDatastore) CheckRevision(_ context.Context, rev datastore.Revision) error {
	if rev.GreaterThan(f.revision) {
		return datastore.NewInvalidRevisionErr(rev, datastore.CouldNotDetermineRevision)
	}

	return nil
}

func (f fakeDatastore) RevisionFromString(_ string) (datastore.Revision, error) {
	return nil, nil
}

func (f fakeDatastore) Watch(_ context.Context, _ datastore.Revision, _ datastore.WatchOptions) (<-chan *datastore.RevisionChanges, <-chan error) {
	return nil, nil
}

func (f fakeDatastore) ReadyState(_ context.Context) (datastore.ReadyState, error) {
	return datastore.ReadyState{}, nil
}

func (f fakeDatastore) Features(_ context.Context) (*datastore.Features, error) {
	return nil, nil
}

func (f fakeDatastore) Statistics(_ context.Context) (datastore.Stats, error) {
	return datastore.Stats{}, nil
}

func (f fakeDatastore) Close() error {
	return nil
}

type fakeSnapshotReader struct{}

func (fakeSnapshotReader) LookupNamespacesWithNames(_ context.Context, nsNames []string) ([]datastore.RevisionedDefinition[*corev1.NamespaceDefinition], error) {
	return nil, fmt.Errorf("not implemented")
}

func (fakeSnapshotReader) ReadNamespaceByName(_ context.Context, nsName string) (ns *corev1.NamespaceDefinition, lastWritten datastore.Revision, err error) {
	return nil, nil, fmt.Errorf("not implemented")
}

func (fakeSnapshotReader) LookupCaveatsWithNames(_ context.Context, names []string) ([]datastore.RevisionedDefinition[*corev1.CaveatDefinition], error) {
	return nil, fmt.Errorf("not implemented")
}

func (fakeSnapshotReader) ReadCaveatByName(_ context.Context, name string) (caveat *corev1.CaveatDefinition, lastWritten datastore.Revision, err error) {
	return nil, nil, fmt.Errorf("not implemented")
}

func (fakeSnapshotReader) ListAllCaveats(context.Context) ([]datastore.RevisionedDefinition[*corev1.CaveatDefinition], error) {
	return nil, fmt.Errorf("not implemented")
}

func (fakeSnapshotReader) ListAllNamespaces(context.Context) ([]datastore.RevisionedDefinition[*corev1.NamespaceDefinition], error) {
	return nil, nil
}

func (fakeSnapshotReader) QueryRelationships(context.Context, datastore.RelationshipsFilter, ...options.QueryOptionsOption) (datastore.RelationshipIterator, error) {
	return nil, fmt.Errorf("not implemented")
}

func (fakeSnapshotReader) ReverseQueryRelationships(context.Context, datastore.SubjectsFilter, ...options.ReverseQueryOptionsOption) (datastore.RelationshipIterator, error) {
	return nil, fmt.Errorf("not implemented")
}
