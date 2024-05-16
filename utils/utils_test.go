package utils

import (
	"encoding/hex"
	"math/big"
	"testing"
)

func TestParseHexValue(t *testing.T) {
	tests := []struct {
		hexStr string
		want   *big.Int
	}{
		{"0x1", big.NewInt(1)},
		{"0x10", big.NewInt(16)},
		{"0xff", big.NewInt(255)},
		{"0x1234567890abcdef", big.NewInt(0x1234567890abcdef)},
	}

	for _, tt := range tests {
		got := ParseHexValue(tt.hexStr)
		if got.Cmp(tt.want) != 0 {
			t.Errorf("ParseHexValue(%s) = %v, want %v", tt.hexStr, got, tt.want)
		}
	}
}

func TestRemove0xPrefix(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"0x1", "1"},
		{"0XABC", "ABC"},
		{"123", "123"},
		{"0x123", "123"},
		{"0x1234567890abcdef", "1234567890abcdef"},
	}

	for _, tt := range tests {
		got := Remove0xPrefix(tt.input)
		if got != tt.want {
			t.Errorf("Remove0xPrefix(%s) = %s, want %s", tt.input, got, tt.want)
		}
	}
}

func TestKeccak256(t *testing.T) {
	tests := []struct {
		input []byte
		want  string
	}{
		{[]byte("hello"), "1c8aff950685c2ed4bc3174f3472287b56d9517b9c948127319a09a7a36deac8"},
		{[]byte("haqq"), "ede9ccb406cc78631779409e4f3d0946ec6bfc530918f2dc8f63c284d209e724"},
	}

	for _, tt := range tests {
		got := Keccak256(tt.input)
		if hex.EncodeToString(got) != tt.want {
			t.Errorf("Keccak256(%s) = %x, want %s", tt.input, got, tt.want)
		}
	}
}

func TestCalculateStorageKey(t *testing.T) {
	tests := []struct {
		addr string
		i    int
		want string
	}{
		{"0xA367C471fFEdbB3230793e0aaf045c38e57eDf98", 0, "0xece64beae9f44f327fa25deecc04fcb83b8512d3873bc0f6702645d10aaafaad"},
		{"0xA367C471fFEdbB3230793e0aaf045c38e57eDf98", 1, "0x706a64cd6ab6caa25d744643a971945a13ac5b19961a5295e0771dd24711cc34"},
		{"0xA367C471fFEdbB3230793e0aaf045c38e57eDf98", 2, "0xd20799b9ccb19c9d821e349cec115df5cfd391b8d9c5b5ea10f9cc3d4f1e801e"},
		{"0xA367C471fFEdbB3230793e0aaf045c38e57eDf98", 3, "0xd6b77ced29b77d9d8fdab16e04c4ea5d9056bc8f52f1b081d4e80c158d5e91bd"},
	}

	for _, tt := range tests {
		got := CalculateStorageKey(tt.addr, tt.i)
		if got != tt.want {
			t.Errorf("CalculateStorageKey(%s, %d) = %s, want %s", tt.addr, tt.i, got, tt.want)
		}
	}
}
