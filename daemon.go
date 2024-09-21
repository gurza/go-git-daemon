package daemon

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-git/v5/plumbing/protocol/packp"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/server"
)

func InfoRefs(ctx context.Context, fs billy.Filesystem, repoPath string, w io.Writer) error {
	srv := server.NewServer(server.NewFilesystemLoader(fs))

	ep, err := transport.NewEndpoint(repoPath)
	if err != nil {
		return errors.New("failed to create endpoint")
	}

	sess, err := srv.NewUploadPackSession(ep, nil)
	if err != nil {
		return errors.New("failed to create session")
	}
	defer sess.Close()

	refs, err := sess.AdvertisedReferencesContext(ctx)
	if err != nil {
		return errors.New("failed to advertise references")
	}
	return refs.Encode(w)
}

func UploadPack(ctx context.Context, fs billy.Filesystem, repoPath string, r io.Reader, w io.Writer) error {
	srv := server.NewServer(server.NewFilesystemLoader(fs))

	ep, err := transport.NewEndpoint(repoPath)
	if err != nil {
		return fmt.Errorf("failed to create endpoint: %w", err)
	}

	sess, err := srv.NewUploadPackSession(ep, nil)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer sess.Close()

	req := packp.NewUploadPackRequest()
	if err := req.Decode(r); err != nil {
		return fmt.Errorf("failed to decode request: %w", err)
	}

	resp, err := sess.UploadPack(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to process upload-pack request: %w", err)
	}

	if err := resp.Encode(w); err != nil {
		return fmt.Errorf("failed to encode response: %w", err)
	}

	return nil
}
