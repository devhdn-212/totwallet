package dto

type HealthCheckData struct {
	RealIP      string   `json:"real_ip"`
	ContainerIP string   `json:"container_ip"`
	IPList      []string `json:"ip_list"`
}
