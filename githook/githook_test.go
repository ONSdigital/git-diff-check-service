package githook_test

import (
	"bytes"
	"crypto/subtle"
	"fmt"
	"net/http"
	"testing"

	"github.com/ONSdigital/git-diff-check-service/githook"
)

func TestSignPayload(t *testing.T) {
	expected := []byte("5d61605c3feea9799210ddcb71307d4ba264225f")

	s := githook.SignPayload([]byte("{}"), []byte("secret"))
	if subtle.ConstantTimeCompare(s, expected) != 1 {
		t.Errorf("Signature compute incorrect, got %s, expected %s", s, expected)
	}
}

func TestParse(t *testing.T) {

	body := bytes.NewReader([]byte(`{}`))

	r, err := http.NewRequest("POST", "http://localhost", body)
	if err != nil {
		t.Fatalf("Failed to create new http.Request: %v", err)
	}
	fmt.Println(r)

}
