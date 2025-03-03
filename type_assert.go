package daemon

import (
	"fmt"

	"github.com/go-git/go-git/v5/plumbing/transport"
)

// asUploadPackSession safely converts a [transport.Session]
// to a [transport.UploadPackSession].
func asUploadPackSession(sess transport.Session) (transport.UploadPackSession, error) {
	upSess, ok := sess.(transport.UploadPackSession)
	if !ok {
		return nil, fmt.Errorf("invalid session type: %T, expected UploadPackSession", sess)
	}
	return upSess, nil
}

// asReceivePackSession safely converts a [transport.Session]
// to a [transport.ReceivePackSession].
func asReceivePackSession(sess transport.Session) (transport.ReceivePackSession, error) {
	rpSess, ok := sess.(transport.ReceivePackSession)
	if !ok {
		return nil, fmt.Errorf("invalid session type: %T, expected ReceivePackSession", sess)
	}
	return rpSess, nil
}
