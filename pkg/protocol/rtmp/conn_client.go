package rtmp

import (
	"bytes"
	"fmt"
	"gosm/pkg/avformat"
	"net"
	"net/url"
	"strings"
	"sync/atomic"
)

type Client struct {
	url    *url.URL
	app    string
	stream string
	tid    uint32         // transactionID
	nc     *NetConnection // rtmp net-connection
}

// NewClient .
func NewClient(remote string) (*Client, error) {
	url, err := url.Parse(remote)
	if err != nil {
		return nil, fmt.Errorf("RTMP: parse rtmp url error, %v", err)
	}
	urls := strings.SplitN(strings.TrimLeft(url.Path, "/"), "/", 2)
	if len(urls) != 2 {
		return nil, fmt.Errorf("RTMP: parse app & stream error, %v", err)
	}
	client := &Client{
		url:    url,
		app:    urls[0],
		stream: urls[1],
		tid:    0,
		nc:     nil,
	}
	return client, nil
}

// Handshake .
func (client *Client) Handshake() error {
	var err error
	var goConn net.Conn

	switch strings.ToLower(client.url.Scheme) {
	case "rtmp":
		goConn, err = net.Dial("tcp", client.url.Host)
	default:
		return fmt.Errorf("RTMP: client not support protocol: %s", client.url.Scheme)
	}
	if err != nil {
		return err
	}

	client.nc = NewNetConn(nil, goConn)
	if err := client.nc.ComplexClientHandshake(); err != nil {
		return err
	}
	client.nc.Serve()

	return nil
}

// Connect .
func (client *Client) Connect() error {
	// 1. set chunk size
	if err := client.nc.SetChunkSize(4096); err != nil {
		return err
	}

	// 2. command connect()
	argument := make(map[string]interface{})
	argument["app"] = "live"
	argument["type"] = "nonprivate"
	argument["flashVer"] = "FMLE/3.0 (compatible; FMSc/1.0)"
	argument["swfUrl"] = client.url.Scheme + "://" + client.url.Host + "/" + client.app
	argument["tcUrl"] = client.url.Scheme + "://" + client.url.Host + "/" + client.app

	connect := &Command{
		Name:          "connect",
		TransactionID: atomic.AddUint32(&client.tid, 1),
		Objects:       []interface{}{},
		UserArguments: []interface{}{argument},
	}
	if err := client.nc.WriteCommand(SIDNetConnnection, connect); err != nil {
		return err
	}

	// 3. connect response, onResult(), see code in conn_command.go
	return nil
}

// Publish .
func (client *Client) Publish() error {
	// 1. release
	release := &Command{
		Name:          "releaseStream",
		TransactionID: atomic.AddUint32(&client.tid, 1),
		Objects:       []interface{}{nil},
		UserArguments: []interface{}{client.stream},
	}
	if err := client.nc.WriteCommand(SIDNetConnnection, release); err != nil {
		return err
	}

	// 2. FC publish
	fcPublish := &Command{
		Name:          "FCPublish",
		TransactionID: atomic.AddUint32(&client.tid, 1),
		Objects:       []interface{}{nil},
		UserArguments: []interface{}{client.stream},
	}
	if err := client.nc.WriteCommand(SIDNetConnnection, fcPublish); err != nil {
		return err
	}

	// 3. create stream
	createStream := &Command{
		Name:          "createStream",
		TransactionID: atomic.AddUint32(&client.tid, 1),
		Objects:       []interface{}{nil},
		UserArguments: []interface{}{nil},
	}
	if err := client.nc.WriteCommand(SIDNetConnnection, createStream); err != nil {
		return err
	}

	// 4. publish
	publish := &Command{
		Name:          "publish",
		TransactionID: atomic.AddUint32(&client.tid, 1),
		Objects:       []interface{}{nil},
		UserArguments: []interface{}{client.stream, "live"},
	}
	if err := client.nc.WriteCommand(SIDNetStream, publish); err != nil {
		return err
	}

	// 5. @setDataFrame & onMetadata

	return nil
}

/************************************/
/******** Subscribe Interface *******/
/************************************/

// WriteAVPacket .
func (client *Client) WriteAVPacket(packet *avformat.AVPacket) error {
	message := &Message{
		TypeID:    packet.TypeID,
		Length:    packet.Length,
		Timestamp: packet.Timestamp,
		StreamID:  packet.StreamID,
		Body:      bytes.NewBuffer(packet.Body),
	}
	return client.nc.AsyncWrite(message)
}

// Close .
func (client *Client) Close() error {
	client.nc.Close()
	return nil
}
