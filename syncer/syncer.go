package syncer

import (
	"fmt"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
)

func Copy(srcImage, dstImage string, srcAuth, dstAuth authn.Authenticator) error {
	_, err := name.ParseReference(srcImage)
	if err != nil {
		return fmt.Errorf("invalid source image %q: %w", srcImage, err)
	}

	_, err = name.ParseReference(dstImage)
	if err != nil {
		return fmt.Errorf("invalid destination image %q: %w", dstImage, err)
	}

	// crane.Copy uses same auth for src and dst, so pull + push separately
	img, err := crane.Pull(srcImage, crane.WithAuth(srcAuth))
	if err != nil {
		return fmt.Errorf("pull %s failed: %w", srcImage, err)
	}

	err = crane.Push(img, dstImage, crane.WithAuth(dstAuth))
	if err != nil {
		return fmt.Errorf("push %s failed: %w", dstImage, err)
	}

	return nil
}
