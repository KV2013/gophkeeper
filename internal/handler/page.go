package handler

import (
	"fmt"
	"math"
	"net/http"
	"strconv"

	"github.com/victor/gophkeeper/internal/model"
)

// Параметры пагинации по умолчанию.
const (
	defaultPageSize = 10
	maxPageSize     = 100
)

// objectsPageResponse — JSON-API-ответ списка объектов с пагинацией.
type objectsPageResponse struct {
	Data     []*model.Object `json:"data"`
	Metadata pageMetadata    `json:"metadata"`
	Links    pageLinks       `json:"links"`
}

// pageMetadata — метаданные пагинации.
type pageMetadata struct {
	Total      int `json:"total"`
	Pages      int `json:"pages"`
	PageSize   int `json:"page_size"`
	PageNumber int `json:"page_number"`
}

// pageLinks — ссылки пагинации.
type pageLinks struct {
	First string  `json:"first"`
	Last  string  `json:"last"`
	Prev  *string `json:"prev"`
	Next  *string `json:"next"`
}

// pageParams извлекает page[number] и page[size] из запроса с валидацией.
func pageParams(r *http.Request) (page, size int) {
	page, size = 1, defaultPageSize
	if v := r.URL.Query().Get("page[number]"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			page = n
		}
	}
	if v := r.URL.Query().Get("page[size]"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			size = n
		}
	}
	if size > maxPageSize {
		size = maxPageSize
	}
	return page, size
}

// buildLinks строит ссылки пагинации на основе текущего запроса.
func buildLinks(r *http.Request, page, size, pages int) pageLinks {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	base := fmt.Sprintf("%s://%s/api/v1/objects", scheme, r.Host)
	mk := func(p int) string {
		return fmt.Sprintf("%s?page[number]=%d&page[size]=%d", base, p, size)
	}

	links := pageLinks{First: mk(1), Last: mk(1)}
	if pages > 0 {
		links.Last = mk(pages)
	}
	if page > 1 {
		v := mk(page - 1)
		links.Prev = &v
	}
	if page < pages {
		v := mk(page + 1)
		links.Next = &v
	}
	return links
}

// pagesCount вычисляет количество страниц.
func pagesCount(total, size int) int {
	if size <= 0 {
		return 0
	}
	return int(math.Ceil(float64(total) / float64(size)))
}
