package geo

import "testing"

func TestNopLocator_AlwaysMisses(t *testing.T) {
	var l NopLocator
	loc, ok := l.Lookup("81.2.69.142")
	if ok {
		t.Fatalf("expected NopLocator to never resolve, got %+v", loc)
	}
}
