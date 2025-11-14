package domain

import (
	"errors"
	"fmt"
	"net/http"
)

// ErrorCode - код ошибки для API
type ErrorCode string

const (
	ErrorCodeTeamExists        ErrorCode = "TEAM_EXISTS"
	ErrorCodePullRequestExists ErrorCode = "PR_EXISTS"
	ErrorCodePullRequestMerged ErrorCode = "PR_MERGED"
	ErrorCodeReviewerMissing   ErrorCode = "NOT_ASSIGNED"
	ErrorCodeNoCandidate       ErrorCode = "NO_CANDIDATE"
	ErrorCodeNotFound          ErrorCode = "NOT_FOUND"
	ErrorCodeInternalError     ErrorCode = "INTERNAL_ERROR"
	ErrorCodeInvalidInput      ErrorCode = "INVALID_INPUT"
)

// Error - доменная ошибка с HTTP статусом и кодом
type Error struct {
	Status  int       // HTTP status code
	Code    ErrorCode // Код ошибки для API
	Message string    // Сообщение об ошибке
	Err     error     // Wrapped error для контекста
}

// Error реализует интерфейс error
func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap позволяет использовать errors.Is и errors.As
func (e *Error) Unwrap() error {
	return e.Err
}

// NewError создаёт новую доменную ошибку
func NewError(status int, code ErrorCode, message string, err error) *Error {
	return &Error{
		Status:  status,
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Предопределённые доменные ошибки
var (
	// ErrPRExists - pull request уже существует
	ErrPRExists = NewError(
		http.StatusConflict,
		ErrorCodePullRequestExists,
		"pull request already exists",
		nil,
	)

	// ErrResourceNotFound - ресурс не найден
	ErrResourceNotFound = NewError(
		http.StatusNotFound,
		ErrorCodeNotFound,
		"resource not found",
		nil,
	)

	// ErrTeamExists - команда уже существует
	ErrTeamExists = NewError(
		http.StatusBadRequest,
		ErrorCodeTeamExists,
		"team already exists",
		nil,
	)

	// ErrInternal - внутренняя ошибка сервера
	ErrInternal = NewError(
		http.StatusInternalServerError,
		ErrorCodeInternalError,
		"internal server error",
		nil,
	)

	// ErrReassignOnMerged - попытка переназначить ревьювера на merged PR
	ErrReassignOnMerged = NewError(
		http.StatusConflict,
		ErrorCodePullRequestMerged,
		"cannot reassign reviewer on merged pull request",
		nil,
	)

	// ErrReviewerMissing - указанный пользователь не является ревьювером этого PR
	ErrReviewerMissing = NewError(
		http.StatusConflict,
		ErrorCodeReviewerMissing,
		"user is not assigned as reviewer to this pull request",
		nil,
	)

	// ErrNoCandidate - нет активных кандидатов для замены ревьювера
	ErrNoCandidate = NewError(
		http.StatusConflict,
		ErrorCodeNoCandidate,
		"no active replacement candidate available in team",
		nil,
	)

	// ErrInvalidInput - невалидные входные данные
	ErrInvalidInput = NewError(
		http.StatusBadRequest,
		ErrorCodeInvalidInput,
		"invalid input data",
		nil,
	)
)

// IsDomainError проверяет, является ли ошибка доменной
func IsDomainError(err error) bool {
	var e *Error
	return errors.As(err, &e)
}

// WrapError оборачивает обычную ошибку в доменную с контекстом
func WrapError(err error, status int, code ErrorCode, message string) *Error {
	return NewError(status, code, message, err)
}
