package model

// InterfaceState 表示网口状态
type InterfaceState struct {
	Name        string `json:"name"`
	AdminStatus string `json:"admin-status"`
	OperStatus  string `json:"oper-status"`
}
