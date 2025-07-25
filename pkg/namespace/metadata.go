package namespace

import (
	"google.golang.org/protobuf/types/known/anypb"

	core "github.com/authzed/spicedb/pkg/proto/core/v1"
	iv1 "github.com/authzed/spicedb/pkg/proto/impl/v1"
)

// WithSourcePosition defines an interface for proto messages in core with SourcePosition information attached.
type WithSourcePosition interface {
	GetSourcePosition() *core.SourcePosition
}

// userDefinedMetadataTypeUrls are the type URLs of any user-defined metadata found
// in the namespace proto. If placed here, FilterUserDefinedMetadataInPlace will remove the
// metadata when called on the namespace.
var userDefinedMetadataTypeUrls = map[string]struct{}{
	"type.googleapis.com/impl.v1.DocComment": {},
}

// FilterUserDefinedMetadataInPlace removes user-defined metadata (e.g. comments) from the given namespace
// *in place*.
func FilterUserDefinedMetadataInPlace(nsconfig *core.NamespaceDefinition) {
	nsconfig.Metadata = nil
	for _, relation := range nsconfig.Relation {
		if relation.Metadata != nil {
			var filteredMessages []*anypb.Any
			for _, msg := range relation.Metadata.MetadataMessage {
				if _, ok := userDefinedMetadataTypeUrls[msg.TypeUrl]; !ok {
					filteredMessages = append(filteredMessages, msg)
				}
			}
			relation.Metadata.MetadataMessage = filteredMessages
		}
	}
}

// GetComments returns the comment metadata found within the given metadata message.
func GetComments(metadata *core.Metadata) []string {
	if metadata == nil {
		return []string{}
	}

	comments := []string{}
	for _, msg := range metadata.MetadataMessage {
		var dc iv1.DocComment
		if err := msg.UnmarshalTo(&dc); err == nil {
			comments = append(comments, dc.Comment)
		}
	}

	return comments
}

// AddComment adds a comment to the given metadata message.
func AddComment(metadata *core.Metadata, comment string) (*core.Metadata, error) {
	if metadata == nil {
		metadata = &core.Metadata{}
	}

	var dc iv1.DocComment
	dc.Comment = comment

	encoded, err := anypb.New(&dc)
	if err != nil {
		return metadata, err
	}

	metadata.MetadataMessage = append(metadata.MetadataMessage, encoded)
	return metadata, nil
}

// GetRelationKind returns the kind of the relation.
func GetRelationKind(relation *core.Relation) iv1.RelationMetadata_RelationKind {
	metadata := relation.Metadata
	if metadata == nil {
		return iv1.RelationMetadata_UNKNOWN_KIND
	}

	for _, msg := range metadata.MetadataMessage {
		var rm iv1.RelationMetadata
		if err := msg.UnmarshalTo(&rm); err == nil {
			return rm.Kind
		}
	}

	return iv1.RelationMetadata_UNKNOWN_KIND
}

// SetRelationKind sets the kind of relation.
func SetRelationKind(relation *core.Relation, kind iv1.RelationMetadata_RelationKind) error {
	metadata := relation.Metadata
	if metadata == nil {
		metadata = &core.Metadata{}
		relation.Metadata = metadata
	}

	var rm iv1.RelationMetadata
	rm.Kind = kind

	encoded, err := anypb.New(&rm)
	if err != nil {
		return err
	}

	metadata.MetadataMessage = append(metadata.MetadataMessage, encoded)
	return nil
}

// GetTypeAnnotations returns the type annotations for a permission relation.
func GetTypeAnnotations(relation *core.Relation) []string {
	if relation.Metadata == nil {
		return nil
	}

	for _, metadataAny := range relation.Metadata.MetadataMessage {
		var relationMetadata iv1.RelationMetadata
		if err := metadataAny.UnmarshalTo(&relationMetadata); err != nil {
			// Skip if this metadata message is not RelationMetadata
			continue
		}

		if relationMetadata.Kind == iv1.RelationMetadata_PERMISSION {
			if relationMetadata.TypeAnnotations != nil {
				return relationMetadata.TypeAnnotations.Types
			}
			return nil
		}
	}

	return nil
}

// SetTypeAnnotations sets the type annotations for a permission relation.
// If typeAnnotations is nil, removes any existing type annotations.
func SetTypeAnnotations(relation *core.Relation, typeAnnotations []string) error {
	if relation.Metadata == nil {
		if typeAnnotations == nil {
			return nil // Nothing to remove
		}
		relation.Metadata = &core.Metadata{}
	}

	// Find existing PERMISSION RelationMetadata and update it, or create new one
	for i, metadataAny := range relation.Metadata.MetadataMessage {
		var relationMetadata iv1.RelationMetadata
		if err := metadataAny.UnmarshalTo(&relationMetadata); err != nil {
			continue // Skip if this is not RelationMetadata
		}

		// Only update if this is the PERMISSION metadata
		if relationMetadata.Kind == iv1.RelationMetadata_PERMISSION {
			if typeAnnotations == nil {
				// Remove type annotations by setting to nil
				relationMetadata.TypeAnnotations = nil
			} else {
				// Update existing RelationMetadata with type annotations
				relationMetadata.TypeAnnotations = &iv1.TypeAnnotations{
					Types: typeAnnotations,
				}
			}

			// Re-encode and replace the existing message
			updatedAny, err := anypb.New(&relationMetadata)
			if err != nil {
				return err
			}

			relation.Metadata.MetadataMessage[i] = updatedAny
			return nil
		}
	}

	// If no existing RelationMetadata found and typeAnnotations is nil, nothing to do
	if typeAnnotations == nil {
		return nil
	}

	// If no existing RelationMetadata found, create new one
	relationMetadata := &iv1.RelationMetadata{
		Kind: iv1.RelationMetadata_PERMISSION,
		TypeAnnotations: &iv1.TypeAnnotations{
			Types: typeAnnotations,
		},
	}

	metadataAny, err := anypb.New(relationMetadata)
	if err != nil {
		return err
	}

	relation.Metadata.MetadataMessage = append(relation.Metadata.MetadataMessage, metadataAny)
	return nil
}
