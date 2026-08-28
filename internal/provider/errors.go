package provider

import (
	"fmt"

	sdk "github.com/Graphiant-Inc/graphiant-sdk-go"
)

// apiErrorDetail builds a diagnostic-friendly message from an SDK call error,
// including the raw response body when the SDK captured one (it carries the
// server's actual error text, which the bare Go error usually doesn't).
func apiErrorDetail(err error) string {
	if apiErr, ok := err.(sdk.GenericOpenAPIError); ok {
		if body := apiErr.Body(); len(body) > 0 {
			return fmt.Sprintf("%s: %s", apiErr.Error(), string(body))
		}
	}
	return err.Error()
}
