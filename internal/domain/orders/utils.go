package orders

import (
	"eclaim-workshop-deck-api/internal/domain/panels"
	"eclaim-workshop-deck-api/internal/models"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (s *Service) prepareOrderPanels(req OrderPanelRequest, createdBy, workOrderNo uint) (*models.OrderPanel, error) {
	panelService := panels.NewRepository(s.repo.db)

	orderPanel := &models.OrderPanel{
		WorkOrderNo:          workOrderNo,
		CreatedBy:            &createdBy,
		WorkOrderGroupNumber: 0,
	}

	if req.WOGroupNumber != 0 {
		orderPanel.WorkOrderGroupNumber = req.WOGroupNumber
	}

	if req.InsurancePanelPricingNo != 0 {
		orderPanel.InsurancePanelPricingNo = &req.InsurancePanelPricingNo

		insurancePanel, err := panelService.FindPanelPricingById(req.InsurancePanelPricingNo)

		if err != nil {
			return nil, errors.New("insurance panel pricing no is invalid")
		}

		orderPanel.InsurancePanelName = &insurancePanel.WorkshopPanels.PanelName
		orderPanel.InsurerServiceType = &insurancePanel.ServiceType

		if insurancePanel.InsurerNo == nil || *insurancePanel.InsurerNo == 0 {
			orderPanel.IsSpecialRepair = false
		} else {
			orderPanel.IsSpecialRepair = true
		}
		switch insurancePanel.ServiceType {
		case "repair":
			if req.InsurerMeasurementNo != 0 {
				orderPanel.InsurerMeasurementNo = &req.InsurerMeasurementNo

				var chosenMeasurement models.Measurement
				for _, m := range insurancePanel.Measurements {
					if m.MeasurementNo == *orderPanel.InsurerMeasurementNo {
						chosenMeasurement = m
					}
				}

				orderPanel.InsurerPrice = &chosenMeasurement.LaborFee
			} else {
				orderPanel.InsurerPrice = &insurancePanel.LaborFee
			}
		case "replacement":
			total := insurancePanel.LaborFee + insurancePanel.SparePartCost
			orderPanel.InsurerPrice = &total
		default:
			return nil, errors.New("service type error while preparing order panels (insurance panel)")
		}
	}

	if req.InsurerQty != 0 {
		orderPanel.InsurerQty = &req.InsurerQty
	}

	if req.WorkshopPanelPricingNo != 0 {
		orderPanel.WorkshopPanelPricingNo = &req.WorkshopPanelPricingNo

		workshopPanel, err := panelService.FindPanelPricingById(req.WorkshopPanelPricingNo)

		if err != nil {
			return nil, errors.New("insurance panel pricing no is invalid")
		}

		orderPanel.WorkshopPanelName = &workshopPanel.WorkshopPanels.PanelName
		orderPanel.WorkshopServiceType = &workshopPanel.ServiceType

		if workshopPanel.InsurerNo == nil || *workshopPanel.InsurerNo == 0 {
			orderPanel.IsSpecialRepair = false
		} else {
			orderPanel.IsSpecialRepair = true
		}

		switch workshopPanel.ServiceType {
		case "repair":
			if req.WorkshopMeasurementNo != 0 {
				orderPanel.WorkshopMeasurementNo = &req.WorkshopMeasurementNo

				var chosenMeasurement models.Measurement
				for _, m := range workshopPanel.Measurements {
					if m.MeasurementNo == *orderPanel.WorkshopMeasurementNo {
						chosenMeasurement = m
					}
				}

				orderPanel.WorkshopPrice = &chosenMeasurement.LaborFee
			} else {
				orderPanel.WorkshopPrice = &workshopPanel.LaborFee
			}
		case "replacement":
			total := workshopPanel.LaborFee + workshopPanel.SparePartCost
			orderPanel.WorkshopPrice = &total
		default:
			return nil, errors.New("service type error while preparing order panels (insurance panel)")
		}
	}

	if req.WorkshopQty != 0 {
		orderPanel.WorkshopQty = &req.WorkshopQty
	}

	if req.IsIncluded == true {
		orderPanel.IsIncluded = true
	} else {
		orderPanel.IsIncluded = false
	}

	return orderPanel, nil
}

func (s *Service) prepareClient(req AddClientRequest) (*models.Client, error) {
	var client *models.Client

	if req.ClientName == "" {
		return nil, errors.New("client name is required")
	}
	if req.ClientPhone == "" {
		return nil, errors.New("client email is required")
	}
	if req.CityNo == 0 {
		return nil, errors.New("city no is required")
	}
	if req.CityName == "" {
		return nil, errors.New("city name is required")
	}
	if req.VehicleBrandName == "" {
		return nil, errors.New("vehicle brand is required")
	}
	if req.VehicleSeriesName == "" {
		return nil, errors.New("vehicle series name is required")
	}
	if req.VehicleChassisNo == "" {
		return nil, errors.New("vehicle chassis no is required")
	}
	if req.VehicleLicensePlate == "" {
		return nil, errors.New("vehicle license plate is required")
	}
	if req.VehiclePrice == 0 {
		return nil, errors.New("vehicle price is required")
	}

	client = &models.Client{
		ClientName:          req.ClientName,
		ClientPhone:         req.ClientPhone,
		CityNo:              req.CityNo,
		CityType:            req.CityType,
		CityName:            req.CityName,
		Address:             req.Address,
		VehicleBrandName:    req.VehicleBrandName,
		VehicleSeriesName:   req.VehicleSeriesName,
		VehicleChassisNo:    req.VehicleChassisNo,
		VehicleLicensePlate: req.VehicleLicensePlate,
		VehiclePrice:        req.VehiclePrice,
	}

	if req.ClientEmail != "" {
		client.ClientEmail = req.ClientEmail
	}

	return client, nil
}

func (s *Service) rejectOrderPanelTx(tx *gorm.DB, orderPanelNo, lastModifiedBy, targetGroupNo uint) (*models.OrderPanel, error) {
	orderPanel, err := s.repo.GetOrderPanelWithLock(tx, orderPanelNo)
	if err != nil {
		return nil, fmt.Errorf("failed to lock order panel %d: %w", orderPanelNo, err)
	}

	if orderPanel.WorkOrderGroupNumber == targetGroupNo &&
		orderPanel.NegotiationStatus != "accepted" &&
		orderPanel.NegotiationStatus != "rejected" {

		// Cancel negotiation if exists
		if orderPanel.CurrentRound > 0 {
			currentNegotiation, err := s.repo.GetSpecificNegotiationHistoryRound(tx, orderPanel.OrderPanelNo, orderPanel.CurrentRound)
			if err != nil {
				return nil, fmt.Errorf("failed to get negotiation history: %w", err)
			}

			if currentNegotiation != nil {
				curTime := time.Now()
				currentNegotiation.IsLocked = true
				currentNegotiation.LastModifiedBy = &lastModifiedBy
				currentNegotiation.InsuranceDecision = "declined"
				currentNegotiation.InsuranceNotes = "Cancelled by workshop"
				currentNegotiation.CompletedDate = &curTime

				err = s.repo.UpdateNegotiationHistoryTx(tx, currentNegotiation)
				if err != nil {
					return nil, fmt.Errorf("failed to update negotiation history: %w", err)
				}
			}
		}

		// Mark panel as rejected and excluded
		orderPanel.NegotiationStatus = "rejected"
		orderPanel.IsIncluded = false
		orderPanel.IsLocked = true
		orderPanel.LastModifiedBy = &lastModifiedBy

		err = s.repo.UpdateOrderPanelTx(tx, orderPanel)
		if err != nil {
			return nil, fmt.Errorf("failed to update order panel: %w", err)
		}
	}

	return orderPanel, nil
}

func (s *Service) acceptOrderPanelTx(tx *gorm.DB, orderPanelNo, lastModifiedBy uint) (*models.OrderPanel, error) {
	lockedPanel, err := s.repo.GetOrderPanelWithLock(tx, orderPanelNo)
	if err != nil {
		return nil, fmt.Errorf("failed to lock panel %d: %w", orderPanelNo, err)
	}

	if lockedPanel.NegotiationStatus != "pending_workshop" && lockedPanel.NegotiationStatus != "negotiating" {
		// Already accepted or rejected, skip
		return lockedPanel, nil
	}

	// If panel is still pending workshop action, accept it
	if lockedPanel.NegotiationStatus == "pending_workshop" {
		if lockedPanel.InitialProposer == "insurer" {
			lockedPanel.FinalPanelPricingNo = lockedPanel.InsurancePanelPricingNo
			lockedPanel.FinalPanelName = lockedPanel.InsurancePanelName
			lockedPanel.FinalPrice = lockedPanel.InsurerPrice
			lockedPanel.FinalServiceType = lockedPanel.InsurerServiceType
			lockedPanel.FinalMeasurementNo = lockedPanel.InsurerMeasurementNo
			lockedPanel.FinalQty = lockedPanel.InsurerQty
		}

		lockedPanel.NegotiationStatus = "accepted"
		lockedPanel.LastModifiedBy = &lastModifiedBy

		if err := s.repo.UpdateOrderPanelTx(tx, lockedPanel); err != nil {
			return nil, fmt.Errorf("failed to accept old panel %d: %w", orderPanelNo, err)
		}
	}

	return lockedPanel, nil
}

func (s *Service) forwardOrderPanelProposalTx(tx *gorm.DB, orderPanelNo, lastModifiedBy uint) (*models.OrderPanel, error) {
	lockedPanel, err := s.repo.GetOrderPanelWithLock(tx, orderPanelNo)
	if err != nil {
		return nil, fmt.Errorf("failed to lock panel %d: %w", orderPanelNo, err)
	}

	if lockedPanel.NegotiationStatus == "proposed_additional" {
		lockedPanel.NegotiationStatus = "additional_work"
		lockedPanel.LastModifiedBy = &lastModifiedBy
	}

	if err := s.repo.UpdateOrderPanelTx(tx, lockedPanel); err != nil {
		return nil, fmt.Errorf("failed to update panel %d status: %w", orderPanelNo, err)
	}

	return lockedPanel, nil
}

// AttachFullPhotoURLs modifies the returned order by attaching the base URL to the file paths
func AttachFullPhotoURLs(order *models.Order, baseURL string) {
	for wi := range order.WorkOrders {
		for pi := range order.WorkOrders[wi].OrderPanels {
			for hi := range order.WorkOrders[wi].OrderPanels[pi].RepairHistory {
				for ri := range order.WorkOrders[wi].OrderPanels[pi].RepairHistory[hi].RepairPhotos {

					photo := &order.WorkOrders[wi].OrderPanels[pi].RepairHistory[hi].RepairPhotos[ri]

					if photo.PhotoUrl != "" {
						photo.PhotoUrl = baseURL + photo.PhotoUrl
					}
				}
			}
		}
	}

	invoice := order.Invoice

	if invoice != nil && invoice.InvoiceFileUrl != "" {
		order.Invoice.InvoiceFileUrl = baseURL + order.Invoice.InvoiceFileUrl
	}
}

// parseSparePartForm is a shared helper that reads the multipart "data" JSON and the "files[]" slice from a gin context.
func parseSparePartForm(c *gin.Context) (*RequestOrderSparePartRequest, []*multipart.FileHeader, error) {
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		return nil, nil, err
	}

	dataStr := c.PostForm("data")
	if dataStr == "" {
		return nil, nil, errMissingDataField
	}

	var req RequestOrderSparePartRequest
	if err := json.Unmarshal([]byte(dataStr), &req); err != nil {
		return nil, nil, err
	}

	form, err := c.MultipartForm()
	if err != nil {
		return nil, nil, err
	}

	return &req, form.File["files"], nil
}
