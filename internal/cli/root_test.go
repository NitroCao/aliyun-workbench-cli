package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nitrocao/aliyun-workbench-cli/internal/workbench"
)

func TestLoginTicketReadsEnvironment(t *testing.T) {
	t.Setenv(envLoginTicket, " primary ")

	assert.Equal(t, "primary", loginTicket())
}

func TestRootCommandVersion(t *testing.T) {
	t.Parallel()

	cmd := (&app{}).newRootCommand()

	assert.Equal(t, appVersion, cmd.Version)
}

func TestPrintInstancesTableAlignsLongNames(t *testing.T) {
	t.Parallel()

	instances := []workbench.ECSInstance{
		{
			InstanceID:   "i-short",
			InstanceName: "short",
			Status:       "Running",
			OSName:       "Linux",
		},
		{
			InstanceID:   "i-long",
			InstanceName: "worker-k8s-for-cs-cc1523474193240049f61c3b0f23dd8c9",
			Status:       "Stopped",
			OSName:       "Linux",
		},
	}

	var out bytes.Buffer
	require.NoError(t, printInstancesTable(&out, instances))

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	require.Len(t, lines, 3, out.String())

	statusColumn := strings.Index(lines[0], "STATUS")
	runningColumn := strings.Index(lines[1], "Running")
	stoppedColumn := strings.Index(lines[2], "Stopped")
	require.NotEqual(t, -1, statusColumn, out.String())
	assert.Equal(t, statusColumn, runningColumn, out.String())
	assert.Equal(t, statusColumn, stoppedColumn, out.String())
}
