package rtmp

import (
	"fmt"
	"io"

	"gosm/pkg/protocol/amf"
)

// Writer .
type Writer interface {
	io.Writer
	io.ByteWriter
}

// Command rtmp AMF0/AMF3 command message
type Command struct {
	Name          string
	TransactionID uint32
	Objects       []interface{}
	UserArguments []interface{}
}

// WriteTo write command to writer
func (cmd *Command) WriteTo(w Writer) (n int, err error) {
	amf := amf.AMF0{}
	number := 0
	// command name
	if number, err = amf.WriteString(w, cmd.Name); err != nil {
		return n, fmt.Errorf("RTMP: command, error to write name, %s", err)
	}
	n = n + number
	// transaction id
	if number, err = amf.WriteNumber(w, float64(cmd.TransactionID)); err != nil {
		return n, fmt.Errorf("RTMP: command, error to write transaction id, %s", err)
	}
	n = n + number
	// objects
	for _, object := range cmd.Objects {
		if number, err = amf.WriteTo(w, object); err != nil {
			return n, fmt.Errorf("RTMP: command, error to write object, %s", err)
		}
		n = n + number
	}
	// user arguments
	for _, object := range cmd.UserArguments {
		if number, err = amf.WriteTo(w, object); err != nil {
			return n, fmt.Errorf("RTMP: command, error to write user argument, %s", err)
		}
		n = n + number
	}
	return
}

func (cmd *Command) connectSuccess(transactionID uint32) *Command {
	object := make(map[string]interface{})
	object["fmsVer"] = "GOSM/0,0,1,0"
	object["author"] = "Snake"
	object["capabilities"] = "31"

	argument := make(map[string]interface{})
	argument["level"] = "status"
	argument["code"] = "NetConnection.Connect.Success"
	argument["description"] = "Connection succeeded."

	return &Command{
		Name:          "_result",
		TransactionID: transactionID,
		Objects:       []interface{}{object},
		UserArguments: []interface{}{argument}}
}

func (cmd *Command) connectError(transactionID uint32) *Command {
	argument := make(map[string]interface{})
	argument["level"] = "status"
	argument["code"] = "NetConnection.Connect.Rejected"
	argument["description"] = "Connection rejected."

	return &Command{
		Name:          "_error",
		TransactionID: transactionID,
		Objects:       []interface{}{nil},
		UserArguments: []interface{}{argument}}
}

func (cmd *Command) createStreamSuccess(transactionID uint32, streamID uint32) *Command {
	return &Command{
		Name:          "_result",
		TransactionID: transactionID,
		Objects:       []interface{}{nil},
		UserArguments: []interface{}{streamID}}
}

func (cmd *Command) publishStream() *Command {
	argument := make(map[string]interface{})
	argument["level"] = "status"
	argument["code"] = "NetStream.Publish.Start"
	argument["description"] = "Start publising."

	return &Command{
		Name:          "onStatus",
		TransactionID: 0, // transaction id for netstream always 0
		Objects:       []interface{}{nil},
		UserArguments: []interface{}{argument}}
}

func (cmd *Command) resetStream() *Command {
	argument := make(map[string]interface{})
	argument["level"] = "status"
	argument["code"] = "NetStream.Play.Reset"
	argument["description"] = "Playing and resetting stream."

	return &Command{
		Name:          "onStatus",
		TransactionID: 0, // transaction id for netstream always 0
		Objects:       []interface{}{nil},
		UserArguments: []interface{}{argument}}
}

func (cmd *Command) startStream() *Command {
	argument := make(map[string]interface{})
	argument["level"] = "status"
	argument["code"] = "NetStream.Play.Start"
	argument["description"] = "Started playing stream."

	return &Command{
		Name:          "onStatus",
		TransactionID: 0, // transaction id for netstream always 0
		Objects:       []interface{}{nil},
		UserArguments: []interface{}{argument}}
}
