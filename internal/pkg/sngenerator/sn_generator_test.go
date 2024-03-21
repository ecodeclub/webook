package sngenerator

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const expectedSNLength = 39

func TestGenerateSequenceNumberWith(t *testing.T) {
	sng := NewSequenceNumberGeneratorWith(func(_ time.Time) int64 { return 1234554320123 }, func() string { return "nUfojcH2M5j2j3Tk5A1mf2" })

	testCases := []struct {
		name     string
		input    int64
		expected string
	}{
		{
			name:     "Generate sequence number with minimum input value",
			input:    1,
			expected: "0001",
		},
		{
			name:     "Generate sequence number with non-zero padded input",
			input:    123456789,
			expected: "6789",
		},

		{
			name:     "Generate sequence number with maximum 4-digit input value",
			input:    9999,
			expected: "9999",
		},
		{
			name:     "Generate sequence number with zero padded input",
			input:    123450000,
			expected: "0000",
		},
		{
			name:     "Generate sequence number with input value exceeding 4 digits",
			input:    10000,
			expected: "0000",
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			sn, err := sng.Generate(tc.input)

			assert.NoError(t, err)
			assert.Contains(t, sn, tc.expected)
			assert.Equal(t, expectedSNLength, len(sn))
		})
	}
}

func TestGenerateSequenceNumber(t *testing.T) {
	sn, err := NewSequenceNumberGenerator().Generate(123456789)
	assert.NoError(t, err)
	assert.Contains(t, sn, "6789")
	assert.Equal(t, expectedSNLength, len(sn))
}
