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

// InfoRefs retrieves advertised references for a git repository.
func InfoRefs(ctx context.Context, fs billy.Filesystem, repo string) (*packp.AdvRefs, error) {
	srv := server.NewServer(server.NewFilesystemLoader(fs))
	ep, err := transport.NewEndpoint(repo)
	if err != nil {
		return nil, fmt.Errorf("invalid repository endpoint: %w", err)
	}
	sess, err := srv.NewUploadPackSession(ep, nil)
	if err != nil {
		if errors.Is(err, transport.ErrRepositoryNotFound) {
			return nil, fmt.Errorf("repository not found: %s", repo)
		}
		return nil, fmt.Errorf("failed to create upload pack session: %w", err)
	}
	defer sess.Close()

	refs, err := sess.AdvertisedReferencesContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve advertised references: %w", err)
	}
	return refs, nil
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
