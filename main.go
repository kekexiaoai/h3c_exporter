package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/kekexiaoai/h3c_exporter/config"
	"github.com/kekexiaoai/h3c_exporter/metrics"
	"github.com/kekexiaoai/h3c_exporter/pkg/gnmi"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	flag.Parse()

	// 加载配置
	cfg, switches, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 收集 default 中定义的标签名
	allLabels := make([]string, 0, len(cfg.Default.Labels))
	for k := range cfg.Default.Labels {
		allLabels = append(allLabels, k)
	}

	// 初始化指标
	metrics.Init(allLabels)

	// 设置设备标签
	metrics.SetDeviceLabels(switches)

	// 提取设备地址列表
	devices := make([]string, len(switches))
	for i, sw := range switches {
		devices[i] = sw.Address
	}

	// 启动 gNMI 订阅
	client := gnmi.NewClient()
	for _, sw := range switches {
		delay := time.Duration(rand.Intn(5000)) * time.Millisecond // 0-5秒随机延迟
		go func(sw config.ResolvedSwitch, d time.Duration) {
			time.Sleep(d)
			client.Subscribe(sw)
		}(sw, delay)
	}

	// 注册 HTTP 路由
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		statusHandler(w, r, devices)
	})

	// 启动服务
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("Starting server on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

// statusHandler 处理 /status 请求
func statusHandler(w http.ResponseWriter, r *http.Request, devices []string) {
	status := metrics.GetSwitchStatus(devices)

	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>H3C Exporter Status</title>
    <style>
        table { border-collapse: collapse; width: 80%; margin: 20px auto; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
        tr:nth-child(even) { background-color: #f9f9f9; }
    </style>
</head>
<body>
    <h1>H3C Exporter Switch Status</h1>
    <table>
        <tr>
            <th>Device</th>
            <th>Connected</th>
            <th>Subscribed</th>
            <th>Errors (connect/subscribe/parse/panic)</th>
        </tr>
        {{range $device, $data := .}}
        <tr>
            <td>{{$device}}</td>
            <td>{{if eq $data.connected 1.0}}Yes{{else}}No{{end}}</td>
            <td>{{if eq $data.subscribed 1.0}}Yes{{else}}No{{end}}</td>
            <td>{{printf "%.0f/%.0f/%.0f/%.0f" $data.errors.connect $data.errors.subscribe $data.errors.parse $data.errors.panic}}</td>
        </tr>
        {{end}}
    </table>
</body>
</html>
`
	t, err := template.New("status").Parse(tmpl)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("Template parse error: %v", err)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	if err := t.Execute(w, status); err != nil {
		log.Printf("Template execute error: %v", err)
	}
}
