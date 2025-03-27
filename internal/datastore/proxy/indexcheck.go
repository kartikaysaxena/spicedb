package proxy

import (
	"context"
	"fmt"

	v1 "github.com/authzed/authzed-go/proto/authzed/api/v1"

	"github.com/authzed/spicedb/pkg/datastore"
	"github.com/authzed/spicedb/pkg/datastore/options"
	"github.com/authzed/spicedb/pkg/datastore/queryshape"
	"github.com/authzed/spicedb/pkg/genutil/mapz"
	core "github.com/authzed/spicedb/pkg/proto/core/v1"
	"github.com/authzed/spicedb/pkg/spiceerrors"
	"github.com/authzed/spicedb/pkg/tuple"
)

// NewIndexCheckingDatastoreProxy returns a datastore proxy that runs EXPLAIN ANALYZE on all
// relationships queries and ensures that the index(es) used within match those defined in the
// schema for the datastore.
func NewIndexCheckingDatastoreProxy(d datastore.SQLDatastore) datastore.Datastore {
	return &indexcheckingProxy{delegate: d}
}

// WrapWithIndexCheckingDatastoreProxyIfApplicable wraps the provided datastore with an
// index-checking proxy if the datastore is an SQLDatastore.
func WrapWithIndexCheckingDatastoreProxyIfApplicable(ds datastore.Datastore) datastore.Datastore {
	uds := datastore.UnwrapAs[datastore.SQLDatastore](ds)
	if uds == nil {
		return ds
	}
	return NewIndexCheckingDatastoreProxy(uds)
}

type indexcheckingProxy struct{ delegate datastore.SQLDatastore }

func (p *indexcheckingProxy) SnapshotReader(rev datastore.Revision) datastore.Reader {
	delegateReader := p.delegate.SnapshotReader(rev)
	return &indexcheckingReader{p.delegate, delegateReader}
}

func (p *indexcheckingProxy) ReadWriteTx(
	ctx context.Context,
	f datastore.TxUserFunc,
	opts ...options.RWTOptionsOption,
) (datastore.Revision, error) {
	return p.delegate.ReadWriteTx(ctx, func(ctx context.Context, delegateRWT datastore.ReadWriteTransaction) error {
		return f(ctx, &indexcheckingRWT{&indexcheckingReader{p.delegate, delegateRWT}, delegateRWT})
	}, opts...)
}

func (p *indexcheckingProxy) MetricsID() (string, error) {
	return p.delegate.MetricsID()
}

func (p *indexcheckingProxy) OptimizedRevision(ctx context.Context) (datastore.Revision, error) {
	return p.delegate.OptimizedRevision(ctx)
}

func (p *indexcheckingProxy) CheckRevision(ctx context.Context, revision datastore.Revision) error {
	return p.delegate.CheckRevision(ctx, revision)
}

func (p *indexcheckingProxy) HeadRevision(ctx context.Context) (datastore.Revision, error) {
	return p.delegate.HeadRevision(ctx)
}

func (p *indexcheckingProxy) RevisionFromString(serialized string) (datastore.Revision, error) {
	return p.delegate.RevisionFromString(serialized)
}

func (p *indexcheckingProxy) Watch(ctx context.Context, afterRevision datastore.Revision, options datastore.WatchOptions) (<-chan datastore.RevisionChanges, <-chan error) {
	return p.delegate.Watch(ctx, afterRevision, options)
}

func (p *indexcheckingProxy) Features(ctx context.Context) (*datastore.Features, error) {
	return p.delegate.Features(ctx)
}

func (p *indexcheckingProxy) OfflineFeatures() (*datastore.Features, error) {
	return p.delegate.OfflineFeatures()
}

func (p *indexcheckingProxy) Statistics(ctx context.Context) (datastore.Stats, error) {
	return p.delegate.Statistics(ctx)
}

func (p *indexcheckingProxy) Unwrap() datastore.Datastore {
	return p.delegate
}

func (p *indexcheckingProxy) ReadyState(ctx context.Context) (datastore.ReadyState, error) {
	return p.delegate.ReadyState(ctx)
}

func (p *indexcheckingProxy) Close() error { return p.delegate.Close() }

type indexcheckingReader struct {
	parent   datastore.SQLDatastore
	delegate datastore.Reader
}

func (r *indexcheckingReader) CountRelationships(ctx context.Context, name string) (int, error) {
	return r.delegate.CountRelationships(ctx, name)
}

func (r *indexcheckingReader) LookupCounters(ctx context.Context) ([]datastore.RelationshipCounter, error) {
	return r.delegate.LookupCounters(ctx)
}

func (r *indexcheckingReader) ReadCaveatByName(ctx context.Context, name string) (*core.CaveatDefinition, datastore.Revision, error) {
	return r.delegate.ReadCaveatByName(ctx, name)
}

func (r *indexcheckingReader) LookupCaveatsWithNames(ctx context.Context, caveatNames []string) ([]datastore.RevisionedCaveat, error) {
	return r.delegate.LookupCaveatsWithNames(ctx, caveatNames)
}

func (r *indexcheckingReader) ListAllCaveats(ctx context.Context) ([]datastore.RevisionedCaveat, error) {
	return r.delegate.ListAllCaveats(ctx)
}

func (r *indexcheckingReader) ListAllNamespaces(ctx context.Context) ([]datastore.RevisionedNamespace, error) {
	return r.delegate.ListAllNamespaces(ctx)
}

func (r *indexcheckingReader) LookupNamespacesWithNames(ctx context.Context, nsNames []string) ([]datastore.RevisionedNamespace, error) {
	return r.delegate.LookupNamespacesWithNames(ctx, nsNames)
}

func (r *indexcheckingReader) ReadNamespaceByName(ctx context.Context, nsName string) (*core.NamespaceDefinition, datastore.Revision, error) {
	return r.delegate.ReadNamespaceByName(ctx, nsName)
}

func (r *indexcheckingReader) mustEnsureIndexes(ctx context.Context, sql string, args []any, shape queryshape.Shape, explain string, expectedIndexes options.SQLIndexInformation) error {
	// If no indexes are expected, there is nothing to check.
	if len(expectedIndexes.ExpectedIndexNames) == 0 {
		return nil
	}

	parsed, err := r.parent.ParseExplain(explain)
	if err != nil {
		return fmt.Errorf("failed to parse explain output: %w", err)
	}

	// If an index is not used (perhaps because the data is too small), the query is still valid.
	if len(parsed.IndexesUsed) == 0 {
		return nil
	}

	// Otherwise, ensure the index used is one of the expected indexes.
	indexesUsed := mapz.NewSet(parsed.IndexesUsed...)
	indexesExpected := mapz.NewSet(expectedIndexes.ExpectedIndexNames...)
	if indexesExpected.Intersect(indexesUsed).IsEmpty() {
		return fmt.Errorf("expected index(es) %v for query shape %v not used: %s", expectedIndexes.ExpectedIndexNames, shape, explain)
	}

	return nil
}

func (r *indexcheckingReader) QueryRelationships(ctx context.Context, filter datastore.RelationshipsFilter, opts ...options.QueryOptionsOption) (datastore.RelationshipIterator, error) {
	queryOpts := options.NewQueryOptionsWithOptions(opts...)
	if err := validateQueryShape(queryOpts.QueryShape, filter); err != nil {
		return nil, err
	}

	opts = append(opts, options.WithSQLExplainCallbackForTest(r.mustEnsureIndexes))
	return r.delegate.QueryRelationships(ctx, filter, opts...)
}

func (r *indexcheckingReader) ReverseQueryRelationships(ctx context.Context, subjectsFilter datastore.SubjectsFilter, opts ...options.ReverseQueryOptionsOption) (datastore.RelationshipIterator, error) {
	queryOpts := options.NewReverseQueryOptionsWithOptions(opts...)
	if err := validateReverseQueryShape(queryOpts.QueryShapeForReverse, subjectsFilter, queryOpts); err != nil {
		return nil, err
	}

	opts = append(opts, options.WithSQLExplainCallbackForTestForReverse(r.mustEnsureIndexes))
	return r.delegate.ReverseQueryRelationships(ctx, subjectsFilter, opts...)
}

type indexcheckingRWT struct {
	*indexcheckingReader
	delegate datastore.ReadWriteTransaction
}

func (rwt *indexcheckingRWT) RegisterCounter(ctx context.Context, name string, filter *core.RelationshipFilter) error {
	return rwt.delegate.RegisterCounter(ctx, name, filter)
}

func (rwt *indexcheckingRWT) UnregisterCounter(ctx context.Context, name string) error {
	return rwt.delegate.UnregisterCounter(ctx, name)
}

func (rwt *indexcheckingRWT) StoreCounterValue(ctx context.Context, name string, value int, computedAtRevision datastore.Revision) error {
	return rwt.delegate.StoreCounterValue(ctx, name, value, computedAtRevision)
}

func (rwt *indexcheckingRWT) WriteCaveats(ctx context.Context, caveats []*core.CaveatDefinition) error {
	return rwt.delegate.WriteCaveats(ctx, caveats)
}

func (rwt *indexcheckingRWT) DeleteCaveats(ctx context.Context, names []string) error {
	return rwt.delegate.DeleteCaveats(ctx, names)
}

func (rwt *indexcheckingRWT) WriteRelationships(ctx context.Context, mutations []tuple.RelationshipUpdate) error {
	return rwt.delegate.WriteRelationships(ctx, mutations)
}

func (rwt *indexcheckingRWT) WriteNamespaces(ctx context.Context, newConfigs ...*core.NamespaceDefinition) error {
	return rwt.delegate.WriteNamespaces(ctx, newConfigs...)
}

func (rwt *indexcheckingRWT) DeleteNamespaces(ctx context.Context, nsNames ...string) error {
	return rwt.delegate.DeleteNamespaces(ctx, nsNames...)
}

func (rwt *indexcheckingRWT) DeleteRelationships(ctx context.Context, filter *v1.RelationshipFilter, options ...options.DeleteOptionsOption) (uint64, bool, error) {
	return rwt.delegate.DeleteRelationships(ctx, filter, options...)
}

func (rwt *indexcheckingRWT) BulkLoad(ctx context.Context, iter datastore.BulkWriteRelationshipSource) (uint64, error) {
	return rwt.delegate.BulkLoad(ctx, iter)
}

var (
	_ datastore.Datastore            = (*indexcheckingProxy)(nil)
	_ datastore.Reader               = (*indexcheckingReader)(nil)
	_ datastore.ReadWriteTransaction = (*indexcheckingRWT)(nil)
)

func validateQueryShape(queryShape queryshape.Shape, filter datastore.RelationshipsFilter) error {
	switch queryShape {
	case queryshape.CheckPermissionSelectDirectSubjects:
		if filter.OptionalCaveatName != "" {
			return fmt.Errorf("optional caveats not supported for CheckPermissionSelectDirectSubjects")
		}

		if filter.OptionalResourceType == "" {
			return fmt.Errorf("optional resource type required for CheckPermissionSelectDirectSubjects")
		}

		if len(filter.OptionalResourceIds) == 0 {
			return fmt.Errorf("optional resource ids required for CheckPermissionSelectDirectSubjects")
		}

		if filter.OptionalResourceRelation == "" {
			return fmt.Errorf("optional resource relation required for CheckPermissionSelectDirectSubjects")
		}

		if len(filter.OptionalSubjectsSelectors) == 0 {
			return fmt.Errorf("optional subjects selectors required for CheckPermissionSelectDirectSubjects")
		}

		for _, subjectSelector := range filter.OptionalSubjectsSelectors {
			if subjectSelector.OptionalSubjectType == "" {
				return fmt.Errorf("optional subject type required for CheckPermissionSelectDirectSubjects")
			}

			if len(subjectSelector.OptionalSubjectIds) == 0 {
				return fmt.Errorf("optional subject ids required for CheckPermissionSelectDirectSubjects")
			}
		}

		return nil

	case queryshape.CheckPermissionSelectIndirectSubjects:
		if filter.OptionalCaveatName != "" {
			return fmt.Errorf("optional caveats not supported for CheckPermissionSelectIndirectSubjects")
		}

		if filter.OptionalResourceType == "" {
			return fmt.Errorf("optional resource type required for CheckPermissionSelectIndirectSubjects")
		}

		if len(filter.OptionalResourceIds) == 0 {
			return fmt.Errorf("optional resource ids required for CheckPermissionSelectIndirectSubjects")
		}

		if filter.OptionalResourceRelation == "" {
			return fmt.Errorf("optional resource relation required for CheckPermissionSelectIndirectSubjects")
		}

		if len(filter.OptionalSubjectsSelectors) == 0 {
			return fmt.Errorf("optional subjects selectors required for CheckPermissionSelectIndirectSubjects")
		}

		for _, subjectSelector := range filter.OptionalSubjectsSelectors {
			if subjectSelector.OptionalSubjectType != "" {
				return fmt.Errorf("optional subject type required for CheckPermissionSelectIndirectSubjects")
			}

			if len(subjectSelector.OptionalSubjectIds) != 0 {
				return fmt.Errorf("no optional subject ids allowed for CheckPermissionSelectIndirectSubjects")
			}

			if subjectSelector.RelationFilter.IsEmpty() {
				return fmt.Errorf("relation filter required for CheckPermissionSelectIndirectSubjects")
			}
		}

		return nil

	case queryshape.AllSubjectsForResources:
		if filter.OptionalCaveatName != "" {
			return fmt.Errorf("optional caveats not supported for AllSubjectsForResources")
		}

		if filter.OptionalResourceType == "" {
			return fmt.Errorf("optional resource type required for AllSubjectsForResources")
		}

		if len(filter.OptionalResourceIds) == 0 {
			return fmt.Errorf("optional resource ids required for AllSubjectsForResources")
		}

		if filter.OptionalResourceRelation == "" {
			return fmt.Errorf("optional resource relation required for AllSubjectsForResources")
		}

		if len(filter.OptionalSubjectsSelectors) != 0 {
			return fmt.Errorf("no optional subjects selectors allowed for AllSubjectsForResources")
		}

		return nil

	case queryshape.FindResourceOfType:
		if filter.OptionalResourceType == "" {
			return fmt.Errorf("optional resource type required for FindResourceOfType")
		}

		if len(filter.OptionalResourceIds) != 0 {
			return fmt.Errorf("no optional resource ids allowed for FindResourceOfType")
		}

		if filter.OptionalResourceRelation != "" {
			return fmt.Errorf("no optional resource relation allowed for FindResourceOfType")
		}

		if len(filter.OptionalSubjectsSelectors) != 0 {
			return fmt.Errorf("no optional subjects selectors allowed for FindResourceOfType")
		}

		return nil

	case queryshape.Varying:
		// Nothing to validate.
		return nil

	case queryshape.Unspecified:
		fallthrough

	case "":
		return spiceerrors.MustBugf("query shape must be specified")

	default:
		return fmt.Errorf("unsupported query shape: %s", queryShape)
	}
}

func validateReverseQueryShape(queryShape queryshape.Shape, subjectFilter datastore.SubjectsFilter, queryOpts *options.ReverseQueryOptions) error {
	switch queryShape {
	case queryshape.MatchingResourcesForSubject:
		if subjectFilter.SubjectType == "" {
			return fmt.Errorf("subject type required for MatchingResourcesForSubject")
		}

		if len(subjectFilter.OptionalSubjectIds) == 0 {
			return fmt.Errorf("subject ids required for MatchingResourcesForSubject")
		}

		if queryOpts.ResRelation == nil {
			return fmt.Errorf("resource relation required for MatchingResourcesForSubject")
		}

		if queryOpts.ResRelation.Namespace == "" {
			return fmt.Errorf("resource relation namespace required for MatchingResourcesForSubject")
		}

		if queryOpts.ResRelation.Relation == "" {
			return fmt.Errorf("resource relation required for MatchingResourcesForSubject")
		}

		return nil

	case queryshape.FindSubjectOfType:
		if subjectFilter.SubjectType == "" {
			return fmt.Errorf("subject type required for FindSubjectOfType")
		}

		if len(subjectFilter.OptionalSubjectIds) != 0 {
			return fmt.Errorf("no optional subject ids allowed for FindSubjectOfType")
		}

		if queryOpts.ResRelation != nil {
			return fmt.Errorf("no resource relation allowed for FindSubjectOfType")
		}

		return nil

	case queryshape.Varying:
		// Nothing to validate.
		return nil

	case queryshape.Unspecified:
		fallthrough

	case "":
		return spiceerrors.MustBugf("query shape must be specified")

	default:
		return fmt.Errorf("unsupported query shape: %s", queryShape)
	}
}
