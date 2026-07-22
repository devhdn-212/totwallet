package dto

type SettingData struct {
	ID               int    `json:"setting_id"`
	Appversion       string `json:"setting_appversion"`
	Startmaintenance string `json:"setting_startmaintenance"`
	Endmaintenance   string `json:"setting_endmaintenance"`
	Shio_parent      int    `json:"setting_shio_parent"`
	Created          string `json:"setting_created"`
	Update           string `json:"setting_updated"`
}

type SettingSave struct {
	Type              string `json:"type" validate:"required"`
	ID                int    `json:"setting_id" `
	Appversion        string `json:"setting_appversion" validate:"required"`
	Start_maintenance string `json:"setting_startmaintenance" validate:"required"`
	End_maintenance   string `json:"setting_endmaintenance" validate:"required"`
	Shioparent        int    `json:"setting_shio_parent" `
}
