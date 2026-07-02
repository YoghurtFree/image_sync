package syncer

import (
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
)

func TestCopyInvalidSourceReference(t *testing.T) {
	srcAuth := authn.FromConfig(authn.AuthConfig{Username: "user", Password: "pass"})
	dstAuth := authn.FromConfig(authn.AuthConfig{Username: "user", Password: "pass"})

	err := Copy("not a valid image reference!!!", "other-registry.example.com/image:tag", srcAuth, dstAuth)
	if err == nil {
		t.Error("Copy should fail for invalid source reference")
	}
}

func TestCopyInvalidDestinationReference(t *testing.T) {
	srcAuth := authn.FromConfig(authn.AuthConfig{Username: "user", Password: "pass"})
	dstAuth := authn.FromConfig(authn.AuthConfig{Username: "user", Password: "pass"})

	err := Copy("docker.io/library/alpine:latest", "not a valid image reference!!!", srcAuth, dstAuth)
	if err == nil {
		t.Error("Copy should fail for invalid destination reference")
	}
}
