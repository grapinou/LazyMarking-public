package tools

import "math/rand/v2"

func ShuffleSlice[T any](slice []T) {

	rand.Shuffle(len(slice), func(i, j int) { slice[i], slice[j] = slice[j], slice[i] })

}
