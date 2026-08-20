package handler

import (
	"crypto/tls"
	"net/http/httptest"
	"testing"
)

func TestPageParams(t *testing.T) {
	tests := map[string]struct {
		url      string
		wantPage int
		wantSize int
	}{
		"дефолты":            {url: "/api/v1/objects", wantPage: 1, wantSize: 10},
		"номер и размер":     {url: "/api/v1/objects?page[number]=2&page[size]=5", wantPage: 2, wantSize: 5},
		"некорректный номер": {url: "/api/v1/objects?page[number]=abc", wantPage: 1, wantSize: 10},
		"номер меньше 1":     {url: "/api/v1/objects?page[number]=0", wantPage: 1, wantSize: 10},
		"размер больше 100":  {url: "/api/v1/objects?page[size]=500", wantPage: 1, wantSize: 100},
		"нулевой размер":     {url: "/api/v1/objects?page[size]=0", wantPage: 1, wantSize: 10},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			r := httptest.NewRequest("GET", tc.url, nil)
			page, size := pageParams(r)
			if page != tc.wantPage || size != tc.wantSize {
				t.Fatalf("pageParams: got (%d,%d), want (%d,%d)", page, size, tc.wantPage, tc.wantSize)
			}
		})
	}
}

func TestPagesCount(t *testing.T) {
	tests := map[string]struct {
		total, size int
		want        int
	}{
		"ровно одна страница": {total: 10, size: 10, want: 1},
		"неполная страница":   {total: 15, size: 10, want: 2},
		"пусто":               {total: 0, size: 10, want: 0},
		"нулевой размер":      {total: 10, size: 0, want: 0},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if got := pagesCount(tc.total, tc.size); got != tc.want {
				t.Fatalf("pagesCount: got %d, want %d", got, tc.want)
			}
		})
	}
}

func TestBuildLinks(t *testing.T) {
	const base = "https://localhost:8080/api/v1/objects"

	strPtr := func(s string) *string { return &s }

	tests := map[string]struct {
		page, size, pages int
		wantFirst         string
		wantLast          string
		wantPrev          *string
		wantNext          *string
	}{
		"первая из двух": {
			page: 1, size: 10, pages: 2,
			wantFirst: base + "?page[number]=1&page[size]=10",
			wantLast:  base + "?page[number]=2&page[size]=10",
			wantPrev:  nil,
			wantNext:  strPtr(base + "?page[number]=2&page[size]=10"),
		},
		"средняя": {
			page: 2, size: 10, pages: 3,
			wantFirst: base + "?page[number]=1&page[size]=10",
			wantLast:  base + "?page[number]=3&page[size]=10",
			wantPrev:  strPtr(base + "?page[number]=1&page[size]=10"),
			wantNext:  strPtr(base + "?page[number]=3&page[size]=10"),
		},
		"одна страница": {
			page: 1, size: 10, pages: 1,
			wantFirst: base + "?page[number]=1&page[size]=10",
			wantLast:  base + "?page[number]=1&page[size]=10",
			wantPrev:  nil,
			wantNext:  nil,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			r := httptest.NewRequest("GET", base, nil)
			r.TLS = &tls.ConnectionState{}

			links := buildLinks(r, tc.page, tc.size, tc.pages)
			if links.First != tc.wantFirst {
				t.Fatalf("First: got %q, want %q", links.First, tc.wantFirst)
			}
			if links.Last != tc.wantLast {
				t.Fatalf("Last: got %q, want %q", links.Last, tc.wantLast)
			}
			checkPtr := func(got, want *string, field string) {
				if (got == nil) != (want == nil) {
					t.Fatalf("%s: nil mismatch: got %v, want %v", field, got, want)
				}
				if got != nil && *got != *want {
					t.Fatalf("%s: got %q, want %q", field, *got, *want)
				}
			}
			checkPtr(links.Prev, tc.wantPrev, "Prev")
			checkPtr(links.Next, tc.wantNext, "Next")
		})
	}
}
