package utils

import (
	"encoding/json"
	"errors"
)

// ErrPatchNotAnObject is what MergeJsonPatch answers for a patch that is a
// scalar or an array. RFC 7386 would replace the whole document with it; every
// jsonb column of this service is documented as an object, so it is refused
// instead. Callers turn it into a 400.
var ErrPatchNotAnObject = errors.New("the patch has to be a JSON object")

// MergeJsonPatch applies patch on top of stored with RFC 7386 semantics: objects
// are merged key by key, a null value deletes the key, anything else replaces
// what was there. See docs/steering/models-layer.md.
func MergeJsonPatch(stored json.RawMessage, patch json.RawMessage) (json.RawMessage, error) {
	document := map[string]any{}

	if len(stored) > 0 {
		if err := json.Unmarshal(stored, &document); err != nil {
			return nil, err
		}

		// A stored `null` unmarshals into a nil map rather than leaving the
		// initialised one alone, and writing into that panics.
		if document == nil {
			document = map[string]any{}
		}
	}

	changes, err := patchObject(patch)

	if err != nil {
		return nil, err
	}

	return json.Marshal(mergeInto(document, changes))
}

func patchObject(patch json.RawMessage) (map[string]any, error) {
	if len(patch) == 0 {
		return map[string]any{}, nil
	}

	changes := map[string]any{}

	if err := json.Unmarshal(patch, &changes); err != nil {
		var value any

		if json.Unmarshal(patch, &value) == nil {
			return nil, ErrPatchNotAnObject
		}

		return nil, err
	}

	return changes, nil
}

func mergeInto(document map[string]any, changes map[string]any) map[string]any {
	for key, value := range changes {
		if value == nil {
			delete(document, key)
			continue
		}

		nested, ok := value.(map[string]any)

		if !ok {
			document[key] = value
			continue
		}

		current, ok := document[key].(map[string]any)

		if !ok {
			current = map[string]any{}
		}

		document[key] = mergeInto(current, nested)
	}

	return document
}
