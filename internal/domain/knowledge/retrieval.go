package knowledge

import (
	"hash/fnv"
	"math"
)

const localEmbeddingDimensions = 64

type retrievalScore struct {
	lexical int
	vector  float64
}

func scoreKnowledgeMatch(queryTokens []string, queryEmbedding []float64, text string) retrievalScore {
	candidateTokens := tokenize(text)
	return retrievalScore{
		lexical: overlapScore(queryTokens, candidateTokens),
		vector:  cosineSimilarity(queryEmbedding, embedTokens(candidateTokens)),
	}
}

func (score retrievalScore) matched() bool {
	return score.lexical > 0
}

func (score retrievalScore) weighted() float64 {
	return float64(score.lexical*10) + score.vector
}

func compareRetrievalScore(left, right retrievalScore) int {
	leftScore := left.weighted()
	rightScore := right.weighted()
	if math.Abs(leftScore-rightScore) > 0.000001 {
		if leftScore > rightScore {
			return 1
		}
		return -1
	}
	if math.Abs(left.vector-right.vector) > 0.000001 {
		if left.vector > right.vector {
			return 1
		}
		return -1
	}
	return 0
}

func embedText(text string) []float64 {
	return embedTokens(tokenize(text))
}

func embedTokens(tokens []string) []float64 {
	vector := make([]float64, localEmbeddingDimensions)
	for _, token := range tokens {
		index, sign := embeddingBucket(token)
		vector[index] += sign * tokenWeight(token)
	}
	normalizeVector(vector)
	return vector
}

func embeddingBucket(token string) (int, float64) {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(token))
	value := hash.Sum32()
	sign := 1.0
	if value&1 == 1 {
		sign = -1.0
	}
	return int((value >> 1) % localEmbeddingDimensions), sign
}

func tokenWeight(token string) float64 {
	if len([]rune(token)) >= 4 {
		return 1.2
	}
	return 1
}

func normalizeVector(vector []float64) {
	var sum float64
	for _, value := range vector {
		sum += value * value
	}
	if sum == 0 {
		return
	}
	scale := math.Sqrt(sum)
	for index := range vector {
		vector[index] = vector[index] / scale
	}
}

func cosineSimilarity(left, right []float64) float64 {
	if len(left) == 0 || len(left) != len(right) {
		return 0
	}
	var dot float64
	for index := range left {
		dot += left[index] * right[index]
	}
	return dot
}
