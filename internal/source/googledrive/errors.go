package googledrive

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"context-os/domain/contracts"
)

type googleAPIError struct {
	status  int
	message string
}

func (e googleAPIError) Error() string {
	if e.message == "" {
		return fmt.Sprintf("google api status %d", e.status)
	}
	return fmt.Sprintf("google api status %d: %s", e.status, e.message)
}

// classifyGoogleError maps Google API and connector failures into stable pipeline error kinds.
func classifyGoogleError(err error) (contracts.ErrorKind, bool) {
	if err == nil {
		return contracts.ErrorKindPermanent, false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return contracts.ErrorKindCanceled, true
	}

	var apiErr googleAPIError
	if errors.As(err, &apiErr) {
		switch {
		case apiErr.status == http.StatusTooManyRequests || apiErr.status >= http.StatusInternalServerError:
			return contracts.ErrorKindTemporary, true
		case apiErr.status == http.StatusForbidden:
			lower := strings.ToLower(apiErr.message)
			if strings.Contains(lower, "ratelimitexceeded") || strings.Contains(lower, "userratelimitexceeded") {
				return contracts.ErrorKindTemporary, true
			}
			return contracts.ErrorKindPermanent, false
		case apiErr.status == http.StatusUnauthorized || apiErr.status == http.StatusNotFound:
			return contracts.ErrorKindPermanent, false
		default:
			return contracts.ErrorKindInvalidRequest, false
		}
	}

	if strings.Contains(strings.ToLower(err.Error()), "credentials") || strings.Contains(strings.ToLower(err.Error()), "token") {
		return contracts.ErrorKindPermanent, false
	}
	return contracts.ErrorKindTemporary, true
}

// connectorError wraps a lower-level failure with Google Drive object context for pipeline provenance.
func (c connector) connectorError(req contracts.SourceRequest, objectType, objectID string, kind contracts.ErrorKind, retryable bool, err error) error {
	if objectType == "" {
		objectType = req.Metadata[contracts.MetadataObjectType]
	}
	if objectID == "" {
		objectID = req.Metadata[contracts.MetadataObjectID]
	}
	return &contracts.ConnectorError{
		Connector:  c.base.Name(),
		URI:        req.URI,
		ObjectType: objectType,
		ObjectID:   objectID,
		Kind:       kind,
		Retryable:  retryable,
		Err:        err,
	}
}
