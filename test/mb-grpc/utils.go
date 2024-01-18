package mbgrpc

import (
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

func isSliceEqual[T proto.Message](a []T, b []T) bool {
	if len(a) != len(b) {
		return false
	}

	for i := 0; i < len(a); i++ {
		if !proto.Equal(a[i], b[i]) {
			return false
		}
	}

	return true
}

func formatSlice[T proto.Message] (a []T) string {
	res := "["
	for i := 0; i < len(a); i++ {
		if i != 0 {
			res += ","
		}

		res += prototext.Format(a[i])
	}
	res += "]"
	return res
}

