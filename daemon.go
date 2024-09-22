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

func newSession(srv transport.Transport, ep *transport.Endpoint, svc ServiceType) (transport.Session, error) {
	var (
		sess transport.Session
		err  error
	)

	switch svc {
	case ServiceUploadPack:
		sess, err = srv.NewUploadPackSession(ep, nil)
	case ServiceReceivePack:
		sess, err = srv.NewReceivePackSession(ep, nil)
	default:
		return nil, fmt.Errorf("unsupported service type: %s", svc)
	}

	if err != nil {
		if errors.Is(err, transport.ErrRepositoryNotFound) {
			return nil, fmt.Errorf("repository not found: %s", ep.Path)
		}
		return nil, fmt.Errorf("failed to create %s session for %s: %w", svc, ep.Path, err)
	}
	return sess, nil
}

// InfoRefs retrieves the advertised references for the given repository.
func InfoRefs(ctx context.Context, fs billy.Filesystem, repo string, svc ServiceType) (*packp.AdvRefs, error) {
	srv := server.NewServer(server.NewFilesystemLoader(fs))
	ep, err := transport.NewEndpoint(repo)
	if err != nil {
		return nil, fmt.Errorf("invalid repository endpoint: %w", err)
	}
	sess, err := newSession(srv, ep, svc)
	if err != nil {
		return nil, err
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
	sessg, err := newSession(srv, ep, ServiceUploadPack)
	if err != nil {
		return nil, err
	}
	defer sessg.Close()

	sess, ok := sessg.(transport.UploadPackSession)
	if !ok {
		return nil, fmt.Errorf("session does not implement UploadPackSession")
	}
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
	sessg, err := newSession(srv, ep, ServiceReceivePack)
	if err != nil {
		return nil, err
	}
	defer sessg.Close()

	sess, ok := sessg.(transport.ReceivePackSession)
	if !ok {
		return nil, fmt.Errorf("session does not implement UploadPackSession")
	}
	res, err := sess.ReceivePack(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to process receive-pack: %w", err)
	}
	return res, nil
}
