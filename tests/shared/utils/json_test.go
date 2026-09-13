package utils_test

import (
	"encoding/json"
	"testing"

	"auth_service/shared/utils"

	"github.com/stretchr/testify/assert"
)

func merge(t *testing.T, stored string, patch string) map[string]any {
	t.Helper()

	merged, err := utils.MergeJsonPatch(json.RawMessage(stored), json.RawMessage(patch))

	assert.NoError(t, err)

	result := map[string]any{}

	assert.NoError(t, json.Unmarshal(merged, &result))

	return result
}

// The whole point of the merge: a PUT that names one key must not erase the
// keys it does not name.
func TestMergeKeepsWhatThePatchDoesNotName(t *testing.T) {
	assert.Equal(
		t,
		map[string]any{"a": float64(1), "b": float64(2)},
		merge(t, `{"a": 1}`, `{"b": 2}`),
	)
}

func TestPatchWinsOverStored(t *testing.T) {
	assert.Equal(t, map[string]any{"a": "new"}, merge(t, `{"a": "old"}`, `{"a": "new"}`))
}

// The only way to delete a key, and the reason the merge is a merge patch
// rather than a plain deep merge.
func TestNullDeletesTheKey(t *testing.T) {
	assert.Equal(t, map[string]any{"b": float64(2)}, merge(t, `{"a": 1, "b": 2}`, `{"a": null}`))
}

func TestDeletingAnAbsentKeyIsANoop(t *testing.T) {
	assert.Equal(t, map[string]any{"a": float64(1)}, merge(t, `{"a": 1}`, `{"b": null}`))
}

func TestNestedObjectsMergeInsteadOfReplacing(t *testing.T) {
	assert.Equal(
		t,
		map[string]any{"outer": map[string]any{"kept": float64(1), "added": float64(2)}},
		merge(t, `{"outer": {"kept": 1}}`, `{"outer": {"added": 2}}`),
	)
}

func TestNestedNullDeletesOnlyThatKey(t *testing.T) {
	assert.Equal(
		t,
		map[string]any{"outer": map[string]any{"kept": float64(1)}},
		merge(t, `{"outer": {"kept": 1, "gone": 2}}`, `{"outer": {"gone": null}}`),
	)
}

// An array is a value, not a document to merge into: two elements do not have a
// key to be matched by.
func TestArraysAreReplacedWholesale(t *testing.T) {
	assert.Equal(
		t,
		map[string]any{"list": []any{float64(3)}},
		merge(t, `{"list": [1, 2]}`, `{"list": [3]}`),
	)
}

// Every column is `not null default '{}'`, but a row written before a migration
// can still hand back nothing, and answering nil would write NULL into it.
func TestEmptyStoredIsTreatedAsAnEmptyObject(t *testing.T) {
	assert.Equal(t, map[string]any{"a": float64(1)}, merge(t, ``, `{"a": 1}`))
	assert.Equal(t, map[string]any{"a": float64(1)}, merge(t, `null`, `{"a": 1}`))
	assert.Equal(t, map[string]any{}, merge(t, ``, ``))
}

func TestEmptyPatchLeavesTheDocumentAlone(t *testing.T) {
	assert.Equal(t, map[string]any{"a": float64(1)}, merge(t, `{"a": 1}`, `{}`))
}

// RFC 7386 would replace the document with the scalar. Every jsonb column here
// is documented as an object, so it is refused and the service answers 400.
func TestANonObjectPatchIsRefused(t *testing.T) {
	for _, patch := range []string{`5`, `"text"`, `[1, 2]`, `true`} {
		_, err := utils.MergeJsonPatch(json.RawMessage(`{}`), json.RawMessage(patch))

		assert.ErrorIs(t, err, utils.ErrPatchNotAnObject, "`%s` should be refused", patch)
	}
}

func TestMalformedPatchIsAnError(t *testing.T) {
	_, err := utils.MergeJsonPatch(json.RawMessage(`{}`), json.RawMessage(`{"a":`))

	assert.Error(t, err)
	assert.NotErrorIs(t, err, utils.ErrPatchNotAnObject)
}

// encoding/json sorts map keys, so the column is stable across writes and a
// faster encoder cannot be swapped in without this failing.
func TestOutputIsDeterministic(t *testing.T) {
	merged, err := utils.MergeJsonPatch(json.RawMessage(`{"b": 1}`), json.RawMessage(`{"a": 2, "c": 3}`))

	assert.NoError(t, err)
	assert.Equal(t, `{"a":2,"b":1,"c":3}`, string(merged))
}
