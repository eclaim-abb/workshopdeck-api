package orders

import (
	"eclaim-workshop-deck-api/internal/models"
	"eclaim-workshop-deck-api/pkg/utils"
	"errors"
	"fmt"
	"mime/multipart"
	"os"
	"time"

	"gorm.io/gorm"
)

type Service struct {
	repo      *Repository
	jwtSecret string
}

func NewService(repo *Repository) *Service {
	return &Service{
		repo: repo,
	}
}

// Read
func (s *Service) GetOrders() ([]models.Order, error) {
	return s.repo.GetOrders()
}

func (s *Service) ViewOrderDetails(orderNo uint) (*models.Order, error) {
	order, err := s.repo.ViewOrderDetails(orderNo)

	if err != nil {
		return nil, err
	}

	AttachFullPhotoURLs(&order, os.Getenv("BASE_URL"))

	return &order, nil
}

// Create
func (s *Service) AddClient(req AddClientRequest) (*models.Client, error) {
	client, err := s.prepareClient(req)
	if err != nil {
		return nil, err
	}

	if err := s.repo.AddClient(client); err != nil {
		return nil, err
	}

	return s.repo.FindClientById(client.ClientNo)
}

func (s *Service) CreateOrder(req CreateOrderRequest) (*models.Order, error) {
	var orderType string
	if req.InsuranceNo != 0 {
		orderType = "insurance"
	} else {
		orderType = "manual"
	}

	var clientNo uint
	if req.ClientNo != 0 {
		clientNo = req.ClientNo
	} else {
		client, err := s.prepareClient(*req.ClientDetails)
		if err != nil {
			return nil, err
		}
		client.CreatedBy = &req.CreatedBy
		if err := s.repo.AddClient(client); err != nil {
			return nil, err
		}

		clientNo = client.ClientNo
	}

	if req.WorkshopNo == 0 {
		return nil, errors.New("workshop no is required")
	}
	if req.ClaimDetails == "" {
		return nil, errors.New("claim details is required")
	}
	if req.CreatedBy == 0 {
		return nil, errors.New("created by is required")
	}
	if req.Status == "" {
		return nil, errors.New("status is required")
	}

	order := &models.Order{
		WorkshopNo:   req.WorkshopNo,
		OrderType:    orderType,
		ClaimDetails: req.ClaimDetails,
		ClientNo:     clientNo,
		CreatedBy:    &req.CreatedBy,
		Status:       req.Status,
	}

	if req.InsuranceNo != 0 {
		order.InsuranceNo = &req.InsuranceNo
	}

	if !req.ETA.IsZero() {
		order.Eta = &req.ETA
	}

	if err := s.repo.CreateOrder(order); err != nil {
		return nil, err
	}

	return s.repo.FindOrderById(order.OrderNo)
}

func (s *Service) CreateWorkOrder(req CreateWorkOrderRequest) (*models.WorkOrder, error) {
	if req.CreatedBy == 0 {
		return nil, errors.New("created by is needed")
	}

	if len(req.OrderPanels) == 0 {
		return nil, errors.New("order panels are needed")
	}

	workOrder := &models.WorkOrder{
		OrderNo:                  req.OrderNo,
		CreatedBy:                &req.CreatedBy,
		AdditionalWorkOrderCount: 0,
	}

	if req.AdditionalWorkOrderCount != 0 {
		workOrder.AdditionalWorkOrderCount = req.AdditionalWorkOrderCount
	}

	if req.WorkOrderDocumentNumber != "" {
		workOrder.WorkOrderDocumentNumber = req.WorkOrderDocumentNumber
	}

	if req.WorkOrderUrl != "" {
		workOrder.WorkOrderUrl = req.WorkOrderUrl
	}

	if err := s.repo.CreateWorkOrder(workOrder); err != nil {
		return nil, err
	}

	var allPanels []*models.OrderPanel

	for _, o := range req.OrderPanels {
		orderPanel, err := s.prepareOrderPanels(o, req.CreatedBy, workOrder.WorkOrderNo)

		if err != nil {
			return nil, err
		}

		allPanels = append(allPanels, orderPanel)
	}

	if err := s.repo.CreateOrderPanelsBatch(allPanels); err != nil {
		return nil, err
	}

	return workOrder, nil
}

// Update
func (s *Service) ProposeAdditionalWork(
	req *ProposeAdditionalWorkRequest,
	files []*multipart.FileHeader,
	uploadFn func(file multipart.File, header *multipart.FileHeader, folder string) (string, error),
) (*models.WorkOrder, error) {
	if req.LastModifiedBy == 0 {
		return nil, errors.New("last_modified_by is required")
	}
	if req.WorkOrderNo == 0 {
		return nil, errors.New("work_order_no is required")
	}
	if len(req.OrderPanels) == 0 {
		return nil, errors.New("order_panels cannot be empty")
	}
	if len(req.Photos) != len(files) {
		return nil, fmt.Errorf("file count mismatch: expected %d files, got %d", len(req.Photos), len(files))
	}

	// Validate files
	allowedTypes := map[string]bool{
		"image/jpeg": true,
		"image/jpg":  true,
		"image/png":  true,
		"image/webp": true,
	}
	maxSize := int64(10 << 20)

	for _, fh := range files {
		if fh.Size > maxSize {
			return nil, fmt.Errorf("file %s exceeds 10MB limit", fh.Filename)
		}
		contentType := fh.Header.Get("Content-Type")
		if !allowedTypes[contentType] {
			return nil, fmt.Errorf("invalid file type %s for file %s", contentType, fh.Filename)
		}
	}

	workOrder, err := s.repo.FindWorkOrderById(uint(req.WorkOrderNo))
	if err != nil {
		return nil, err
	}
	if workOrder == nil {
		return nil, errors.New("work order not found")
	}

	order, err := s.repo.FindOrderById(workOrder.OrderNo)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, errors.New("order not found")
	}

	// Build photo mapping
	type photoEntry struct {
		header  *multipart.FileHeader
		caption string
	}

	photosByPricing := make(map[uint][]photoEntry)

	for _, meta := range req.Photos {
		if meta.FileIndex < 0 || meta.FileIndex >= len(files) {
			return nil, fmt.Errorf("photo file_index %d is out of range", meta.FileIndex)
		}
		photosByPricing[meta.WorkshopPanelPricingNo] = append(
			photosByPricing[meta.WorkshopPanelPricingNo],
			photoEntry{
				header:  files[meta.FileIndex],
				caption: meta.PhotoCaption,
			},
		)
	}

	newGroupNumber := workOrder.AdditionalWorkOrderCount + 1

	var allPanels []*models.OrderPanel
	for _, o := range req.OrderPanels {
		panel, err := s.prepareOrderPanels(o, req.LastModifiedBy, req.WorkOrderNo)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare panel (pricing_no %d): %w", o.WorkshopPanelPricingNo, err)
		}
		panel.NegotiationStatus = "proposed_additional"
		panel.WorkOrderGroupNumber = newGroupNumber
		panel.CurrentRound = 1
		panel.InitialProposer = "workshop"
		allPanels = append(allPanels, panel)
	}

	type uploadedPhoto struct {
		url     string
		caption string
	}
	uploadedByPricing := make(map[uint][]uploadedPhoto)

	folder := fmt.Sprintf(
		"add/%d/%d%02d%02d",
		order.OrderNo,
		time.Now().Year(),
		time.Now().Month(),
		time.Now().Day(),
	)

	for pricingNo, entries := range photosByPricing {
		for _, entry := range entries {
			file, err := entry.header.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to open file %s: %w", entry.header.Filename, err)
			}
			photoURL, err := uploadFn(file, entry.header, folder)
			file.Close()
			if err != nil {
				return nil, fmt.Errorf("failed to upload file %s: %w", entry.header.Filename, err)
			}
			uploadedByPricing[pricingNo] = append(uploadedByPricing[pricingNo], uploadedPhoto{
				url:     photoURL,
				caption: entry.caption,
			})
		}
	}

	err = utils.WithTransaction(s.repo, func(tx *gorm.DB) error {
		// Create panels
		if err := s.repo.CreateOrderPanelsBatchTx(tx, allPanels); err != nil {
			return fmt.Errorf("failed to create order panels: %w", err)
		}

		// Create negotiation history and photos
		for _, panel := range allPanels {
			if panel.WorkshopPanelPricingNo == nil {
				continue
			}

			uploads := uploadedByPricing[*panel.WorkshopPanelPricingNo]
			if len(uploads) == 0 {
				continue
			}

			negotiationHistory := &models.NegotiationHistory{
				OrderPanelNo:           panel.OrderPanelNo,
				RoundCount:             panel.CurrentRound,
				ProposedPanelPricingNo: *panel.WorkshopPanelPricingNo,
				ProposedPrice:          *panel.WorkshopPrice,
				ProposedServiceType:    *panel.WorkshopServiceType,
				ProposedQty:            *panel.WorkshopQty,
				InsuranceDecision:      "pending",
				CreatedBy:              &req.LastModifiedBy,
			}

			if panel.WorkshopMeasurementNo != nil && *panel.WorkshopMeasurementNo != 0 {
				negotiationHistory.ProposedMeasurementNo = panel.WorkshopMeasurementNo
			}

			if err := s.repo.CreateNegotiationHistory(tx, negotiationHistory); err != nil {
				return fmt.Errorf("failed to create negotiation history for panel %d: %w", panel.OrderPanelNo, err)
			}

			// Create photos
			photoRecords := make([]models.NegotiationPhotos, 0, len(uploads))
			for _, up := range uploads {
				photoRecords = append(photoRecords, models.NegotiationPhotos{
					NegotiationHistoryNo: negotiationHistory.NegotiationHistoryNo,
					PhotoCaption:         up.caption,
					PhotoUrl:             up.url,
					CreatedBy:            &req.LastModifiedBy,
				})
			}
			if err := s.repo.CreateNegotiationPhotos(tx, photoRecords); err != nil {
				return fmt.Errorf("failed to create negotiation photos for panel %d: %w", panel.OrderPanelNo, err)
			}
		}

		// Update work order
		workOrder.AdditionalWorkOrderCount = newGroupNumber
		workOrder.LastModifiedBy = &req.LastModifiedBy
		if err := s.repo.UpdateWorkOrderTx(tx, workOrder); err != nil {
			return fmt.Errorf("failed to update work order: %w", err)
		}

		// Update order
		order.Status = "proposed_additional"
		order.LastModifiedBy = &req.LastModifiedBy
		if err := s.repo.UpdateOrderTx(tx, order); err != nil {
			return fmt.Errorf("failed to update order status: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return workOrder, nil
}
