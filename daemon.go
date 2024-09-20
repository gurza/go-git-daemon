package daemon

import (
	"context"
	"errors"
	"io"

	"github.com/go-git/go-billy/v5"
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
