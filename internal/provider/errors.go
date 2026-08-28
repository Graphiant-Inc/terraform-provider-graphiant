package provider

import (
	"errors"
	"fmt"

	sdk "github.com/Graphiant-Inc/graphiant-sdk-go"
)

// apiErrorDetail builds a diagnostic-friendly message from an SDK call error,
// including the raw response body when the SDK captured one (it carries the
// server's actual error text, which the bare Go error usually doesn't).
func apiErrorDetail(err error) string {
	var apiErr *sdk.GenericOpenAPIError
	if errors.As(err, &apiErr) {
		if body := apiErr.Body(); len(body) > 0 {
			return fmt.Sprintf("%s: %s", apiErr.Error(), string(body))
		}
	}
	return err.Error()
}
