package apns

import (
	"fmt"
	"testing"
	//"time"
)

const (
	TIGER_TOKEN string = "18a711ed1fdf069bd34dc55933310f3b807eea092260c55cc9b6657a18befc56"
	CERT_PATH   string = "/home/kyf/cert/cert.pem"
	GATEWAY     string = "gateway.push.apple.com:2195"
)

func InitPushNotification(alert string) *PushNotification {
	payload := NewPayload()
	payload.Alert = alert
	payload.Badge = 1
	payload.Sound = "bingbong.aiff"

	pn := NewPushNotification()

	pn.DeviceToken = TIGER_TOKEN
	pn.Set("type", "1")
	pn.AddPayload(payload)
	return pn
}

func TestPushSingleDevice(t *testing.T) {
	client := NewClient(GATEWAY, CERT_PATH, CERT_PATH)
	err := client.Connect()
	if err != nil {
		t.Errorf("push failure:%v\n", err)
	} else {
		defer client.Close()
		num := 20
		count := make(chan int, num)
		for i := 1; i <= num; i++ {
			pn := InitPushNotification(fmt.Sprintf("6人游消息推送_%d", i))
			go func(pn *PushNotification) {
				res := client.Send(pn)
				if !res.Success {
					t.Errorf("push failure:%v\n", res.Error)
				} else {
					fmt.Println("push success")
				}
				count <- 1
			}(pn)
		}
		//ticker := time.NewTicker(time.Second * 1)
		for {
			//select {
			//case <-ticker.C:
			fmt.Println("len is ", len(count), ", cap is ", cap(count))
			if len(count) == num {
				break
			}
			//}
		}
	}
}

func TestPushAllDevices(t *testing.T) {

}
