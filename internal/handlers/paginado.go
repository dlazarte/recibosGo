package handlers

import (
	"net/http"
	"strconv"
)

type PageLink struct {
	Num    int
	URL    string
	Active bool
}

type Pagination struct {
	Current    int
	PerPage    int
	TotalItems int64
	TotalPages int
	Target     string
	Links      []PageLink
	HasPrev    bool
	PrevURL    string
	HasNext    bool
	NextURL    string
}

const perPage = 25

func paginaActual(r *http.Request) int {
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		return p
	}
	return 1
}

func paginar(r *http.Request, total int64, target string) Pagination {
	page := paginaActual(r)
	totalPages := int((total + perPage - 1) / perPage)
	if totalPages == 0 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}

	p := Pagination{
		Current:    page,
		PerPage:    perPage,
		TotalItems: total,
		TotalPages: totalPages,
		Target:     target,
	}

	pageURL := func(n int) string {
		q := r.URL.Query()
		q.Set("page", strconv.Itoa(n))
		return r.URL.Path + "?" + q.Encode()
	}

	if page > 1 {
		p.HasPrev = true
		p.PrevURL = pageURL(page - 1)
	}
	if page < totalPages {
		p.HasNext = true
		p.NextURL = pageURL(page + 1)
	}

	start := page - 3
	if start < 1 {
		start = 1
	}
	end := page + 3
	if end > totalPages {
		end = totalPages
	}
	for n := start; n <= end; n++ {
		p.Links = append(p.Links, PageLink{Num: n, URL: pageURL(n), Active: n == page})
	}

	return p
}
