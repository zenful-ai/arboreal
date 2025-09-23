package arboreal

import (
	"math"
	"testing"
)

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		vec1     []float32
		vec2     []float32
		expected float64
		delta    float64 // for floating point comparison
	}{
		{
			name:     "identical vectors",
			vec1:     []float32{1, 2, 3},
			vec2:     []float32{1, 2, 3},
			expected: 1.0,
			delta:    1e-10,
		},
		{
			name:     "orthogonal vectors",
			vec1:     []float32{1, 0},
			vec2:     []float32{0, 1},
			expected: 0.0,
			delta:    1e-10,
		},
		{
			name:     "opposite vectors",
			vec1:     []float32{1, 0},
			vec2:     []float32{-1, 0},
			expected: -1.0,
			delta:    1e-10,
		},
		{
			name:     "similar vectors",
			vec1:     []float32{1, 2, 3},
			vec2:     []float32{2, 4, 6},
			expected: 1.0,
			delta:    1e-10,
		},
		{
			name:     "unit vectors at 45 degrees",
			vec1:     []float32{1, 0},
			vec2:     []float32{float32(math.Sqrt(2) / 2), float32(math.Sqrt(2) / 2)},
			expected: math.Sqrt(2) / 2,
			delta:    1e-6,
		},
		{
			name:     "zero vectors",
			vec1:     []float32{0, 0, 0},
			vec2:     []float32{0, 0, 0},
			expected: math.NaN(),
			delta:    0,
		},
		{
			name:     "one zero vector",
			vec1:     []float32{1, 2, 3},
			vec2:     []float32{0, 0, 0},
			expected: math.NaN(),
			delta:    0,
		},
		{
			name:     "single element vectors",
			vec1:     []float32{5},
			vec2:     []float32{3},
			expected: 1.0,
			delta:    1e-10,
		},
		{
			name:     "negative values",
			vec1:     []float32{-1, -2, -3},
			vec2:     []float32{1, 2, 3},
			expected: -1.0,
			delta:    1e-10,
		},
		{
			name:     "mixed positive and negative",
			vec1:     []float32{1, -1, 0},
			vec2:     []float32{-1, 1, 0},
			expected: -1.0,
			delta:    1e-10,
		},
		{
			name:     "fractional values",
			vec1:     []float32{0.5, 0.5},
			vec2:     []float32{0.3, 0.4},
			expected: 0.9899494953470402,
			delta:    1e-6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CosineSimilarity(tt.vec1, tt.vec2)

			// Handle NaN cases
			if math.IsNaN(tt.expected) {
				if !math.IsNaN(result) {
					t.Errorf("CosineSimilarity() = %v, expected NaN", result)
				}
				return
			}

			if math.Abs(result-tt.expected) > tt.delta {
				t.Errorf("CosineSimilarity() = %v, expected %v (within %v)", result, tt.expected, tt.delta)
			}
		})
	}
}

func TestCosineSimilarityPanic(t *testing.T) {
	tests := []struct {
		name string
		vec1 []float32
		vec2 []float32
	}{
		{
			name: "mismatched vector lengths",
			vec1: []float32{1, 2, 3},
			vec2: []float32{1, 2},
		},
		{
			name: "one empty vector",
			vec1: []float32{1, 2, 3},
			vec2: []float32{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("CosineSimilarity() should have panicked for %s", tt.name)
				}
			}()
			CosineSimilarity(tt.vec1, tt.vec2)
		})
	}
}

func TestCosineSimilarityEmptyVectors(t *testing.T) {
	t.Run("empty vectors return NaN", func(t *testing.T) {
		result := CosineSimilarity([]float32{}, []float32{})
		if !math.IsNaN(result) {
			t.Errorf("Expected NaN for empty vectors, got %v", result)
		}
	})
}

// Benchmark tests for performance
func BenchmarkCosineSimilarity(b *testing.B) {
	vec1 := make([]float32, 1000)
	vec2 := make([]float32, 1000)

	// Fill with some values
	for i := range vec1 {
		vec1[i] = float32(i)
		vec2[i] = float32(i * 2)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CosineSimilarity(vec1, vec2)
	}
}

func BenchmarkCosineSimilaritySmall(b *testing.B) {
	vec1 := []float32{1, 2, 3, 4, 5}
	vec2 := []float32{2, 4, 6, 8, 10}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CosineSimilarity(vec1, vec2)
	}
}

