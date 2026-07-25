package dto

// API envelope — mirrors packages/types/src/common.ts exactly.

type Response[T any] struct {
	Success bool   `json:"success"`
	Data    T      `json:"data,omitempty"`
	Error   *Error `json:"error,omitempty"`
}

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

type PaginationMeta struct {
	Limit      int        `json:"limit"`
	NextCursor *string    `json:"nextCursor,omitempty"`
}

func OK[T any](data T) Response[T] {
	return Response[T]{Success: true, Data: data}
}

func Fail(code, message, field string) Response[struct{}] {
	return Response[struct{}]{
		Success: false,
		Error:   &Error{Code: code, Message: message, Field: field},
	}
}
