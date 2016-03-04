package apns

import (
	"fmt"
	"testing"
	"time"
)

const (
	TIGER_TOKEN string = "18a711ed1fdf069bd34dc55933310f3b807eea092260c55cc9b6657a18befc56"
	CERT_PATH   string = "/home/kyf/cert/cert.pem"
	GATEWAY     string = "gateway.push.apple.com:2195"
)

var (
	//tokens []string = []string{"42757435a4d8cd502f8f4e4c582125644514292f2b68a8dc842cf1642b05fa90", "832afd0e6ccca288a77d73c0c8a8914b02e0c4d6abf8af9badafbd78ba5b0cac"}
	tokens []string = []string{"18a711ed1fdf069bd34dc55933310f3b807eea092260c55cc9b6657a18befc56", "42757435a4d8cd502f8f4e4c582125644514292f2b68a8dc842cf1642b05fa90", "832afd0e6ccca288a77d73c0c8a8914b02e0c4d6abf8af9badafbd78ba5b0cac"}
)

func InitPushNotification(alert string, index int32, token string) *PushNotification {
	payload := NewPayload()
	payload.Alert = alert
	payload.Badge = 1
	payload.Sound = "bingbong.aiff"

	pn := NewPushNotification(index)

	pn.DeviceToken = token
	pn.Set("type", "1")
	pn.AddPayload(payload)
	return pn
}

func getDeviceTokens() []string {
	result := make([]string, 0)
	for i := 0; i <= 1000; i++ {
		result = append(result, tokens[int(index)%len(tokens)])
	}
	return result
}

func TestPushSingleDevice(t *testing.T) {

}

func Execute() {
	client := NewClient(GATEWAY, CERT_PATH, CERT_PATH)
	err := client.Connect()
	if err != nil {
		t.Errorf("push failure:%v\n", err)
	} else {
		defer client.Close()
		num := 260
		start := 250
		ErrChan := make(chan error)
		go func() {
			for {
				select {
				case e := <-client.ErrorChan:
					ErrChan <- e
				}
			}

		}()
		time.Sleep(time.Second * 1)
	MyLoop:
		for ; start <= num; start++ {
			pn := InitPushNotification(fmt.Sprintf("6人游消息推送_%d", start), int32(start))
			res := client.Send(pn)
			fmt.Println("res is ", res, " start is ", start)
		}

		ticker := time.NewTicker(time.Second * 20)
		for {
			select {
			case <-ticker.C:
				fmt.Println("exit >>>>>")
				return
			case e := <-ErrChan:
				fmt.Printf("return error:%v, lastIdentifier is %v\n", e, client.LastIdentifier)
				client.Close()
				client.Connect()
				start = int(client.LastIdentifier + 1)
				goto MyLoop
			}
		}
	}
}

func TestPushAllDevices(t *testing.T) {

}
