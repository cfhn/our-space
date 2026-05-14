package frontend

import "testing"

func TestEmbeddedFrontend(t *testing.T) {
	f, err := frontend.Open("dist/index.html")
	if err != nil {
		t.Errorf("expected dist/index.html, got error=%v", err)
	}

	t.Cleanup(func() {
		f.Close()
	})
}
