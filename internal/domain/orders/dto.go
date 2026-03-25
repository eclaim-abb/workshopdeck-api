package orders

import "time"

type AddClientRequest struct {
	ClientName          string `json:"client_name" binding:"required"`
	ClientEmail         string `json:"client_email"`
	ClientPhone         string `json:"client_phone" binding:"required"`
	CityNo              uint   `json:"city_no" binding:"required"`
	CityType            string `json:"city_type" binding:"required"`
	CityName            string `json:"city_name" binding:"required"`
	Address             string `json:"address" binding:"required"`
	VehicleBrandName    string `json:"vehicle_brand_name" binding:"required"`
	VehicleSeriesName   string `json:"vehicle_series_name" binding:"required"`
	VehicleChassisNo    string `json:"vehicle_chassis_no" binding:"required"`
	VehicleLicensePlate string `json:"vehicle_license_plate" binding:"required"`
	VehiclePrice        uint   `json:"vehicle_price" binding:"required"`
}

type CreateOrderRequest struct {
	WorkshopNo    uint              `json:"workshop_no" binding:"required"`
	InsuranceNo   uint              `json:"insurance_no"`
	ClientNo      uint              `json:"client_no"`
	ClientDetails *AddClientRequest `json:"client_details"`
	ClaimDetails  string            `json:"claim_details" binding:"required"`
	ETA           time.Time         `json:"eta"`
	Status        string            `json:"status" binding:"required"`
	CreatedBy     uint              `json:"created_by" binding:"required"`
}

type AddOrderPanelRepairStatus struct {
	OrderPanelNo uint                  `json:"order_panel_no" binding:"required"`
	Status       string                `json:"status" binding:"required"`
	RepairStatus string                `json:"repair_status" binding:"required"`
	Notes        string                `json:"notes" binding:"required"`
	CreatedBy    uint                  `json:"created_by" binding:"required"`
	RepairPhotos []RepairPhotoMetadata `json:"repair_photos"`
}

type OrderPanelRequest struct {
	OrderPanelNo            uint   `json:"order_panel_no" binding:"required"`
	WOGroupNumber           uint   `json:"wo_group_number"`
	InsurancePanelPricingNo uint   `json:"insurance_panel_pricing_no"`
	InsurancePanelName      string `json:"insurance_panel_name"`
	InsurerMeasurementNo    uint   `json:"insurer_measurement_no"`
	InsurerPrice            uint   `json:"insurer_price"`
	InsurerQty              uint   `json:"insurer_qty"`
	InsurerServiceType      string `json:"insurer_service_type"`
	WorkshopPanelPricingNo  uint   `json:"workshop_panel_pricing_no"`
	WorkshopPanelName       string `json:"workshop_panel_name"`
	WorkshopPrice           uint   `json:"workshop_price"`
	WorkshopMeasurementNo   uint   `json:"workshop_measurement_no"`
	WorkshopQty             uint   `json:"workshop_qty"`
	WorkshopServiceType     string `json:"workshop_service_type"`
	IsIncluded              bool   `json:"is_included"`
	IsSpecialRepair         bool   `json:"is_special_repair"`
}

type CreateWorkOrderRequest struct {
	OrderNo                  uint                `json:"order_no" binding:"required"`
	AdditionalWorkOrderCount uint                `json:"add_wo_count"`
	WorkOrderDocumentNumber  string              `json:"wo_doc_number"`
	WorkOrderUrl             string              `json:"wo_url"`
	OrderPanels              []OrderPanelRequest `json:"order_panels"`

	CreatedBy uint `json:"created_by"`
}

type ProposeAdditionalWorkRequest struct {
	WorkOrderNo    uint                          `json:"work_order_no" binding:"required"`
	OrderPanels    []OrderPanelRequest           `json:"order_panels"`
	Photos         []AdditionalWorkPhotoMetadata `json:"photos"`
	LastModifiedBy uint                          `json:"last_modified_by"`
}

type AcceptDeclineOrder struct {
	ETA            time.Time `json:"eta"`
	DiscountType   string    `json:"discount_type"`
	Discount       float64   `json:"discount"`
	LastModifiedBy uint      `json:"last_modified_by" binding:"required"`
}

type NegotiationPhotoMetadata struct {
	OrderPanelNo uint `json:"order_panel_no" binding:"required"`
	FileIndex    int  `json:"file_index" binding:"required"`
}

type RepairPhotoMetadata struct {
	FileIndex    int    `json:"file_index" binding:"required"`
	PhotoCaption string `json:"photo_caption"`
	PhotoType    string `json:"photo_type" binding:"required"`
}

type AdditionalWorkPhotoMetadata struct {
	WorkshopPanelPricingNo uint   `json:"workshop_panel_pricing_no"`
	FileIndex              int    `json:"file_index" binding:"required"`
	PhotoCaption           string `json:"photo_caption"`
}

type SubmitNegotiationRequest struct {
	WorkOrderNo    uint                       `json:"work_order_no" binding:"required"`
	OrderPanels    []OrderPanelRequest        `json:"order_panels" binding:"required,dive"`
	Reason         string                     `json:"reason" binding:"required"`
	Photos         []NegotiationPhotoMetadata `json:"photos"`
	LastModifiedBy uint                       `json:"last_modified_by" binding:"required"`
}

type NegotiationCreatedInfo struct {
	OrderPanelNo         uint `json:"order_panel_no"`
	NegotiationHistoryNo uint `json:"negotiation_history_no"`
	RoundCount           uint `json:"round_count"`
	PhotosUploaded       int  `json:"photos_uploaded"`
}

type CancelNegotiationRequest struct {
	OrderNo        uint `json:"order_no"`
	LastModifiedBy uint `json:"last_modified_by"`
}

type ApproveAdditionalProposalRequest struct {
	CancelNegotiationRequest
	ETA          time.Time `json:"eta"`
	DiscountType string    `json:"discount_type"`
	Discount     float64   `json:"discount"`
}

type ExtendDeadlineRequest struct {
	CancelNegotiationRequest
	NewDeadline time.Time `json:"new_deadline"`
	Reason      *string   `json:"reason"`
}

type CompleteRepairsRequest struct {
	CancelNegotiationRequest
	CompletionNotes *string               `json:"completion_notes"`
	RepairPhotos    []RepairPhotoMetadata `json:"repair_photos"`
}
