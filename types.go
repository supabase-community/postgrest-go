package postgrest

// PostgrestResponse represents the response format from PostgREST
// https://github.com/supabase/supabase-js/issues/32
type PostgrestResponse[T any] struct {
	Error      *PostgrestError `json:"error,omitempty"`
	Data       T               `json:"data,omitempty"`
	Count      *int64          `json:"count,omitempty"`
	Status     int             `json:"status"`
	StatusText string          `json:"statusText"`
}

// PostgrestResponseSuccess represents a successful response
type PostgrestResponseSuccess[T any] struct {
	Error      *PostgrestError `json:"error"`
	Data       T               `json:"data"`
	Count      *int64          `json:"count"`
	Status     int             `json:"status"`
	StatusText string          `json:"statusText"`
}

// PostgrestResponseFailure represents a failed response
type PostgrestResponseFailure struct {
	Error      *PostgrestError `json:"error"`
	Data       interface{}     `json:"data"`
	Count      *int64          `json:"count"`
	Status     int             `json:"status"`
	StatusText string          `json:"statusText"`
}

// ClientServerOptions represents options for the client
type ClientServerOptions struct {
	PostgrestVersion string
}
