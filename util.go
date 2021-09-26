package udphp

import "encoding/json"

func MustJson(v interface{}) []byte {
	b, _ := json.Marshal(v)

	return b
}
