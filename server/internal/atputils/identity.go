package atputils
import (
	"context"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"errors"
)

func GetHandleFromDid(ctx context.Context, did string) (string, error) {
	sdid, err := syntax.ParseDID(did)
	if err != nil {
		return "", errors.New("did did not parse: " + err.Error())
	}
	resolver := identity.DefaultDirectory()

	ident, err := resolver.LookupDID(ctx, sdid)
	if err != nil {
		return "", errors.New("failed to lookupDID: " + err.Error())
	}
	return ident.Handle.String(), nil
}

func GetDidFromHandle(ctx context.Context, handle string) (string, error) {
	shandle, err := syntax.ParseHandle(handle)
	if err != nil {
		return "", errors.New("handle did not parse: " + err.Error())
	}
	resolver := identity.DefaultDirectory()
	ident, err := resolver.LookupHandle(ctx,shandle)
	if err != nil {
		return "", errors.New("failed to lookupHandle: " + err.Error())
	}
	return ident.DID.String(), nil
}
