package postgrest

// PostgrestError represents an error response from PostgREST
// https://postgrest.org/en/stable/api.html?highlight=options#errors-and-http-status-codes
type PostgrestError struct {
	Message string
	Details string
	Hint    string
	Code    string
}

func (e *PostgrestError) Error() string {
	return e.Message
}

// NewPostgrestError creates a new PostgrestError
func NewPostgrestError(message, details, hint, code string) *PostgrestError {
	return &PostgrestError{
		Message: message,
		Details: details,
		Hint:    hint,
		Code:    code,
	}
}
