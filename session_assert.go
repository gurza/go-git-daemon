package daemon

import (
	"errors"

	"github.com/go-git/go-git/v5/plumbing/transport"
)

// asUploadPackSession performs type assertion to UploadPackSession.
func asUploadPackSession(sess transport.Session) (transport.UploadPackSession, error) {
	upSess, ok := sess.(transport.UploadPackSession)
	if !ok {
		return nil, errors.New("session does not implement UploadPackSession")
	}
	return upSess, nil
}

// asReceivePackSession performs type assertion to ReceivePackSession.
func asReceivePackSession(sess transport.Session) (transport.ReceivePackSession, error) {
	rpSess, ok := sess.(transport.ReceivePackSession)
	if !ok {
		return nil, errors.New("session does not implement ReceivePackSession")
	}
	return rpSess, nil
}
