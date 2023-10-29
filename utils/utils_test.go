package utils

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTrimmedURL(t *testing.T) {
	withSlash, err := url.Parse("http://somewhere.com/")
	require.Equal(t, nil, err)
	withoutSlash, err := url.Parse("http://somewhere.com")
	require.Equal(t, nil, err)

	require.Equal(t, TrimmedURL(withSlash), TrimmedURL(withoutSlash))

	withSlash, err = url.Parse("http://somewhere.com/with/path/")
	require.Equal(t, nil, err)
	withoutSlash, err = url.Parse("http://somewhere.com/with/path")
	require.Equal(t, nil, err)

	require.Equal(t, TrimmedURL(withSlash), TrimmedURL(withoutSlash))
}

func TestExists(t *testing.T) {
	tmpDir := t.TempDir()
	stat, err := PathExists(tmpDir)
	require.Equal(t, nil, err)
	require.Equal(t, true, stat)

	stat, err = PathExists(tmpDir + "/non-existent-path")
	require.Equal(t, nil, err)
	require.Equal(t, false, stat)

	subdir := filepath.Join(tmpDir, "unreadable")
	err = os.MkdirAll(subdir, 0700)
	require.Equal(t, nil, err)

	hiddenFile := filepath.Join(subdir, "somefile.tgz")
	fd, err := os.Create(hiddenFile)
	require.Equal(t, nil, err)
	fd.Close()

	stat, err = PathExists(hiddenFile)
	require.Equal(t, nil, err)
	require.Equal(t, true, stat)

	os.Chmod(subdir, 0)

	stat, err = PathExists(hiddenFile)
	require.True(t, os.IsPermission(err))

	os.Chmod(subdir, 0700)
}
