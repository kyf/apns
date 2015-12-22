package apns

import (
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/rand"
	"net"
	"strconv"
	"strings"

	"time"
)

const pushCommandValue = 2

const MaxPayloadSizeBytes = 2048

const IdentifierUbound = 9999

const TimeoutSeconds = 2

const (
	deviceTokenItemid            = 1
	payloadItemid                = 2
	notificationIdentifierItemid = 3
	expirationDateItemid         = 4
	priorityItemid               = 5
	deviceTokenLength            = 32
	notificationIdentifierLength = 4
	expirationDateLength         = 4
	priorityLength               = 1
)

type Payload struct {
	Alert            interface{} `json:"alert,omitempty"`
	Badge            int         `json:"badge,omitempty"`
	Sound            string      `json:"sound,omitempty"`
	ContentAvailable int         `json:"content-available,omitempty"`
	Category         string      `json:"category,omitempty"`
}

func NewPayload() *Payload {
	return new(Payload)
}

type PushNotification struct {
	Identifier  int32
	Expiry      uint32
	DeviceToken string
	payload     map[string]interface{}
	Priority    uint8
}

func NewPushNotification() (pn *PushNotification) {
	pn = new(PushNotification)
	pn.payload = make(map[string]interface{})
	pn.Identifier = rand.New(rand.NewSource(time.Now().UnixNano())).Int31n(IdentifierUbound)
	pn.Priority = 10
	return
}

func (pn *PushNotification) AddPayload(p *Payload) {
	if p.Badge == 0 {
		p.Badge = -1
	}
	pn.Set("aps", p)
}

func (pn *PushNotification) Get(key string) interface{} {
	return pn.payload[key]
}

func (pn *PushNotification) Set(key string, value interface{}) {
	pn.payload[key] = value
}

func (pn *PushNotification) PayloadJSON() ([]byte, error) {
	return json.Marshal(pn.payload)
}

func (pn *PushNotification) PayloadString() (string, error) {
	j, err := pn.PayloadJSON()
	return string(j), err
}

func (pn *PushNotification) ToBytes() ([]byte, error) {
	token, err := hex.DecodeString(pn.DeviceToken)
	if err != nil {
		return nil, err
	}
	if len(token) != deviceTokenLength {
		return nil, errors.New("device token has incorrect length")
	}
	payload, err := pn.PayloadJSON()
	if err != nil {
		return nil, err
	}
	if len(payload) > MaxPayloadSizeBytes {
		return nil, errors.New("payload is larger than the " + strconv.Itoa(MaxPayloadSizeBytes) + " byte limit")
	}

	frameBuffer := new(bytes.Buffer)
	binary.Write(frameBuffer, binary.BigEndian, uint8(deviceTokenItemid))
	binary.Write(frameBuffer, binary.BigEndian, uint16(deviceTokenLength))
	binary.Write(frameBuffer, binary.BigEndian, token)
	binary.Write(frameBuffer, binary.BigEndian, uint8(payloadItemid))
	binary.Write(frameBuffer, binary.BigEndian, uint16(len(payload)))
	binary.Write(frameBuffer, binary.BigEndian, payload)
	binary.Write(frameBuffer, binary.BigEndian, uint8(notificationIdentifierItemid))
	binary.Write(frameBuffer, binary.BigEndian, uint16(notificationIdentifierLength))
	binary.Write(frameBuffer, binary.BigEndian, pn.Identifier)
	binary.Write(frameBuffer, binary.BigEndian, uint8(expirationDateItemid))
	binary.Write(frameBuffer, binary.BigEndian, uint16(expirationDateLength))
	binary.Write(frameBuffer, binary.BigEndian, pn.Expiry)
	binary.Write(frameBuffer, binary.BigEndian, uint8(priorityItemid))
	binary.Write(frameBuffer, binary.BigEndian, uint16(priorityLength))
	binary.Write(frameBuffer, binary.BigEndian, pn.Priority)

	buffer := bytes.NewBuffer([]byte{})
	binary.Write(buffer, binary.BigEndian, uint8(pushCommandValue))
	binary.Write(buffer, binary.BigEndian, uint32(frameBuffer.Len()))
	binary.Write(buffer, binary.BigEndian, frameBuffer.Bytes())
	return buffer.Bytes(), nil
}

var ApplePushResponses = map[uint8]string{
	0:   "NO_ERRORS",
	1:   "PROCESSING_ERROR",
	2:   "MISSING_DEVICE_TOKEN",
	3:   "MISSING_TOPIC",
	4:   "MISSING_PAYLOAD",
	5:   "INVALID_TOKEN_SIZE",
	6:   "INVALID_TOPIC_SIZE",
	7:   "INVALID_PAYLOAD_SIZE",
	8:   "INVALID_TOKEN",
	10:  "SHUTDOWN",
	255: "UNKNOWN",
}

type PushNotificationResponse struct {
	Success       bool
	AppleResponse string
	Error         error
}

func NewPushNotificationResponse() (resp *PushNotificationResponse) {
	resp = new(PushNotificationResponse)
	resp.Success = false
	return
}

var _ APNSClient = &Client{}

type APNSClient interface {
	Connect() error
	Send(pn *PushNotification) (resp *PushNotificationResponse)
	Close()
}

type Client struct {
	Gateway           string
	CertificateFile   string
	CertificateBase64 string
	KeyFile           string
	KeyBase64         string
	tlsConn           *tls.Conn
	conn              net.Conn
}

func NewClient(gateway, certificateFile, keyFile string) (c *Client) {
	c = new(Client)
	c.Gateway = gateway
	c.CertificateFile = certificateFile
	c.KeyFile = keyFile
	return
}

func (client *Client) Connect() error {
	var cert tls.Certificate
	var err error

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
	client.conn = conn
	tlsConn := tls.Client(conn, conf)
	err = tlsConn.Handshake()
	if err != nil {
		return err
	}
	client.tlsConn = tlsConn

	return nil

}

func (client *Client) Send(pn *PushNotification) (resp *PushNotificationResponse) {
	resp = new(PushNotificationResponse)

	payload, err := pn.ToBytes()
	if err != nil {
		resp.Success = false
		resp.Error = err
		return
	}

	_, err = client.tlsConn.Write(payload)
	if err != nil {
		resp.Success = false
		resp.Error = err
		return
	}

	timeoutChannel := make(chan bool, 1)
	go func() {
		time.Sleep(time.Second * TimeoutSeconds)
		timeoutChannel <- true
	}()

	responseChannel := make(chan []byte, 1)
	go func() {
		buffer := make([]byte, 6, 6)
		client.tlsConn.Read(buffer)
		responseChannel <- buffer
	}()

	select {
	case r := <-responseChannel:
		resp.Success = false
		resp.AppleResponse = ApplePushResponses[r[1]]
		err = errors.New(resp.AppleResponse)
	case <-timeoutChannel:
		resp.Success = true
	}

	return
}

func (client *Client) Close() {
	client.tlsConn.Close()
	client.conn.Close()
}
