package postgrest

import (
	"fmt"
	"testing"
)

func TestDefaultHeaders(t *testing.T) {
	headers := DefaultHeaders()

	if _, ok := headers["X-Client-Info"]; !ok {
		t.Fatalf("X-Client-Info header not found")
	}

	expectedPrefix := fmt.Sprintf("postgrest-go/%s", version)
	if headers["X-Client-Info"] != expectedPrefix {
		t.Fatalf("unexpected X-Client-Info value: got %s, expected %s",
			headers["X-Client-Info"], expectedPrefix)
	}
}
