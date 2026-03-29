package auth

import (
	"testing"
)

func TestGenerateCodeVerifier(t *testing.T) {
	v1, err := GenerateCodeVerifier()
	if err != nil {
		t.Fatalf("GenerateCodeVerifier failed: %v", err)
	}
	if len(v1) < 43 {
		t.Errorf("Expected verifier length >= 43, got %d", len(v1))
	}

	// Should be unique
	v2, _ := GenerateCodeVerifier()
	if v1 == v2 {
		t.Error("Expected different verifiers on successive calls")
	}
}

func TestGenerateCodeChallenge(t *testing.T) {
	verifier := "test_verifier_string_that_is_long_enough_for_pkce"
	challenge := GenerateCodeChallenge(verifier)

	if challenge == "" {
		t.Error("Expected non-empty challenge")
	}
	if challenge == verifier {
		t.Error("Challenge should not equal verifier")
	}

	// Same verifier should produce same challenge
	challenge2 := GenerateCodeChallenge(verifier)
	if challenge != challenge2 {
		t.Error("Same verifier should produce same challenge")
	}

	// Different verifier should produce different challenge
	challenge3 := GenerateCodeChallenge("different_verifier_string")
	if challenge == challenge3 {
		t.Error("Different verifiers should produce different challenges")
	}
}

func TestGenerateState(t *testing.T) {
	s1, err := GenerateState()
	if err != nil {
		t.Fatalf("GenerateState failed: %v", err)
	}
	if s1 == "" {
		t.Error("Expected non-empty state")
	}

	// Should be unique
	s2, _ := GenerateState()
	if s1 == s2 {
		t.Error("Expected different states on successive calls")
	}
}

func TestBase64URLEncode_NoPadding(t *testing.T) {
	// base64URLEncode should not contain padding characters
	result := base64URLEncode([]byte("test"))
	for _, c := range result {
		if c == '=' {
			t.Error("base64URLEncode should not contain padding")
		}
	}
}
