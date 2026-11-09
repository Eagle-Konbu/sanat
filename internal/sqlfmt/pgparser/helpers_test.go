package pgparser_test

import (
	"reflect"
	"testing"
)

// mustBe returns node as a T, failing the test if it is not one.
func mustBe[T any](t *testing.T, node any) T {
	t.Helper()

	got, ok := node.(T)
	if !ok {
		t.Fatalf("got %T, want %T", node, got)
	}

	return got
}

// rawFields names the pgast fields holding source spellings, which
// structure-only tests ignore; raw_test.go asserts them.
var rawFields = map[string]bool{"Raw": true, "ValueRaw": true, "FieldRaw": true}

// clearRaw blanks every source-spelling field reachable from node, a pointer
// into a parsed tree, so it can be compared against a hand-built tree.
func clearRaw(node any) {
	clearRawValue(reflect.ValueOf(node))
}

func clearRawValue(v reflect.Value) {
	switch v.Kind() {
	case reflect.Pointer:
		if !v.IsNil() {
			clearRawValue(v.Elem())
		}
	case reflect.Interface:
		clearRawInInterface(v)
	case reflect.Slice:
		for i := range v.Len() {
			clearRawValue(v.Index(i))
		}
	case reflect.Struct:
		clearRawInStruct(v)
	default:
	}
}

func clearRawInStruct(v reflect.Value) {
	for i := range v.NumField() {
		field := v.Field(i)
		if rawFields[v.Type().Field(i).Name] && field.Kind() == reflect.String {
			field.SetString("")
		} else {
			clearRawValue(field)
		}
	}
}

// clearRawInInterface clears an interface's dynamic value, which is not
// settable in place when it is a struct rather than a pointer.
func clearRawInInterface(v reflect.Value) {
	if v.IsNil() {
		return
	}

	elem := v.Elem()
	if elem.Kind() == reflect.Pointer {
		clearRawValue(elem)

		return
	}

	cleared := reflect.New(elem.Type()).Elem()
	cleared.Set(elem)
	clearRawValue(cleared)
	v.Set(cleared)
}
