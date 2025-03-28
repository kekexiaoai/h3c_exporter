package gnmi

import (
	"encoding/json"
	"log"
	"math/rand"
	"time"

	"github.com/openconfig/gnmi/proto/gnmi"

	"github.com/kekexiaoai/h3c_exporter/config"
	"github.com/kekexiaoai/h3c_exporter/metrics"
	"github.com/kekexiaoai/h3c_exporter/model"
	"github.com/kekexiaoai/h3c_exporter/pkg/sdk"
)

type Client struct{}

func NewClient() *Client {
	return &Client{}
}

type grpcCon struct {
	gnmiClient  gnmi.GNMIClient
	grpcSession *sdk.GrpcSession
}

func connect(sw config.ResolvedSwitch) (*grpcCon, error) {
	grpcSession, err := sdk.NewClient(sw.Address, sw.Port, sw.Username, sw.Password)
	if err != nil {
		metrics.IncError(sw.Address, "connect")
		metrics.UpdateConnectStatus(sw.Address, false)
		log.Printf("Failed to connect to %s: %v", sw.Address, err)
		return nil, err
	}
	gnmiClient := gnmi.NewGNMIClient(grpcSession.Conn)
	metrics.UpdateConnectStatus(sw.Address, true)
	return &grpcCon{gnmiClient: gnmiClient, grpcSession: grpcSession}, nil
}

// Subscribe 订阅指定交换机的网口状态
func (c *Client) Subscribe(sw config.ResolvedSwitch) {
	device := sw.Address
	// 指数退避参数
	baseDelay := 5 * time.Second
	maxDelay := 1 * time.Minute

	for {
		// 使用 defer 捕获可能的 panic
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Recovered from panic in Subscribe for %s: %v", device, r)
					metrics.IncError(device, "panic")
					metrics.UpdateSubscribeStatus(device, false)
				}
			}()

			con, err := connect(sw)
			if err != nil {
				time.Sleep(jitterDelay(baseDelay, maxDelay))
				return
			}
			defer con.grpcSession.Close() // 确保连接关闭

			subReq := &gnmi.SubscribeRequest{
				Request: &gnmi.SubscribeRequest_Subscribe{
					Subscribe: &gnmi.SubscriptionList{
						Subscription: []*gnmi.Subscription{
							{
								Path: &gnmi.Path{
									Elem: []*gnmi.PathElem{
										{Name: "interfaces"},
										{Name: "interface"},
										{Name: "state"},
									},
								},
								Mode:           gnmi.SubscriptionMode_SAMPLE,
								SampleInterval: uint64(time.Duration(sw.SampleInterval) * time.Second),
							},
						},
						Mode:     gnmi.SubscriptionList_STREAM,
						Encoding: gnmi.Encoding_JSON,
					},
				},
			}

			ctx, cancel := sdk.CtxWithToken(con.grpcSession.Token, time.Hour)
			defer cancel() // 确保上下文取消

			stream, err := con.gnmiClient.Subscribe(ctx)
			if err != nil {
				log.Printf("Subscribe to %s failed: %v", device, err)
				metrics.IncError(device, "subscribe")
				metrics.UpdateSubscribeStatus(device, false)
				time.Sleep(jitterDelay(baseDelay, maxDelay))
				return
			}

			if err := stream.Send(subReq); err != nil {
				log.Printf("Send subscribe request to %s failed: %v", device, err)
				metrics.IncError(device, "subscribe")
				metrics.UpdateSubscribeStatus(device, false)
				time.Sleep(jitterDelay(baseDelay, maxDelay))
				return
			}

			log.Printf("Subscribed to %s successfully, sample interval %d s", device, sw.SampleInterval)
			metrics.UpdateSubscribeStatus(device, true)
			for {
				resp, err := stream.Recv()
				if err != nil {
					log.Printf("Recv from %s failed: %v", device, err)
					metrics.IncError(device, "subscribe")
					metrics.UpdateSubscribeStatus(device, false)
					break
				}
				processResponse(device, resp)
			}
			time.Sleep(jitterDelay(baseDelay, maxDelay))
		}()
	}
}

// jitterDelay 计算带有随机抖动的退避时间
func jitterDelay(base, max time.Duration) time.Duration {
	delay := base
	// 添加最多 50% 的随机抖动
	jitter := time.Duration(rand.Int63n(int64(base) / 2))
	delay += jitter
	if delay > max {
		delay = max
	}
	log.Printf("Jittered delay for %s: %v", base, delay)
	return delay
}

func processResponse(device string, resp *gnmi.SubscribeResponse) {
	switch r := resp.Response.(type) {
	case *gnmi.SubscribeResponse_Update:
		update := r.Update
		for _, u := range update.Update {
			val := u.Val
			if val != nil && val.GetJsonVal() != nil {
				jsonData := val.GetJsonVal()

				var intfState model.InterfaceState
				if err := json.Unmarshal(jsonData, &intfState); err != nil {
					log.Printf("JSON parse error for %s: %v", device, err)
					metrics.IncError(device, "parse")
					continue
				}
				metrics.Update(device, intfState)
			}
		}
	case *gnmi.SubscribeResponse_SyncResponse:
		log.Printf("Sync completed for %s: %v", device, r.SyncResponse)
	case *gnmi.SubscribeResponse_Error:
		log.Printf("Error received from %s: %v", device, r.Error)
		metrics.IncError(device, "subscribe")
	}
}
