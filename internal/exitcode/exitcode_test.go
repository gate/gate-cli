package exitcode

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// Exit status and wrapped errors must survive the command's error boundary;
// otherwise automation can confuse an authentication failure with success.
func TestExitCodeErrorContract(t *testing.T) {
	for _, code := range []int{0, 1, 4, 255} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			cause := errors.New("synthetic cause")
			err := New(code, cause)
			var typed *Error
			require.ErrorAs(t, fmt.Errorf("command: %w", err), &typed)
			require.Equal(t, code, typed.Code)
			require.ErrorIs(t, err, cause)
			require.Equal(t, "synthetic cause", err.Error())
			withoutCause := New(code, nil)
			require.Equal(t, fmt.Sprintf("exit code %d", code), withoutCause.Error())
			require.Nil(t, errors.Unwrap(withoutCause))
		})
	}
}
