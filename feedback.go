package apns

import (
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"net"
	"strings"
	"time"
)

const FeedbackTimeoutSeconds = 5

var FeedbackChannel = make(chan (*FeedbackResponse))

var ShutdownChannel = make(chan bool)

type FeedbackResponse struct {
	Timestamp   uint32
	DeviceToken string
}

func NewFeedbackResponse() (resp *FeedbackResponse) {
	resp = new(FeedbackResponse)
	return
}

func (client *Client) ListenForFeedback() (err error) {
	var cert tls.Certificate

	if len(client.CertificateBase64) == 0 && len(client.KeyBase64) == 0 {
		cert, err = tls.LoadX509KeyPair(client.CertificateFile, client.KeyFile)
	} else {
		cert, err = tls.X509KeyPair([]byte(client.CertificateBase64), []byte(client.KeyBase64))
	}

	if err != nil {
		return err
	}

	gatewayParts := strings.Split(client.Gateway, ":")
	conf := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ServerName:   gatewayParts[0],
	}

	conn, err := net.Dial("tcp", client.Gateway)
	if err != nil {
		return err
	}
	defer conn.Close()
	conn.SetReadDeadline(time.Now().Add(FeedbackTimeoutSeconds * time.Second))

	tlsConn := tls.Client(conn, conf)
	err = tlsConn.Handshake()
	if err != nil {
		return err
	}

	var tokenLength uint16
	buffer := make([]byte, 38, 38)
	deviceToken := make([]byte, 32, 32)

	for {
		_, err := tlsConn.Read(buffer)
		if err != nil {
			ShutdownChannel <- true
			break
		}

		resp := NewFeedbackResponse()

		r := bytes.NewReader(buffer)
		binary.Read(r, binary.BigEndian, &resp.Timestamp)
		binary.Read(r, binary.BigEndian, &tokenLength)
		binary.Read(r, binary.BigEndian, &deviceToken)
		if tokenLength != 32 {
			return errors.New("token length should be equal to 32, but isn't")
		}
		resp.DeviceToken = hex.EncodeToString(deviceToken)

		FeedbackChannel <- resp
	}

	return nil
}
