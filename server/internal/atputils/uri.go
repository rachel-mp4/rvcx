package atputils

import (
	"fmt"
)

func URI(did string, collection string, rkey string) string {
	return fmt.Sprintf("at://%s/%s/%s", did, collection, rkey)
}
