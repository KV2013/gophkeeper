package middleware

import "net/http"

// Chain объединяет middleware в цепочку. Первый аргумент — самый внешний слой.
func Chain(mws ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		for i := len(mws) - 1; i >= 0; i-- {
			h = mws[i](h)
		}
		return h
	}
}
