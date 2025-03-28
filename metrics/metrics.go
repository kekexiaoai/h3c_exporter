package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	client_model "github.com/prometheus/client_model/go"

	"github.com/kekexiaoai/h3c_exporter/config"
	"github.com/kekexiaoai/h3c_exporter/model"
)

type Metrics struct {
	AdminStatus     *prometheus.GaugeVec
	OperStatus      *prometheus.GaugeVec
	ConnectStatus   *prometheus.GaugeVec
	SubscribeStatus *prometheus.GaugeVec
	ErrorCount      *prometheus.CounterVec
	states          map[string]map[string]model.InterfaceState
	mu              sync.Mutex
	deviceLabels    map[string]map[string]string
	extraLabelNames []string
	updates         chan updateTask
}

type updateTask struct {
	device string
	state  model.InterfaceState
}

var appMetrics *Metrics

func Init(extraLabelNames []string) {
	appMetrics = &Metrics{
		AdminStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "interface_admin_status",
				Help: "Administrative status of the interface (1=UP, 0=DOWN)",
			},
			append([]string{"device", "interface"}, extraLabelNames...),
		),
		OperStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "interface_oper_status",
				Help: "Operational status of the interface (1=UP, 0=DOWN)",
			},
			append([]string{"device", "interface"}, extraLabelNames...),
		),
		ConnectStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "gnmi_connect_status",
				Help: "Connection status to the switch (1=connected, 0=disconnected)",
			},
			append([]string{"device"}, extraLabelNames...),
		),
		SubscribeStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "gnmi_subscribe_status",
				Help: "Subscription status to the switch (1=active, 0=inactive)",
			},
			append([]string{"device"}, extraLabelNames...),
		),
		ErrorCount: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gnmi_error_total",
				Help: "Total number of gNMI errors",
			},
			append([]string{"device", "type"}, extraLabelNames...),
		),
		states:          make(map[string]map[string]model.InterfaceState),
		deviceLabels:    make(map[string]map[string]string),
		extraLabelNames: extraLabelNames,
		updates:         make(chan updateTask, 1000), // 缓冲区大小
	}
	go appMetrics.processUpdates() // 启动后台处理 goroutine
	prometheus.MustRegister(appMetrics.AdminStatus)
	prometheus.MustRegister(appMetrics.OperStatus)
	prometheus.MustRegister(appMetrics.ConnectStatus)
	prometheus.MustRegister(appMetrics.SubscribeStatus)
	prometheus.MustRegister(appMetrics.ErrorCount)
}

func SetDeviceLabels(switches []config.ResolvedSwitch) {
	appMetrics.mu.Lock()
	defer appMetrics.mu.Unlock()

	for _, sw := range switches {
		appMetrics.deviceLabels[sw.Address] = sw.Labels
	}
}

func Update(device string, state model.InterfaceState) {
	appMetrics.updates <- updateTask{device, state} // 发送到 channel
}

func (m *Metrics) processUpdates() {
	for task := range m.updates {
		m.mu.Lock()
		// 更新状态
		if _, exists := m.states[task.device]; !exists {
			m.states[task.device] = make(map[string]model.InterfaceState)
		}
		m.states[task.device][task.state.Name] = task.state

		// 计算指标值
		adminStatus := 0.0
		if task.state.AdminStatus == "UP" {
			adminStatus = 1.0
		}
		operStatus := 0.0
		if task.state.OperStatus == "UP" {
			operStatus = 1.0
		}

		// 构造完整标签
		labels := map[string]string{
			"device":    task.device,
			"interface": task.state.Name,
		}
		deviceLabels := m.deviceLabels[task.device]
		if deviceLabels == nil {
			deviceLabels = make(map[string]string)
		}
		for _, labelName := range m.extraLabelNames {
			if value, exists := deviceLabels[labelName]; exists {
				labels[labelName] = value
			} else {
				labels[labelName] = ""
			}
		}

		// 设置指标
		m.AdminStatus.With(labels).Set(adminStatus)
		m.OperStatus.With(labels).Set(operStatus)
		m.mu.Unlock()
	}
}

func UpdateConnectStatus(device string, connected bool) {
	appMetrics.mu.Lock()
	defer appMetrics.mu.Unlock()

	status := 0.0
	if connected {
		status = 1.0
	}

	labels := map[string]string{"device": device}
	deviceLabels := appMetrics.deviceLabels[device]
	if deviceLabels == nil {
		deviceLabels = make(map[string]string)
	}
	for _, labelName := range appMetrics.extraLabelNames {
		if value, exists := deviceLabels[labelName]; exists {
			labels[labelName] = value
		} else {
			labels[labelName] = ""
		}
	}
	appMetrics.ConnectStatus.With(labels).Set(status)
}

func UpdateSubscribeStatus(device string, active bool) {
	appMetrics.mu.Lock()
	defer appMetrics.mu.Unlock()

	status := 0.0
	if active {
		status = 1.0
	}

	labels := map[string]string{"device": device}
	deviceLabels := appMetrics.deviceLabels[device]
	if deviceLabels == nil {
		deviceLabels = make(map[string]string)
	}
	for _, labelName := range appMetrics.extraLabelNames {
		if value, exists := deviceLabels[labelName]; exists {
			labels[labelName] = value
		} else {
			labels[labelName] = ""
		}
	}
	appMetrics.SubscribeStatus.With(labels).Set(status)
}

func IncError(device, errorType string) {
	appMetrics.mu.Lock()
	defer appMetrics.mu.Unlock()

	labels := map[string]string{
		"device": device,
		"type":   errorType,
	}
	deviceLabels := appMetrics.deviceLabels[device]
	if deviceLabels == nil {
		deviceLabels = make(map[string]string)
	}
	for _, labelName := range appMetrics.extraLabelNames {
		if value, exists := deviceLabels[labelName]; exists {
			labels[labelName] = value
		} else {
			labels[labelName] = ""
		}
	}
	appMetrics.ErrorCount.With(labels).Inc()
}

func GetSwitchStatus(devices []string) map[string]map[string]interface{} {
	appMetrics.mu.Lock()
	defer appMetrics.mu.Unlock()

	status := make(map[string]map[string]interface{})
	for _, device := range devices {
		data := make(map[string]interface{})
		deviceLabels := appMetrics.deviceLabels[device]
		if deviceLabels == nil {
			deviceLabels = make(map[string]string)
		}

		baseLabels := map[string]string{"device": device}
		for _, labelName := range appMetrics.extraLabelNames {
			if value, exists := deviceLabels[labelName]; exists {
				baseLabels[labelName] = value
			} else {
				baseLabels[labelName] = ""
			}
		}

		if m, err := appMetrics.ConnectStatus.GetMetricWith(baseLabels); err == nil {
			data["connected"] = getGaugeValue(m)
		} else {
			data["connected"] = 0.0
		}
		if m, err := appMetrics.SubscribeStatus.GetMetricWith(baseLabels); err == nil {
			data["subscribed"] = getGaugeValue(m)
		} else {
			data["subscribed"] = 0.0
		}
		data["errors"] = make(map[string]float64)
		for _, errType := range []string{"connect", "subscribe", "parse", "panic"} {
			errLabels := map[string]string{"device": device, "type": errType}
			for _, labelName := range appMetrics.extraLabelNames {
				if value, exists := deviceLabels[labelName]; exists {
					errLabels[labelName] = value
				} else {
					errLabels[labelName] = ""
				}
			}
			if m, err := appMetrics.ErrorCount.GetMetricWith(errLabels); err == nil {
				data["errors"].(map[string]float64)[errType] = getCounterValue(m)
			} else {
				data["errors"].(map[string]float64)[errType] = 0.0
			}
		}
		status[device] = data
	}
	return status
}

func getGaugeValue(m prometheus.Metric) float64 {
	var metric client_model.Metric
	m.Write(&metric)
	return *metric.Gauge.Value
}

func getCounterValue(m prometheus.Metric) float64 {
	var metric client_model.Metric
	m.Write(&metric)
	return *metric.Counter.Value
}
