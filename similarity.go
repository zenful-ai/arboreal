package arboreal

import "math"

func CosineSimilarity(vec1, vec2 []float32) float64 {
	var dotProduct, magnitudeVec1, magnitudeVec2 float64

	for i := range vec1 {
		dotProduct += float64(vec1[i] * vec2[i])
		magnitudeVec1 += math.Pow(float64(vec1[i]), 2)
		magnitudeVec2 += math.Pow(float64(vec2[i]), 2)
	}

	return dotProduct / (math.Sqrt(magnitudeVec1) * math.Sqrt(magnitudeVec2))
}
