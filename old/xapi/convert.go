package xapi

import (
	"github.com/samber/lo"
)

type Convertable[T any] interface {
	Convert() T
}

func ConvertEach[T any, U Convertable[T]](items []U) []T {
	return lo.Map(items, func(item U, index int) T {
		return item.Convert()
	})
}

func ConvertEachMap[T any, U Convertable[T], K comparable](items map[K]U) map[K]T {
	return lo.MapEntries(items, func(k K, v U) (K, T) {
		return k, v.Convert()
	})
}
