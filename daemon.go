package daemon

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-git/v5/plumbing/format/pktline"
	"github.com/go-git/go-git/v5/plumbing/protocol/packp"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/server"
)

type ServiceType string

const (
	ServiceUploadPack  ServiceType = "git-upload-pack"
	ServiceReceivePack ServiceType = "git-receive-pack"
)

// newTransport creates a transport and endpoint pair for git operations using
// a filesystem-backed repository.
func newTransport(fs billy.Filesystem, repo string) (transport.Transport, *transport.Endpoint, error) {
	srv := server.NewServer(server.NewFilesystemLoader(fs))
	ep, err := transport.NewEndpoint(repo)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid repository endpoint: %w", err)
	}
	return srv, ep, nil
}

// newSession creates a service session for git operations using existing
// transport components.
func newSessionWithTransport(srv transport.Transport, ep *transport.Endpoint, svc ServiceType) (transport.Session, error) {
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
			return nil, fmt.Errorf("repository not found: %q", ep.Path)
		}
		return nil, fmt.Errorf("failed to create %s session for %q: %w", svc, ep.Path, err)
	}

	return sess, nil
}

// newSession creates a service session for git operations using
// a filesystem-backed repository.
func newSession(fs billy.Filesystem, repo string, svc ServiceType) (transport.Session, error) {
	srv, ep, err := newTransport(fs, repo)
	if err != nil {
		return nil, err
	}

	sess, err := newSessionWithTransport(srv, ep, svc)
	if err != nil {
		return nil, err
	}

	return sess, nil
}

// InfoRefs retrieves the advertised references for the given repository.
func InfoRefs(ctx context.Context, fs billy.Filesystem, repo string, svc ServiceType) (*packp.AdvRefs, error) {
	sess, err := newSession(fs, repo, svc)
	if err != nil {
		return nil, err
	}
	defer sess.Close()

	res, err := sess.AdvertisedReferencesContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve advertised references: %w", err)
	}
	res.Prefix = [][]byte{
		[]byte(fmt.Sprintf("# service=%s", svc)),
		pktline.Flush,
	}
	// FIXME: add no-thin capability to work-around some go-git limitations
	err = res.Capabilities.Add("no-thin")
	if err != nil {
		return nil, fmt.Errorf("failed to add no-thin capability: %w", err)
	}
	return res, nil
}

// UploadPack processes the git upload-pack operation for the given repository.
func UploadPack(ctx context.Context, fs billy.Filesystem, repo string, r io.Reader) (*packp.UploadPackResponse, error) {
	req := packp.NewUploadPackRequest()
	if err := req.Decode(r); err != nil {
		return nil, fmt.Errorf("failed to decode upload-pack request: %w", err)
	}

	sess, err := newSession(fs, repo, ServiceReceivePack)
	if err != nil {
		return nil, err
	}
	defer sess.Close()

	upSess, err := asUploadPackSession(sess)
	if err != nil {
		return nil, err
	}

	res, err := upSess.UploadPack(ctx, req)
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

	sess, err := newSession(fs, repo, ServiceReceivePack)
	if err != nil {
		return nil, err
	}
	defer sess.Close()

	rpSess, err := asReceivePackSession(sess)
	if err != nil {
		return nil, err
	}

	res, err := rpSess.ReceivePack(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to process receive-pack: %w", err)
	}

	return res, nil
}
