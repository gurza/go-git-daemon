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

// GitServiceName identifies Git protocol services.
type GitServiceName string

const (
	GitUploadPack  GitServiceName = "git-upload-pack"  // Handles fetch/clone operations
	GitReceivePack GitServiceName = "git-receive-pack" // Handles push operations
)

// Service handles Git daemon protocol operations for a repository.
type Service struct {
	srv transport.Transport
	ep  *transport.Endpoint
}

func NewService(fs billy.Filesystem, repo string) (*Service, error) {
	srv := server.NewServer(server.NewFilesystemLoader(fs))
	ep, err := transport.NewEndpoint(repo)
	if err != nil {
		return nil, fmt.Errorf("invalid repository endpoint: %w", err)
	}
	return &Service{srv: srv, ep: ep}, nil
}

func (s *Service) newSession(nm GitServiceName) (transport.Session, error) {
	var (
		sess transport.Session
		err  error
	)

	switch nm {
	case GitUploadPack:
		sess, err = s.srv.NewUploadPackSession(s.ep, nil)
	case GitReceivePack:
		sess, err = s.srv.NewReceivePackSession(s.ep, nil)
	default:
		return nil, fmt.Errorf("unsupported service: %s", nm)
	}

	if err != nil {
		// FIXME: Add other specific error checks
		if errors.Is(err, transport.ErrRepositoryNotFound) {
			return nil, fmt.Errorf("repository not found: %q", s.ep.Path)
		}
		return nil, fmt.Errorf("failed to create %s session for %q: %w", nm, s.ep.Path, err)
	}

	return sess, nil
}

// InfoRefs returns advertised references for the specified Git service.
func (s *Service) InfoRefs(ctx context.Context, nm GitServiceName) (*packp.AdvRefs, error) {
	sess, err := s.newSession(nm)
	if err != nil {
		return nil, err
	}
	defer sess.Close()

	res, err := sess.AdvertisedReferencesContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve advertised references: %w", err)
	}

	// Set prefix and capabilities
	if err := s.configureAdvRefs(res, nm); err != nil {
		return nil, err
	}

	return res, nil
}

func (s *Service) configureAdvRefs(res *packp.AdvRefs, nm GitServiceName) error {
	res.Prefix = [][]byte{
		[]byte(fmt.Sprintf("# service=%s", nm)),
		pktline.Flush,
	}

	// Add the no-thin capability to work around go-git limitations
	if err := res.Capabilities.Add("no-thin"); err != nil {
		return fmt.Errorf("failed to add no-thin capability: %w", err)
	}

	return nil
}

// UploadPack processes fetch/clone protocol requests.
func (s *Service) UploadPack(ctx context.Context, r io.Reader) (*packp.UploadPackResponse, error) {
	req := packp.NewUploadPackRequest()
	if err := req.Decode(r); err != nil {
		return nil, fmt.Errorf("failed to decode upload-pack request: %w", err)
	}

	sess, err := s.newSession(GitUploadPack)
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

// ReceivePack processes push protocol requests.
func (s *Service) ReceivePack(ctx context.Context, r io.Reader) (*packp.ReportStatus, error) {
	req := packp.NewReferenceUpdateRequest()
	if err := req.Decode(r); err != nil {
		return nil, fmt.Errorf("failed to decode receive-pack request: %w", err)
	}

	sess, err := s.newSession(GitReceivePack)
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
