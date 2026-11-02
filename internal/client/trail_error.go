package client

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/gate/gate-cli/internal/operations"
	"github.com/gate/gate-cli/internal/output"
)

// TrailCommandError locates existing Trail SDK failures through the authoritative
// operation catalog. It only constructs diagnostics; it never sends a request.
func TrailCommandError(err error, response *http.Response, operationID, settle, body string) error {
	return trailCommandError(err, response, operationID, settle, body, operations.LookupByID)
}

func trailCommandError(err error, response *http.Response, operationID, settle, body string, resolve func(string) (operations.Operation, error)) error {
	var existing *output.GateError
	if errors.As(err, &existing) {
		return err
	}
	switch operationID {
	case "createTrailOrder", "stopTrailOrder", "stopAllTrailOrders", "getTrailOrders", "getTrailOrderDetail", "updateTrailOrder", "getTrailOrderChangeLog":
	default:
		return commandError(err, response, nil)
	}
	op, lookupErr := resolve(operationID)
	if lookupErr != nil || op.OperationID != operationID || op.Method == "" || strings.Count(op.Path, "{settle}") != 1 {
		return commandError(err, response, nil)
	}
	// Match the released SDK's path substitution without changing its input or
	// validation. Do not copy hosts, query values or credential headers here.
	path := strings.Replace(op.Path, "{settle}", url.QueryEscape(settle), 1)
	return commandError(err, response, &output.RequestInfo{Method: op.Method, URL: path, Body: body})
}
