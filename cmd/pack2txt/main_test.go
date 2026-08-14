package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootCmdHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"--help"})

	err := rootCmd.Execute()
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "pack2txt")
}

func TestVersionCmd(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"version"})

	err := rootCmd.Execute()
	assert.NoError(t, err)
}

func TestPackAndUnpackCommands(t *testing.T) {
	tempDir := t.TempDir()
	srcDir := filepath.Join(tempDir, "src")
	require.NoError(t, os.MkdirAll(srcDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "hello.txt"), []byte("Hello Pack2Txt!"), 0644))

	outputTxt := filepath.Join(tempDir, "output.txt")
	destDir := filepath.Join(tempDir, "dest")

	// Test pack command
	rootCmd.SetArgs([]string{"pack", srcDir, "-o", outputTxt})
	err := rootCmd.Execute()
	assert.NoError(t, err)
	assert.FileExists(t, outputTxt)

	// Test inspect command
	rootCmd.SetArgs([]string{"inspect", outputTxt})
	err = rootCmd.Execute()
	assert.NoError(t, err)

	// Test unpack command
	rootCmd.SetArgs([]string{"unpack", outputTxt, "-d", destDir, "-f"})
	err = rootCmd.Execute()
	assert.NoError(t, err)

	restoredFile := filepath.Join(destDir, "hello.txt")
	assert.FileExists(t, restoredFile)
	content, err := os.ReadFile(restoredFile)
	assert.NoError(t, err)
	assert.Equal(t, "Hello Pack2Txt!", string(content))
}
