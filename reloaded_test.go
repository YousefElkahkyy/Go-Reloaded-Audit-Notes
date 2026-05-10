package main

import (
	"testing"
)

func TestProcess(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "test Case1",
			input:    "If I make you BREAKFAST IN BED (low, 3) just say thank you instead of: how (cap) did you get in my house (up, 2) ?",
			expected: "If I make you breakfast in bed just say thank you instead of: How did you get in MY HOUSE?",
		},
		{
			name:     "test Case2",
			input:    "I have to pack 101 (bin) outfits. Packed 1a (hex) just to be sure",
			expected: "I have to pack 5 outfits. Packed 26 just to be sure",
		},
		{
			name:     "test Case3",
			input:    "Don not be sad ,because sad backwards is das . And das not good",
			expected: "Don not be sad, because sad backwards is das. And das not good",
		},
		{
			name:     "test Case4",
			input:    "harold wilson (cap, 2) : ' I am a optimist ,but a optimist an fish who carries a raincoat . '",
			expected: "Harold Wilson: 'I am an optimist, but an optimist a fish who carries a raincoat.'",
		},
		{
			name:     "test Case5",
			input:    "it was 10 (bin) ' a apple ' (cap, 2) !?",
			expected: "it was 2 'An Apple'!?",

		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := processText(tt.input)
			if got != tt.expected {
				t.Errorf("\ninput:\n%s\n\ngot:\n%s\n\nwant:\n%s", tt.input, got, tt.expected)
			}
		})
	}
}
