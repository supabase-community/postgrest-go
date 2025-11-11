package postgrest

import "fmt"

// DefaultHeaders returns the default headers for PostgREST requests
func DefaultHeaders() map[string]string {
	return map[string]string{
		"X-Client-Info": fmt.Sprintf("postgrest-go/%s", version),
	}
}
