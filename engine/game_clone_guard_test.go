package engine

import (
	"reflect"
	"slices"
	"testing"
)

func assertNoSharedReferences(t *testing.T, original, clone any) {
	t.Helper()
	ov, cv := reflect.ValueOf(original).Elem(), reflect.ValueOf(clone).Elem()
	if ov.Type() != cv.Type() {
		t.Fatalf("types differ: %v vs %v", ov.Type(), cv.Type())
	}
	checkNoSharedReferences(t, ov.Type().Name(), ov, cv)
}

func checkNoSharedReferences(t *testing.T, path string, original, clone reflect.Value) {
	t.Helper()
	for i := range original.NumField() {
		name := path + "." + original.Type().Field(i).Name
		of, cf := original.Field(i), clone.Field(i)
		kind := of.Kind()
		switch {
		case kind == reflect.Struct:
			checkNoSharedReferences(t, name, of, cf)
		case slices.Contains(unclonableKinds, kind):
			t.Errorf("%s is a %s field: extend this guard before adding one", name, kind)
		case slices.Contains(sharedKinds, kind):
			if isEmpty(of) {
				t.Errorf("%s is empty in the fixture, so the clone cannot be checked: fill it", name)
			} else if of.Pointer() == cf.Pointer() {
				t.Errorf("%s shares memory with its clone", name)
			}
		}
	}
}

var (
	sharedKinds     = []reflect.Kind{reflect.Map, reflect.Slice, reflect.Pointer}
	unclonableKinds = []reflect.Kind{reflect.Chan, reflect.Func, reflect.Interface, reflect.UnsafePointer}
)

func isEmpty(v reflect.Value) bool {
	if v.Kind() == reflect.Pointer {
		return v.IsNil()
	}
	return v.Len() == 0
}

func TestGameState_DeepCopy_NoSharedReferences(t *testing.T) {
	gs := newCloneFixture(t).WorkingState
	assertNoSharedReferences(t, gs, gs.DeepCopy())
}

func TestMatch_Clone_NoSharedReferences(t *testing.T) {
	m := newCloneFixture(t)
	assertNoSharedReferences(t, m, m.Clone())
}

func TestEntities_HoldNoReferenceFields(t *testing.T) {
	tests := []struct {
		name string
		typ  reflect.Type
	}{
		{"Unit", reflect.TypeFor[Unit]()},
		{"Bomb", reflect.TypeFor[Bomb]()},
		{"SoftBlock", reflect.TypeFor[SoftBlock]()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertHoldsValuesOnly(t, tt.name, tt.typ)
		})
	}
}

func assertHoldsValuesOnly(t *testing.T, path string, typ reflect.Type) {
	t.Helper()
	kind := typ.Kind()
	switch {
	case kind == reflect.Struct:
		for f := range typ.Fields() {
			assertHoldsValuesOnly(t, path+"."+f.Name, f.Type)
		}
	case kind == reflect.Array:
		assertHoldsValuesOnly(t, path+"[]", typ.Elem())
	case slices.Contains(sharedKinds, kind) || slices.Contains(unclonableKinds, kind):
		t.Errorf("%s is a %s: cloneEntities copies by value and would share it", path, kind)
	}
}
