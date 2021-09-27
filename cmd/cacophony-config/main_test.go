package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseArgs(t *testing.T) {
	args := []string{"cat.foo=bar", "bar.foo=cat"}
	expected := []setting{
		{section: "cat", field: "foo", value: "bar"},
		{section: "bar", field: "foo", value: "cat"},
	}
	sections, err := getNewSettings(args)
	require.NoError(t, err)
	require.Equal(t, expected, sections)
}

func TestBadArgs(t *testing.T) {
	_, err := getNewSettings([]string{"cat.foobar"})
	require.Error(t, err)

	_, err = getNewSettings([]string{"catfoo=bar"})
	require.Error(t, err)

	_, err = getNewSettings([]string{"cat.dog=foo=bar"})
	require.Error(t, err)

	_, err = getNewSettings([]string{"cat.dog.foo=bar"})
	require.Error(t, err)
}
