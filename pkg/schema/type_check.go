package schema

import (
	"context"
	"fmt"

	"github.com/authzed/spicedb/pkg/genutil/mapz"
	corev1 "github.com/authzed/spicedb/pkg/proto/core/v1"
)

const ellipsesRelation = "..."

// GetRecursiveTerminalTypesForRelation returns, for a given definition and relation, all the potential
// terminal subject type definition names of that relation.
func (ts *TypeSystem) GetRecursiveTerminalTypesForRelation(ctx context.Context, defName string, relationName string) ([]string, error) {
	seen := mapz.NewSet[string]()
	set, err := ts.getTypesForRelationInternal(ctx, defName, relationName, seen, false)
	if err != nil {
		return nil, err
	}
	return set.AsSlice(), nil
}

// GetFullRecursiveSubjectTypesForRelation returns, for a given definition and relation, all the potential
// terminal subject type definition names of that relation, as well as any relation subtypes (eg, `group#member`) that may occur.
func (ts *TypeSystem) GetFullRecursiveSubjectTypesForRelation(ctx context.Context, defName string, relationName string) ([]string, error) {
	seen := mapz.NewSet[string]()
	set, err := ts.getTypesForRelationInternal(ctx, defName, relationName, seen, true)
	if err != nil {
		return nil, err
	}
	return set.AsSlice(), nil
}

func (ts *TypeSystem) getTypesForRelationInternal(ctx context.Context, defName string, relationName string, seen *mapz.Set[string], addNonTerminals bool) (*mapz.Set[string], error) {
	id := fmt.Sprint(defName, "#", relationName)
	if seen.Has(id) {
		return nil, nil
	}
	seen.Add(id)
	def, err := ts.GetDefinition(ctx, defName)
	if err != nil {
		return nil, err
	}
	rel, ok := def.GetRelation(relationName)
	if !ok {
		return nil, asTypeError(NewRelationNotFoundErr(defName, relationName))
	}
	if rel.TypeInformation != nil {
		return ts.getTypesForInfo(ctx, defName, rel.TypeInformation, seen, addNonTerminals)
	} else if rel.UsersetRewrite != nil {
		return ts.getTypesForRewrite(ctx, defName, rel.UsersetRewrite, seen, addNonTerminals)
	}
	return nil, asTypeError(NewMissingAllowedRelationsErr(defName, relationName))
}

func (ts *TypeSystem) getTypesForInfo(ctx context.Context, defName string, rel *corev1.TypeInformation, seen *mapz.Set[string], addNonTerminals bool) (*mapz.Set[string], error) {
	out := mapz.NewSet[string]()
	for _, dr := range rel.GetAllowedDirectRelations() {
		if dr.GetRelation() == ellipsesRelation {
			out.Add(dr.GetNamespace())
		} else if dr.GetRelation() != "" {
			if addNonTerminals {
				out.Add(fmt.Sprintf("%s#%s", dr.GetNamespace(), dr.GetRelation()))
			}
			rest, err := ts.getTypesForRelationInternal(ctx, dr.GetNamespace(), dr.GetRelation(), seen, addNonTerminals)
			if err != nil {
				return nil, err
			}
			out.Merge(rest)
		} else {
			// It's a wildcard, so all things of that type count
			out.Add(dr.GetNamespace())
		}
	}
	return out, nil
}

func (ts *TypeSystem) getTypesForRewrite(ctx context.Context, defName string, rel *corev1.UsersetRewrite, seen *mapz.Set[string], addNonTerminals bool) (*mapz.Set[string], error) {
	out := mapz.NewSet[string]()

	// We're finding the union of all the things touched, regardless.
	toCheck := []*corev1.SetOperation{rel.GetUnion(), rel.GetIntersection(), rel.GetExclusion()}

	for _, op := range toCheck {
		if op == nil {
			continue
		}
		for _, child := range op.GetChild() {
			if computed := child.GetComputedUserset(); computed != nil {
				set, err := ts.getTypesForRelationInternal(ctx, defName, computed.GetRelation(), seen, addNonTerminals)
				if err != nil {
					return nil, err
				}
				out.Merge(set)
			}
			if rewrite := child.GetUsersetRewrite(); rewrite != nil {
				sub, err := ts.getTypesForRewrite(ctx, defName, rewrite, seen, addNonTerminals)
				if err != nil {
					return nil, err
				}
				out.Merge(sub)
			}
			if userset := child.GetTupleToUserset(); userset != nil {
				set, err := ts.getTypesForRelationInternal(ctx, defName, userset.GetTupleset().GetRelation(), seen, addNonTerminals)
				if err != nil {
					return nil, err
				}
				if set == nil {
					// We've already seen it.
					continue
				}
				for _, s := range set.AsSlice() {
					targets, err := ts.getTypesForRelationInternal(ctx, s, userset.GetComputedUserset().GetRelation(), seen, addNonTerminals)
					if err != nil {
						return nil, err
					}
					if targets == nil {
						// Already added
						continue
					}
					out.Merge(targets)
				}
			}
			if functioned := child.GetFunctionedTupleToUserset(); functioned != nil {
				set, err := ts.getTypesForRelationInternal(ctx, defName, functioned.GetTupleset().GetRelation(), seen, addNonTerminals)
				if err != nil {
					return nil, err
				}
				if set == nil {
					// We've already seen it.
					continue
				}
				for _, s := range set.AsSlice() {
					targets, err := ts.getTypesForRelationInternal(ctx, s, functioned.GetComputedUserset().GetRelation(), seen, addNonTerminals)
					if targets == nil {
						continue
					}
					if err != nil {
						return nil, err
					}
					out.Merge(targets)
				}
			}
		}
	}
	return out, nil
}
