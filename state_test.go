package arboreal

import (
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
)

func TestMonotonicIdGenerator(t *testing.T) {
	t.Run("generates monotonic IDs", func(t *testing.T) {
		generator := MonotonicIdGenerator("test-")

		id1 := generator()
		id2 := generator()
		id3 := generator()

		expected := []string{"test-1", "test-2", "test-3"}
		actual := []string{id1, id2, id3}

		for i, expected := range expected {
			if actual[i] != expected {
				t.Errorf("Expected ID %q, got %q", expected, actual[i])
			}
		}
	})

	t.Run("different generators are independent", func(t *testing.T) {
		gen1 := MonotonicIdGenerator("gen1-")
		gen2 := MonotonicIdGenerator("gen2-")

		id1a := gen1()
		id2a := gen2()
		id1b := gen1()
		id2b := gen2()

		if id1a != "gen1-1" {
			t.Errorf("Expected gen1-1, got %s", id1a)
		}
		if id2a != "gen2-1" {
			t.Errorf("Expected gen2-1, got %s", id2a)
		}
		if id1b != "gen1-2" {
			t.Errorf("Expected gen1-2, got %s", id1b)
		}
		if id2b != "gen2-2" {
			t.Errorf("Expected gen2-2, got %s", id2b)
		}
	})

	t.Run("concurrent access is safe", func(t *testing.T) {
		generator := MonotonicIdGenerator("concurrent-")
		var wg sync.WaitGroup
		const numGoroutines = 100
		const idsPerGoroutine = 10

		results := make([]string, numGoroutines*idsPerGoroutine)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineIndex int) {
				defer wg.Done()
				for j := 0; j < idsPerGoroutine; j++ {
					id := generator()
					results[goroutineIndex*idsPerGoroutine+j] = id
				}
			}(i)
		}

		wg.Wait()

		// Check that all IDs are unique
		idSet := make(map[string]bool)
		for _, id := range results {
			if idSet[id] {
				t.Errorf("Duplicate ID found: %s", id)
			}
			idSet[id] = true

			// Check format
			if !strings.HasPrefix(id, "concurrent-") {
				t.Errorf("ID %s doesn't have expected prefix", id)
			}
		}

		if len(idSet) != numGoroutines*idsPerGoroutine {
			t.Errorf("Expected %d unique IDs, got %d", numGoroutines*idsPerGoroutine, len(idSet))
		}
	})

	t.Run("empty prefix works", func(t *testing.T) {
		generator := MonotonicIdGenerator("")

		id1 := generator()
		id2 := generator()

		if id1 != "1" {
			t.Errorf("Expected '1', got %s", id1)
		}
		if id2 != "2" {
			t.Errorf("Expected '2', got %s", id2)
		}
	})

	t.Run("special character prefix", func(t *testing.T) {
		generator := MonotonicIdGenerator("@#$-")

		id := generator()
		expected := "@#$-1"

		if id != expected {
			t.Errorf("Expected %q, got %q", expected, id)
		}
	})
}

func TestGenerateStringIdentifier(t *testing.T) {
	t.Run("generates string with correct prefix and length", func(t *testing.T) {
		prefix := "test-"
		length := 16

		id, err := GenerateStringIdentifier(prefix, length)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(id) != length {
			t.Errorf("Expected length %d, got %d", length, len(id))
		}

		if !strings.HasPrefix(id, prefix) {
			t.Errorf("Expected prefix %q, but ID %q doesn't have it", prefix, id)
		}
	})

	t.Run("generates unique IDs", func(t *testing.T) {
		prefix := "unique-"
		length := 20
		numIDs := 100

		ids := make([]string, numIDs)
		for i := 0; i < numIDs; i++ {
			id, err := GenerateStringIdentifier(prefix, length)
			if err != nil {
				t.Fatalf("Error generating ID %d: %v", i, err)
			}
			ids[i] = id
		}

		// Check uniqueness
		idSet := make(map[string]bool)
		for _, id := range ids {
			if idSet[id] {
				t.Errorf("Duplicate ID found: %s", id)
			}
			idSet[id] = true
		}
	})

	t.Run("uses valid base32 characters", func(t *testing.T) {
		prefix := "b32-"
		length := 15

		id, err := GenerateStringIdentifier(prefix, length)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Remove prefix and check remaining characters
		withoutPrefix := id[len(prefix):]
		validPattern := regexp.MustCompile(`^[0-9a-v]+$`) // base32 uses 0-9 and a-v

		if !validPattern.MatchString(withoutPrefix) {
			t.Errorf("ID contains invalid base32 characters: %q", withoutPrefix)
		}
	})

	t.Run("deterministic mode with ZEN_SEED_RNG", func(t *testing.T) {
		// Set the environment variable
		originalEnv := os.Getenv("ZEN_SEED_RNG")
		defer func() {
			if originalEnv == "" {
				os.Unsetenv("ZEN_SEED_RNG")
			} else {
				os.Setenv("ZEN_SEED_RNG", originalEnv)
			}
		}()

		os.Setenv("ZEN_SEED_RNG", "12345")

		// Reset the once to allow re-seeding for test
		seedRNGOnce = sync.Once{}

		prefix := "det-"
		length := 12

		// Generate the same ID multiple times
		id1, err1 := GenerateStringIdentifier(prefix, length)
		if err1 != nil {
			t.Fatalf("Error generating first ID: %v", err1)
		}

		// Reset the once again
		seedRNGOnce = sync.Once{}

		id2, err2 := GenerateStringIdentifier(prefix, length)
		if err2 != nil {
			t.Fatalf("Error generating second ID: %v", err2)
		}

		if id1 != id2 {
			t.Errorf("Expected deterministic IDs to be equal, got %q and %q", id1, id2)
		}
	})

	t.Run("empty prefix", func(t *testing.T) {
		length := 10

		id, err := GenerateStringIdentifier("", length)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(id) != length {
			t.Errorf("Expected length %d, got %d", length, len(id))
		}
	})

	t.Run("minimum length", func(t *testing.T) {
		prefix := "x"
		length := 1

		id, err := GenerateStringIdentifier(prefix, length)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(id) != length {
			t.Errorf("Expected length %d, got %d", length, len(id))
		}

		if id != prefix {
			t.Errorf("Expected %q, got %q", prefix, id)
		}
	})

	t.Run("longer than prefix", func(t *testing.T) {
		prefix := "very-long-prefix-"
		length := 32

		id, err := GenerateStringIdentifier(prefix, length)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(id) != length {
			t.Errorf("Expected length %d, got %d", length, len(id))
		}

		if !strings.HasPrefix(id, prefix) {
			t.Errorf("Expected prefix %q, but ID %q doesn't have it", prefix, id)
		}

		// Check that there's some random content after the prefix
		withoutPrefix := id[len(prefix):]
		if len(withoutPrefix) == 0 {
			t.Error("Expected some content after prefix")
		}
	})
}

func TestGenerateStringIdentifierEdgeCases(t *testing.T) {
	t.Run("invalid ZEN_SEED_RNG value", func(t *testing.T) {
		originalEnv := os.Getenv("ZEN_SEED_RNG")
		defer func() {
			if originalEnv == "" {
				os.Unsetenv("ZEN_SEED_RNG")
			} else {
				os.Setenv("ZEN_SEED_RNG", originalEnv)
			}
		}()

		os.Setenv("ZEN_SEED_RNG", "not-a-number")

		// Reset the once to allow re-processing
		seedRNGOnce = sync.Once{}

		// Should still work (falls back to secure random)
		id, err := GenerateStringIdentifier("test-", 10)

		if err != nil {
			t.Fatalf("Should handle invalid ZEN_SEED_RNG gracefully: %v", err)
		}

		if len(id) != 10 {
			t.Errorf("Expected length 10, got %d", len(id))
		}
	})
}

// Benchmark tests
func BenchmarkMonotonicIdGenerator(b *testing.B) {
	generator := MonotonicIdGenerator("bench-")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		generator()
	}
}

func BenchmarkGenerateStringIdentifier(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GenerateStringIdentifier("bench-", 16)
	}
}

func BenchmarkGenerateStringIdentifierDeterministic(b *testing.B) {
	os.Setenv("ZEN_SEED_RNG", "12345")
	defer os.Unsetenv("ZEN_SEED_RNG")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		seedRNGOnce = sync.Once{} // Reset for each iteration in benchmark
		GenerateStringIdentifier("bench-", 16)
	}
}

func BenchmarkMonotonicIdGeneratorConcurrent(b *testing.B) {
	generator := MonotonicIdGenerator("concurrent-")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			generator()
		}
	})
}