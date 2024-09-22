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

type ServiceType string

const (
	ServiceUploadPack  ServiceType = "git-upload-pack"
	ServiceReceivePack ServiceType = "git-receive-pack"
)

// InfoRefs retrieves the advertised references for the given repository.
func InfoRefs(ctx context.Context, fs billy.Filesystem, repo string, svc ServiceType) (*packp.AdvRefs, error) {
	srv := server.NewServer(server.NewFilesystemLoader(fs))
	ep, err := transport.NewEndpoint(repo)
	if err != nil {
		return nil, fmt.Errorf("invalid repository endpoint: %w", err)
	}

	var sess transport.Session
	switch svc {
	case ServiceUploadPack:
		sess, err = srv.NewUploadPackSession(ep, nil)
		if err != nil {
			if errors.Is(err, transport.ErrRepositoryNotFound) {
				return nil, fmt.Errorf("repository not found: %s", repo)
			}
			return nil, fmt.Errorf("failed to create upload pack session: %w", err)
		}
	case ServiceReceivePack:
		sess, err = srv.NewReceivePackSession(ep, nil)
		if err != nil {
			if errors.Is(err, transport.ErrRepositoryNotFound) {
				return nil, fmt.Errorf("repository not found: %s", repo)
			}
			return nil, fmt.Errorf("failed to create upload pack session: %w", err)
		}
	}
	defer sess.Close()

	res, err := sess.AdvertisedReferencesContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve advertised references: %w", err)
	}
	return res, nil
}

// UploadPack processes the git upload-pack operation for the given repository.
func UploadPack(ctx context.Context, fs billy.Filesystem, repo string, r io.Reader) (*packp.UploadPackResponse, error) {
	req := packp.NewUploadPackRequest()
	if err := req.Decode(r); err != nil {
		return nil, fmt.Errorf("failed to decode upload-pack request: %w", err)
	}

	srv := server.NewServer(server.NewFilesystemLoader(fs))
	ep, err := transport.NewEndpoint(repo)
	if err != nil {
		return nil, fmt.Errorf("failed to create endpoint: %w", err)
	}
	sess, err := srv.NewUploadPackSession(ep, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	defer sess.Close()

	res, err := sess.UploadPack(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to process upload-pack: %w", err)
	}
	return res, nil
}

// ReceivePack processes the git receive-pack operation for the given
// repository.
func ReceivePack(ctx context.Context, fs billy.Filesystem, repo string, r io.Reader) (*packp.ReportStatus, error) {
	req := packp.NewReferenceUpdateRequest()
	if err := req.Decode(r); err != nil {
		return nil, fmt.Errorf("failed to decode receive-pack request: %w", err)
	}

	srv := server.NewServer(server.NewFilesystemLoader(fs))
	ep, err := transport.NewEndpoint(repo)
	if err != nil {
		return nil, fmt.Errorf("failed to create endpoint: %w", err)
	}
	sess, err := srv.NewReceivePackSession(ep, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create receive-pack session: %w", err)
	}
	defer sess.Close()

	res, err := sess.ReceivePack(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to process receive-pack: %w", err)
	}
	return res, nil
}
