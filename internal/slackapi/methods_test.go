package slackapi

import "testing"

func TestMethods_NoDuplicateNames(t *testing.T) {
	seen := map[string]bool{}
	for _, meth := range Methods {
		if seen[meth.Name] {
			t.Errorf("duplicate method name %q", meth.Name)
		}
		seen[meth.Name] = true
	}
}

func TestMethods_AllHaveAtLeastOneDot(t *testing.T) {
	for _, meth := range Methods {
		hasDot := false
		for _, r := range meth.Name {
			if r == '.' {
				hasDot = true
				break
			}
		}
		if !hasDot {
			t.Errorf("method %q has no '.' separator; command-tree generation assumes namespace.action", meth.Name)
		}
	}
}

func TestLookup(t *testing.T) {
	meth, ok := Lookup("chat.postMessage")
	if !ok {
		t.Fatal("expected chat.postMessage to be found")
	}
	if meth.Desc == "" {
		t.Error("expected non-empty description")
	}

	if _, ok := Lookup("not.a.real.method"); ok {
		t.Error("expected unknown method to not be found")
	}
}

func TestAll_Sorted(t *testing.T) {
	all := All()
	if len(all) != len(Methods) {
		t.Fatalf("expected %d methods, got %d", len(Methods), len(all))
	}
	for i := 1; i < len(all); i++ {
		if all[i-1].Name > all[i].Name {
			t.Fatalf("All() not sorted: %q before %q", all[i-1].Name, all[i].Name)
		}
	}
}
