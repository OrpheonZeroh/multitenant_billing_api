package models

import "time"

// ErrorCode representa el código de error
type ErrorCode string

const (
	ErrorCodeInvalidRequest ErrorCode = "INVALID_REQUEST"
	ErrorCodeUnauthorized   ErrorCode = "UNAUTHORIZED"
	ErrorCodeForbidden      ErrorCode = "FORBIDDEN"
	ErrorCodeNotFound       ErrorCode = "NOT_FOUND"
	ErrorCodeConflict       ErrorCode = "CONFLICT"
	ErrorCodeRateLimited    ErrorCode = "RATE_LIMITED"
	ErrorCodeInternal       ErrorCode = "INTERNAL"
)

// ErrorDetail representa un detalle específico del error
type ErrorDetail struct {
	Field string `json:"field"`
	Issue string `json:"issue"`
}

// ErrorResponse representa la respuesta de error estandarizada
type ErrorResponse struct {
	Error ErrorInfo `json:"error"`
}

// APIError implementa la interfaz error para uso en la API
type APIError struct {
	ErrorResponse
}

// Error implementa la interfaz error
func (e APIError) Error() string {
	return e.ErrorResponse.Error.Message
}

// NewAPIError crea un nuevo error de API
func NewAPIError(errResp ErrorResponse) error {
	return &APIError{ErrorResponse: errResp}
}

// ErrorInfo representa la información del error
type ErrorInfo struct {
	Code    string        `json:"code"`
	Message string        `json:"message"`
	Details []ErrorDetail `json:"details,omitempty"`
}

// NewErrorResponse crea una nueva respuesta de error
func NewErrorResponse(code ErrorCode, message string) ErrorResponse {
	return ErrorResponse{
		Error: ErrorInfo{
			Code:    string(code),
			Message: message,
		},
	}
}

// NewValidationError crea un error de validación con detalles
func NewValidationError(message string, details []ErrorDetail) ErrorResponse {
	return ErrorResponse{
		Error: ErrorInfo{
			Code:    string(ErrorCodeInvalidRequest),
			Message: message,
			Details: details,
		},
	}
}

// NewConflictError crea un error de conflicto (idempotencia)
func NewConflictError(message string) ErrorResponse {
	return ErrorResponse{
		Error: ErrorInfo{
			Code:    string(ErrorCodeConflict),
			Message: message,
		},
	}
}

// NewUnauthorizedError crea un error de autenticación
func NewUnauthorizedError(message string) ErrorResponse {
	return ErrorResponse{
		Error: ErrorInfo{
			Code:    string(ErrorCodeUnauthorized),
			Message: message,
		},
	}
}

// NewForbiddenError crea un error de permisos
func NewForbiddenError(message string) ErrorResponse {
	return ErrorResponse{
		Error: ErrorInfo{
			Code:    string(ErrorCodeForbidden),
			Message: message,
		},
	}
}

// NewNotFoundError crea un error de recurso no encontrado
func NewNotFoundError(message string) ErrorResponse {
	return ErrorResponse{
		Error: ErrorInfo{
			Code:    string(ErrorCodeNotFound),
			Message: message,
		},
	}
}

// NewRateLimitedError crea un error de rate limiting
func NewRateLimitedError(message string, retryAfter time.Duration) ErrorResponse {
	return ErrorResponse{
		Error: ErrorInfo{
			Code:    string(ErrorCodeRateLimited),
			Message: message,
		},
	}
}

// NewInternalError crea un error interno del servidor
func NewInternalError(message string) ErrorResponse {
	return ErrorResponse{
		Error: ErrorInfo{
			Code:    string(ErrorCodeInternal),
			Message: message,
		},
	}
}
