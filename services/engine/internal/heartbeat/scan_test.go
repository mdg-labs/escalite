package heartbeat

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAlertSourceFromState(t *testing.T) {
	source, err := AlertSourceFromState([]byte(`{"source":"heartbeat","current_step":1}`))
	require.NoError(t, err)
	require.Equal(t, "heartbeat", source)

	empty, err := AlertSourceFromState(nil)
	require.NoError(t, err)
	require.Empty(t, empty)
}
