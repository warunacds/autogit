package provider_test

import (
	"testing"

	"github.com/warunacds/autogit/internal/provider"
)

func TestValidateAndTruncateDiff_Empty(t *testing.T) {
	_, err := provider.ValidateAndTruncateDiff("")
	if err == nil {
		t.Fatal("expected error for empty diff")
	}
}

func TestValidateAndTruncateDiff_Normal(t *testing.T) {
	diff := "some diff content"
	result, err := provider.ValidateAndTruncateDiff(diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != diff {
		t.Fatalf("expected unchanged diff, got %q", result)
	}
}

func TestValidateAndTruncateDiff_Oversized(t *testing.T) {
	big := make([]byte, 200*1024)
	for i := range big {
		big[i] = 'x'
	}
	result, err := provider.ValidateAndTruncateDiff(string(big))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != provider.MaxDiffBytes {
		t.Fatalf("expected truncated to %d bytes, got %d", provider.MaxDiffBytes, len(result))
	}
}
