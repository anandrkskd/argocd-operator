package argoutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetOperatorNamespace(t *testing.T) {
	t.Run("reads namespace from file", func(t *testing.T) {
		dir := t.TempDir()
		nsFile := filepath.Join(dir, "namespace")
		require.NoError(t, os.WriteFile(nsFile, []byte("operator-ns\n"), 0644))

		old := OperatorNamespaceFile
		OperatorNamespaceFile = nsFile
		defer func() { OperatorNamespaceFile = old }()

		ns, err := GetOperatorNamespace()
		assert.NoError(t, err)
		assert.Equal(t, "operator-ns", ns)
	})

	t.Run("returns error when file empty", func(t *testing.T) {
		dir := t.TempDir()
		nsFile := filepath.Join(dir, "namespace")
		require.NoError(t, os.WriteFile(nsFile, []byte("  \n"), 0644))

		old := OperatorNamespaceFile
		OperatorNamespaceFile = nsFile
		defer func() { OperatorNamespaceFile = old }()

		ns, err := GetOperatorNamespace()
		assert.Error(t, err)
		assert.Empty(t, ns)
	})

	t.Run("returns error when file missing", func(t *testing.T) {
		old := OperatorNamespaceFile
		OperatorNamespaceFile = "/nonexistent/path/namespace"
		defer func() { OperatorNamespaceFile = old }()

		ns, err := GetOperatorNamespace()
		assert.Error(t, err)
		assert.Empty(t, ns)
	})
}
