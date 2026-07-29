package dto

type PaginationDTO struct {
	Total       int  `json:"total"`
	PerPage     int  `json:"per_page"`
	CurrentPage int  `json:"current_page"`
	NextPage    *int `json:"next_page,omitempty"`
	PrevPage    *int `json:"prev_page,omitempty"`
	LastPage    int  `json:"last_page"`
}

type PaginatedDTO[T any] struct {
	Data       []T           `json:"data"`
	Pagination PaginationDTO `json:"pagination"`
}

func ToPaginationDTO(total int, perPage int, currentPage int) *PaginationDTO {
	if perPage <= 0 {
		perPage = 10
	}

	if currentPage <= 0 {
		currentPage = 1
	}

	lastPage := (total + perPage - 1) / perPage

	var nextPage *int
	if currentPage < lastPage {
		n := currentPage + 1
		nextPage = &n
	}

	var prevPage *int
	if currentPage > 1 {
		p := currentPage - 1
		prevPage = &p
	}

	return &PaginationDTO{
		Total:       total,
		PerPage:     perPage,
		CurrentPage: currentPage,
		NextPage:    nextPage,
		PrevPage:    prevPage,
		LastPage:    lastPage,
	}
}

func ToPaginatedDTO[T any](data []T, total int, perPage int, currentPage int) *PaginatedDTO[T] {
	return &PaginatedDTO[T]{
		Data:       data,
		Pagination: *ToPaginationDTO(total, perPage, currentPage),
	}
}
