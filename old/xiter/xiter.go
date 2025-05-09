package xiter

import "iter"

func SliceToKeyedMap[T any, K comparable](seq []T, keyFn func(T) K) iter.Seq2[K, T] {
	return func(yield func(K, T) bool) {
		for _, val := range seq {
			key := keyFn(val)

			if !yield(key, val) {
				return
			}
		}
	}
}

func MapMap[K comparable, V any, U any](seq iter.Seq2[K, V], transform func(V) U) iter.Seq2[K, U] {
	return func(yield func(K, U) bool) {
		for key, val := range seq {
			newVal := transform(val)

			if !yield(key, newVal) {
				return
			}
		}
	}
}
