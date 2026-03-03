package errors

import "net/http"

// ForbiddenError represents a 403 Forbidden response.
type ForbiddenError struct {
	Message string
}

func (e ForbiddenError) Error() string {
	return e.Message
}

func (e ForbiddenError) StatusCode() int {
	return http.StatusForbidden
}

// NotFoundError represents a 404 Not Found response.
type NotFoundError struct {
	Message string
}

func (e NotFoundError) Error() string {
	return e.Message
}

func (e NotFoundError) StatusCode() int {
	return http.StatusNotFound
}

// BadRequestError represents a 400 Bad Request response.
type BadRequestError struct {
	Message string
}

func (e BadRequestError) Error() string {
	return e.Message
}

func (e BadRequestError) StatusCode() int {
	return http.StatusBadRequest
}

// UnauthorizedError represents a 401 Unauthorized response.
type UnauthorizedError struct {
	Message string
}

func (e UnauthorizedError) Error() string {
	return e.Message
}

func (e UnauthorizedError) StatusCode() int {
	return http.StatusUnauthorized
}
