package live

import (
	"gosm/pkg/protocol/httpflv"
)

// OnHTTPFlvSubscribe .
func (mgmt *RoomMgmt) OnHTTPFlvSubscribe(stream *httpflv.NetStream) error {
	return nil
}

// OnHTTPFlvUnSubscribe .
func (mgmt *RoomMgmt) OnHTTPFlvUnSubscribe(stream *httpflv.NetStream) error {
	return nil
}
