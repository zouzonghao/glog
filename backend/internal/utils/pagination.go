package utils

import (
	"math"
)

type Page struct {
	Number int
	IsLink bool
}

// GeneratePagination generates a list of pages for a pagination component.
// It shows a limited number of pages around the current page, plus the first and last pages.
func GeneratePagination(currentPage, totalPages int) map[string]interface{} {
	if totalPages <= 1 {
		return nil
	}

	var pages []Page
	window := 2 // Number of pages to show on each side of the current page

	// Always add the first page
	pages = append(pages, Page{Number: 1, IsLink: true})

	// Add ellipsis if needed
	if currentPage > window+2 {
		pages = append(pages, Page{Number: 0, IsLink: false}) // Ellipsis
	}

	// Add pages around the current page
	start := int(math.Max(2, float64(currentPage-window)))
	end := int(math.Min(float64(totalPages-1), float64(currentPage+window)))

	for i := start; i <= end; i++ {
		pages = append(pages, Page{Number: i, IsLink: true})
	}

	// Add ellipsis if needed
	if currentPage < totalPages-(window+1) {
		pages = append(pages, Page{Number: 0, IsLink: false}) // Ellipsis
	}

	// Always add the last page
	if totalPages > 1 {
		pages = append(pages, Page{Number: totalPages, IsLink: true})
	}

	// Remove duplicates that might occur if window is large
	finalPages := []Page{}
	seen := make(map[int]bool)
	for _, p := range pages {
		if p.Number == currentPage {
			p.IsLink = false
		}
		if p.Number == 0 {
			finalPages = append(finalPages, p)
			continue
		}
		if !seen[p.Number] {
			finalPages = append(finalPages, p)
			seen[p.Number] = true
		}
	}

	return map[string]interface{}{
		"CurrentPage": currentPage,
		"TotalPages":  totalPages,
		"HasPrev":     currentPage > 1,
		"HasNext":     currentPage < totalPages,
		"PrevPage":    currentPage - 1,
		"NextPage":    currentPage + 1,
		"Pages":       finalPages,
	}
}
