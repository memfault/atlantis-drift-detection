package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRedactCredentials(t *testing.T) {
	require.Equal(t,
		"git clone https://REDACTED@github.com/memfault/memfault.git /tmp/x",
		redactCredentials("git clone https://x-access-token:ghp_secret@github.com/memfault/memfault.git /tmp/x"))
	require.Equal(t, "https://github.com/a/b", redactCredentials("https://github.com/a/b"))
}
