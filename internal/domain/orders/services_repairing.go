package orders

import (
	"eclaim-workshop-deck-api/internal/models"
	"errors"
	"fmt"
	"mime/multipart"
	"time"

	"gorm.io/gorm"
)

func (s *Service) GetRepairingOrders(workshopId uint) ([]models.Order, error) {
	return s.repo.GetRepairingOrders(workshopId)
}

func (s *Service) ExtendDeadline(req ExtendDeadlineRequest) (*models.Order, error) {
	if req.LastModifiedBy == 0 {
		return nil, errors.New("last_modified_by is needed")
	}
	if req.NewDeadline.IsZero() {
		return nil, errors.New("new_deadline is needed")
	}
	if req.OrderNo == 0 {
		return nil, errors.New("order_no is needed")
	}

	order, err := s.repo.FindOrderById(req.OrderNo)
	if err != nil {
		return nil, errors.New("order not found")
	}

	order.LastModifiedBy = &req.LastModifiedBy
	order.Eta = req.NewDeadline
	if req.Reason != nil && *req.Reason != "" {
		order.Notes = req.Reason
	}

	if err := s.repo.UpdateOrder(order); err != nil {
		return nil, err
	}

	return order, nil
}

func (s *Service) UpdateOrderPanelRepairStatus(
	req *AddOrderPanelRepairStatus,
	files []*multipart.FileHeader,
	uploadFn func(file multipart.File, header *multipart.FileHeader, folder string) (string, error),
) (*models.Order, error) {
	if req.CreatedBy == 0 {
		return nil, errors.New("created_by is needed")
	}
	if req.Notes == "" {
		return nil, errors.New("notes are needed")
	}
	if req.OrderPanelNo == 0 {
		return nil, errors.New("order_panel_no is needed")
	}
	if req.Status == "" {
		return nil, errors.New("status is needed")
	}

	orderPanel, err := s.repo.FindOrderPanelById(req.OrderPanelNo)
	if err != nil {
		return nil, err
	}

	workOrder, err := s.repo.FindWorkOrderById(orderPanel.WorkOrderNo)
	if err != nil {
		return nil, err
	}

	order, err := s.repo.ViewOrderDetails(workOrder.OrderNo)
	if err != nil {
		return nil, err
	}

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

	type photoEntry struct {
		header    *multipart.FileHeader
		caption   string
		photoType string
	}

	photos := make(map[uint][]photoEntry)

	for _, meta := range req.RepairPhotos {
		if meta.FileIndex < 0 || meta.FileIndex >= len(files) {
			return nil, fmt.Errorf("photo file_index %d is out of range", meta.FileIndex)
		}
		photos[req.OrderPanelNo] = append(
			photos[req.OrderPanelNo],
			photoEntry{
				header:    files[meta.FileIndex],
				caption:   meta.PhotoCaption,
				photoType: meta.PhotoType,
			},
		)
	}

	type uploadedPhoto struct {
		url       string
		caption   string
		photoType string
	}
	uploadedPhotos := make(map[uint][]uploadedPhoto)

	folder := fmt.Sprintf(
		"repair/%d/%d/%d%02d%02d",
		orderPanel.WorkOrderNo,
		orderPanel.OrderPanelNo,
		time.Now().Year(),
		time.Now().Month(),
		time.Now().Day(),
	)

	for photoNo, entries := range photos {
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
			uploadedPhotos[photoNo] = append(uploadedPhotos[photoNo], uploadedPhoto{
				url:       photoURL,
				caption:   entry.caption,
				photoType: entry.photoType,
			})
		}
	}

	err = s.repo.WithTransaction(func(tx *gorm.DB) error {
		repairHistory := &models.RepairHistory{
			OrderPanelNo: req.OrderPanelNo,
			Status:       req.RepairStatus,
			CreatedBy:    &req.CreatedBy,
		}

		if req.RepairStatus != "" {
			repairHistory.Status = req.RepairStatus
		} else {
			latestRepairHistory, err := s.repo.GetLatestRepairHistory(tx, req.OrderPanelNo)
			if err != nil {
				return fmt.Errorf("failed to get latest repair history for order panel %d: %w", req.OrderPanelNo, err)
			}
			latestStatus := latestRepairHistory.Status
			if latestStatus != "" {
				repairHistory.Status = latestStatus
			} else {
				repairHistory.Status = "incomplete"
			}
		}

		if req.Notes != "" {
			repairHistory.Note = req.Notes
		}

		if err := s.repo.CreateRepairHistoryTx(tx, repairHistory); err != nil {
			return fmt.Errorf("failed to create repair history for order panel %d: %w", req.OrderPanelNo, err)
		}

		uploads := uploadedPhotos[req.OrderPanelNo]
		if len(uploads) > 0 {
			repairPhotoRecords := make([]models.RepairPhoto, 0, len(uploads))
			for _, up := range uploads {
				repairPhotoRecords = append(repairPhotoRecords, models.RepairPhoto{
					RepairHistoryNo: &repairHistory.RepairHistoryNo,
					PhotoType:       up.photoType,
					PhotoCaption:    up.caption,
					PhotoUrl:        up.url,
					CreatedBy:       &req.CreatedBy,
				})
			}
			if err := s.repo.CreateRepairPhotosTx(tx, repairPhotoRecords); err != nil {
				return fmt.Errorf("failed to create repair photos for panel %d: %w", req.OrderPanelNo, err)
			}
		}

		if orderPanel.CompletionStatus != req.Status {
			orderPanel.CompletionStatus = req.Status
			orderPanel.LastModifiedBy = &req.CreatedBy
			if err := s.repo.UpdateOrderPanelTx(tx, orderPanel); err != nil {
				return fmt.Errorf("failed to update order panel status: %w", err)
			}
		}

		return nil
	})

	return &order, nil
}

func (s *Service) CompleteRepairs(
	req *CompleteRepairsRequest,
	files []*multipart.FileHeader,
	uploadFn func(file multipart.File, header *multipart.FileHeader, folder string) (string, error),
) (*models.Order, error) {
	if req.LastModifiedBy == 0 {
		return nil, errors.New("last_modified_by is needed")
	}
	if req.OrderNo == 0 {
		return nil, errors.New("order no is required")
	}

	workOrder, err := s.repo.FindWorkOrderFromOrderNo(req.OrderNo)
	if err != nil {
		return nil, errors.New("order not found")
	}

	orderPanels := workOrder.OrderPanels

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

	type uploadedPhoto struct {
		url       string
		caption   string
		photoType string
	}

	now := time.Now()
	folder := fmt.Sprintf(
		"repair/%d/complete/%d%02d%02d",
		workOrder.WorkOrderNo,
		now.Year(),
		now.Month(),
		now.Day(),
	)

	uploadedPhotos := make([]uploadedPhoto, 0, len(req.RepairPhotos))

	for _, meta := range req.RepairPhotos {
		if meta.FileIndex < 0 || meta.FileIndex >= len(files) {
			return nil, fmt.Errorf("photo file_index %d is out of range", meta.FileIndex)
		}
		fh := files[meta.FileIndex]
		file, err := fh.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open file %s: %w", fh.Filename, err)
		}
		url, err := uploadFn(file, fh, folder)
		file.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to upload file %s: %w", fh.Filename, err)
		}
		uploadedPhotos = append(uploadedPhotos, uploadedPhoto{
			url:       url,
			caption:   meta.PhotoCaption,
			photoType: meta.PhotoType,
		})
	}

	for _, op := range orderPanels {
		if op.NegotiationStatus != "" && op.NegotiationStatus == "rejected" {
			continue
		}

		latestHistory, err := s.repo.GetLatestRepairHistory(s.repo.db, op.OrderPanelNo)
		if err != nil {
			return nil, fmt.Errorf("failed to get latest repair history for panel %d: %w", op.OrderPanelNo, err)
		}
		if latestHistory != nil && latestHistory.Status == "completed" {
			continue
		}

		err = s.repo.WithTransaction(func(tx *gorm.DB) error {
			note := ""
			if req.CompletionNotes != nil {
				note = *req.CompletionNotes
			}

			repairHistory := &models.RepairHistory{
				OrderPanelNo: op.OrderPanelNo,
				Status:       "completed",
				Note:         note,
				CreatedBy:    &req.LastModifiedBy,
			}

			if err := s.repo.CreateRepairHistoryTx(tx, repairHistory); err != nil {
				return fmt.Errorf("failed to create repair history for order panel %d: %w", op.OrderPanelNo, err)
			}

			if len(uploadedPhotos) > 0 {
				repairPhotoRecords := make([]models.RepairPhoto, 0, len(uploadedPhotos))
				for _, up := range uploadedPhotos {
					repairPhotoRecords = append(repairPhotoRecords, models.RepairPhoto{
						RepairHistoryNo: &repairHistory.RepairHistoryNo,
						PhotoType:       up.photoType,
						PhotoCaption:    up.caption,
						PhotoUrl:        up.url,
						CreatedBy:       &req.LastModifiedBy,
					})
				}
				if err := s.repo.CreateRepairPhotosTx(tx, repairPhotoRecords); err != nil {
					return fmt.Errorf("failed to create repair photos for panel %d: %w", op.OrderPanelNo, err)
				}
			}

			op.LastModifiedBy = &req.LastModifiedBy
			op.CompletionStatus = "completed"
			if err := s.repo.UpdateOrderPanelTx(tx, &op); err != nil {
				return fmt.Errorf("failed to update order panel for order panel %d: %w", op.OrderPanelNo, err)
			}

			return nil
		})

		if err != nil {
			return nil, err
		}
	}

	order, err := s.repo.FindOrderById(req.OrderNo)
	if err != nil {
		return nil, err
	}

	order.Status = "repaired"
	order.LastModifiedBy = &req.LastModifiedBy

	if err := s.repo.UpdateOrder(order); err != nil {
		return nil, fmt.Errorf("failed to update order for order %d: %w", order.OrderNo, err)
	}

	return order, nil
}

type uploadFnType func(file multipart.File, header *multipart.FileHeader, folder string) (string, error)

// validateSparePartFiles checks MIME type and size for every uploaded file.
func validateSparePartFiles(files []*multipart.FileHeader) error {
	allowedTypes := map[string]bool{
		"image/jpeg": true,
		"image/jpg":  true,
		"image/png":  true,
		"image/webp": true,
	}
	maxSize := int64(10 << 20)

	for _, fh := range files {
		if fh.Size > maxSize {
			return fmt.Errorf("file %s exceeds 10MB limit", fh.Filename)
		}
		if !allowedTypes[fh.Header.Get("Content-Type")] {
			return fmt.Errorf("invalid file type %s for file %s", fh.Header.Get("Content-Type"), fh.Filename)
		}
	}
	return nil
}

// uploadedSparePartPhoto holds the result of a single file upload.
type uploadedSparePartPhoto struct {
	url     string
	caption string
}

// uploadPhotoSlice uploads a slice of SparePartPhotoMetadata and returns the
// results in the same order. file_index values must be valid positions in files.
func uploadPhotoSlice(
	photos []SparePartPhotoMetadata,
	files []*multipart.FileHeader,
	folder string,
	uploadFn uploadFnType,
) ([]uploadedSparePartPhoto, error) {
	results := make([]uploadedSparePartPhoto, 0, len(photos))

	for _, meta := range photos {
		if meta.FileIndex < 0 || meta.FileIndex >= len(files) {
			return nil, fmt.Errorf("photo file_index %d is out of range (have %d files)", meta.FileIndex, len(files))
		}
		fh := files[meta.FileIndex]
		file, err := fh.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open file %s: %w", fh.Filename, err)
		}
		photoURL, err := uploadFn(file, fh, folder)
		file.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to upload file %s: %w", fh.Filename, err)
		}
		results = append(results, uploadedSparePartPhoto{url: photoURL, caption: meta.PhotoCaption})
	}

	return results, nil
}

// ---------------------------------------------------------------------------
// RequestSparePart — requests panels from the insurer
// ---------------------------------------------------------------------------

// RequestSparePart handles spare part requests sent to the insurer.
//
// req.Requests must be non-nil and contain at least one panel entry.
// Each panel must have a qty ≥ 1. The shared description and photo set
// apply to all panels in the request.
func (s *Service) RequestSparePart(
	req *RequestOrderSparePartRequest,
	files []*multipart.FileHeader,
	uploadFn uploadFnType,
) (*models.Order, error) {
	r := req.Requests
	// ---- Guards ----
	if r == nil {
		return nil, errors.New("requests field is required")
	}
	if r.CreatedBy == 0 {
		return nil, errors.New("requests.created_by is required")
	}
	if r.Description == "" {
		return nil, errors.New("requests.description is required")
	}
	if len(r.Panels) == 0 {
		return nil, errors.New("requests.panels must contain at least one entry")
	}
	if len(r.Photos) == 0 {
		return nil, errors.New("requests.photos must contain at least one photo")
	}

	// ---- Validate files ----
	if err := validateSparePartFiles(files); err != nil {
		return nil, err
	}

	// ---- Derive order from first panel ----
	orderPanel, err := s.repo.FindOrderPanelById(r.Panels[0].OrderPanelNo)
	if err != nil {
		return nil, fmt.Errorf("order panel %d not found: %w", r.Panels[0].OrderPanelNo, err)
	}

	workOrder, err := s.repo.FindWorkOrderById(orderPanel.WorkOrderNo)
	if err != nil {
		return nil, fmt.Errorf("work order not found: %w", err)
	}

	order, err := s.repo.ViewOrderDetails(workOrder.OrderNo)
	if err != nil {
		return nil, fmt.Errorf("order not found: %w", err)
	}
	// ---- Upload shared photos ----
	now := time.Now()
	folder := fmt.Sprintf(
		"repair/%d/spare/request/%d%02d%02d",
		workOrder.WorkOrderNo,
		now.Year(),
		now.Month(),
		now.Day(),
	)

	uploadedPhotos, err := uploadPhotoSlice(r.Photos, files, folder, uploadFn)
	if err != nil {
		return nil, err
	}

	err = s.repo.WithTransaction(func(tx *gorm.DB) error {
		for _, p := range r.Panels {
			if p.OrderPanelNo == 0 {
				return errors.New("order_panel_no is required for every panel in requests")
			}
			if p.Qty == 0 {
				return fmt.Errorf("qty must be at least 1 for order panel %d", p.OrderPanelNo)
			}

			repairHistory := &models.RepairHistory{
				OrderPanelNo: p.OrderPanelNo,
				Status:       "requesting",
				Note:         r.Description,
				CreatedBy:    &r.CreatedBy,
			}

			if err := s.repo.CreateRepairHistoryTx(tx, repairHistory); err != nil {
				return fmt.Errorf("failed to create repair history for order panel %d: %w", p.OrderPanelNo, err)
			}

			var finPhotos []models.RepairPhoto
			for _, uP := range uploadedPhotos {
				repairPhoto := &models.RepairPhoto{
					RepairHistoryNo: &repairHistory.RepairHistoryNo,
					PhotoType:       "replacement",
					PhotoCaption:    uP.caption,
					PhotoUrl:        uP.url,
					CreatedBy:       &r.CreatedBy,
				}

				finPhotos = append(finPhotos, *repairPhoto)
			}

			if err := s.repo.CreateRepairPhotosTx(tx, finPhotos); err != nil {
				return fmt.Errorf("failed to create repair photos for repair history %d : %w", repairHistory.RepairHistoryNo, err)
			}

			orderRequest := &models.OrderAndRequest{
				RepairHistoryNo: repairHistory.RepairHistoryNo,
				SparePartStatus: "pending_response",
				NeededQty:       p.Qty,
				Description:     r.Description,
				CreatedBy:       &r.CreatedBy,
			}

			if err := s.repo.CreateOrderAndRequestTx(tx, orderRequest); err != nil {
				return fmt.Errorf("failed to create order and request for repair history %d: %w", repairHistory.RepairHistoryNo, err)
			}

			sparePartQuote, err := s.repo.GetSparePartQuoteTx(tx, orderRequest.OrderRequestNo)

			if err != nil {
				return fmt.Errorf("error in finding spare part quote for order request %d: %w", orderRequest.OrderRequestNo, err)
			}

			if sparePartQuote != nil {
				sparePartQuote.CurrentRound += 1
				sparePartQuote.SupplierStatus = "waiting"
				sparePartQuote.LastModifiedBy = &r.CreatedBy
				sparePartQuote.RequestedStock = &p.Qty
				sparePartQuote.RequestedUnitPrice = &p.PricePerUnit

				if err := s.repo.UpdateSparePartQuoteTx(tx, sparePartQuote); err != nil {
					return fmt.Errorf("failed to update spare part quote for order request %d: %w", orderRequest.OrderRequestNo, err)
				}
			} else {
				sparePartQuote = &models.SparePartQuote{
					OrderRequestNo:     orderRequest.OrderRequestNo,
					InsuranceNo:        order.InsuranceNo,
					CurrentRound:       0,
					SupplierStatus:     "waiting",
					RequestedStock:     &p.Qty,
					RequestedUnitPrice: &p.PricePerUnit,
					CreatedBy:          &r.CreatedBy,
				}

				if err := s.repo.CreateSparePartQuoteTx(tx, sparePartQuote); err != nil {
					return fmt.Errorf("failed to create spare part quote for order request %d: %w", orderRequest.OrderRequestNo, err)
				}
			}

			sparePartNegotiationHistory := &models.SparePartNegotiationHistory{
				SparePartQuotesNo: sparePartQuote.SparePartQuoteNo,
				RoundCount:        sparePartQuote.CurrentRound,
				NewRequestedStock: &p.Qty,
				NewUnitPrice:      &p.PricePerUnit,
				Status:            "pending",
				CreatedBy:         &r.CreatedBy,
			}

			if err := s.repo.CreateSparePartNegotiationHistoryTx(tx, sparePartNegotiationHistory); err != nil {
				return fmt.Errorf("failed to create spare part negotiation history for spare part quote %d: %w", sparePartQuote.SparePartQuoteNo, err)
			}
		}

		return nil
	})

	// ---- Persist ----
	// TODO: insert into models.SparePartRequest once that model exists.
	// uploadedPhotos and r.Panels are ready to use.
	_ = uploadedPhotos

	order, err = s.repo.ViewOrderDetails(workOrder.OrderNo)
	if err != nil {
		return nil, fmt.Errorf("order not found: %w", err)
	}

	return &order, nil
}

// ---------------------------------------------------------------------------
// OrderSparePart — orders panels directly from suppliers
// ---------------------------------------------------------------------------

// OrderSparePart places direct spare part orders with suppliers.
//
// req.Orders must be non-empty. Each entry must have:
//   - order_panel_no, description, qty ≥ 1, price_per_unit > 0
//   - at least one supplier (all verified to exist)
//   - at least one photo
func (s *Service) OrderSparePart(
	req *RequestOrderSparePartRequest,
	files []*multipart.FileHeader,
	uploadFn uploadFnType,
) (*models.Order, error) {
	// ---- Guards ----
	if len(req.Orders) == 0 {
		return nil, errors.New("orders must contain at least one entry")
	}

	for i, o := range req.Orders {
		label := fmt.Sprintf("orders[%d]", i)
		if o.OrderPanelNo == 0 {
			return nil, fmt.Errorf("%s: order_panel_no is required", label)
		}
		if o.CreatedBy == 0 {
			return nil, fmt.Errorf("%s: created_by is required", label)
		}
		if o.Description == "" {
			return nil, fmt.Errorf("%s (panel %d): description is required", label, o.OrderPanelNo)
		}
		if o.Qty == 0 {
			return nil, fmt.Errorf("%s (panel %d): qty must be at least 1", label, o.OrderPanelNo)
		}
		if o.PricePerUnit == 0 {
			return nil, fmt.Errorf("%s (panel %d): price_per_unit must be greater than 0", label, o.OrderPanelNo)
		}
		if len(o.Suppliers) == 0 {
			return nil, fmt.Errorf("%s (panel %d): at least one supplier is required", label, o.OrderPanelNo)
		}
		if len(o.Photos) == 0 {
			return nil, fmt.Errorf("%s (panel %d): at least one photo is required", label, o.OrderPanelNo)
		}

		for _, supplierID := range o.Suppliers {
			if _, err := s.repo.FindSupplierFromID(supplierID); err != nil {
				return nil, fmt.Errorf("%s (panel %d): supplier %d not found", label, o.OrderPanelNo, supplierID)
			}
		}
	}

	// ---- Validate files ----
	if err := validateSparePartFiles(files); err != nil {
		return nil, err
	}

	// ---- Derive order from first panel ----
	firstPanel, err := s.repo.FindOrderPanelById(req.Orders[0].OrderPanelNo)
	if err != nil {
		return nil, fmt.Errorf("order panel %d not found: %w", req.Orders[0].OrderPanelNo, err)
	}

	workOrder, err := s.repo.FindWorkOrderById(firstPanel.WorkOrderNo)
	if err != nil {
		return nil, fmt.Errorf("work order not found: %w", err)
	}

	orderDetails, err := s.repo.ViewOrderDetails(workOrder.OrderNo)
	if err != nil {
		return nil, fmt.Errorf("order not found: %w", err)
	}
	now := time.Now()

	// ---- Upload per-order photos and collect results ----
	type orderWithPhotos struct {
		order  OrderDataRequest
		photos []uploadedSparePartPhoto
	}
	ordersWithPhotos := make([]orderWithPhotos, 0, len(req.Orders))

	for _, o := range req.Orders {
		folder := fmt.Sprintf(
			"repair/%d/spare/order/%d/%d%02d%02d",
			workOrder.WorkOrderNo,
			o.OrderPanelNo,
			now.Year(),
			now.Month(),
			now.Day(),
		)
		uploaded, err := uploadPhotoSlice(o.Photos, files, folder, uploadFn)
		if err != nil {
			return nil, fmt.Errorf("panel %d: %w", o.OrderPanelNo, err)
		}
		ordersWithPhotos = append(ordersWithPhotos, orderWithPhotos{order: o, photos: uploaded})
	}

	err = s.repo.WithTransaction(func(tx *gorm.DB) error {
		for i, o := range req.Orders {
			repairHistory := &models.RepairHistory{
				OrderPanelNo: o.OrderPanelNo,
				Status:       "ordering",
				Note:         o.Description,
				CreatedBy:    &o.CreatedBy,
			}

			if err := s.repo.CreateRepairHistoryTx(tx, repairHistory); err != nil {
				return fmt.Errorf("failed to create repair history for order panel %d: %w", o.OrderPanelNo, err)
			}

			repPhotos := ordersWithPhotos[i]
			uploadedPhotos := repPhotos.photos

			var finPhotos []models.RepairPhoto
			for _, uP := range uploadedPhotos {
				repairPhoto := &models.RepairPhoto{
					RepairHistoryNo: &repairHistory.RepairHistoryNo,
					PhotoType:       "replacement",
					PhotoCaption:    uP.caption,
					PhotoUrl:        uP.url,
					CreatedBy:       &o.CreatedBy,
				}

				finPhotos = append(finPhotos, *repairPhoto)
			}

			if err := s.repo.CreateRepairPhotosTx(tx, finPhotos); err != nil {
				return fmt.Errorf("failed to create repair photos for repair history %d : %w", repairHistory.RepairHistoryNo, err)
			}

			orderRequest := &models.OrderAndRequest{
				RepairHistoryNo: repairHistory.RepairHistoryNo,
				SparePartStatus: "pending_response",
				NeededQty:       o.Qty,
				Description:     o.Description,
				CreatedBy:       &o.CreatedBy,
			}

			if err := s.repo.CreateOrderAndRequestTx(tx, orderRequest); err != nil {
				return fmt.Errorf("failed to create order and request for repair history %d: %w", repairHistory.RepairHistoryNo, err)
			}

			sparePartQuote, err := s.repo.GetSparePartQuoteTx(tx, orderRequest.OrderRequestNo)

			if err != nil {
				return fmt.Errorf("error in finding spare part quote for order request %d: %w", orderRequest.OrderRequestNo, err)
			}

			if sparePartQuote != nil {
				sparePartQuote.CurrentRound += 1
				sparePartQuote.SupplierStatus = "waiting"
				sparePartQuote.LastModifiedBy = &o.CreatedBy
				sparePartQuote.RequestedStock = &o.Qty
				sparePartQuote.RequestedUnitPrice = &o.PricePerUnit

				if err := s.repo.UpdateSparePartQuoteTx(tx, sparePartQuote); err != nil {
					return fmt.Errorf("failed to update spare part quote for order request %d: %w", orderRequest.OrderRequestNo, err)
				}
			} else {
				sparePartQuote = &models.SparePartQuote{
					OrderRequestNo:     orderRequest.OrderRequestNo,
					InsuranceNo:        orderDetails.InsuranceNo,
					CurrentRound:       0,
					SupplierStatus:     "waiting",
					RequestedStock:     &o.Qty,
					RequestedUnitPrice: &o.PricePerUnit,
					CreatedBy:          &o.CreatedBy,
				}

				if err := s.repo.CreateSparePartQuoteTx(tx, sparePartQuote); err != nil {
					return fmt.Errorf("failed to create spare part quote for order request %d: %w", orderRequest.OrderRequestNo, err)
				}
			}

			sparePartNegotiationHistory := &models.SparePartNegotiationHistory{
				SparePartQuotesNo: sparePartQuote.SparePartQuoteNo,
				RoundCount:        sparePartQuote.CurrentRound,
				NewRequestedStock: &o.Qty,
				NewUnitPrice:      &o.PricePerUnit,
				Status:            "pending",
				CreatedBy:         &o.CreatedBy,
			}

			if err := s.repo.CreateSparePartNegotiationHistoryTx(tx, sparePartNegotiationHistory); err != nil {
				return fmt.Errorf("failed to create spare part negotiation history for spare part quote %d: %w", sparePartQuote.SparePartQuoteNo, err)
			}
		}

		return nil
	})

	order, err := s.repo.ViewOrderDetails(workOrder.OrderNo)
	if err != nil {
		return nil, fmt.Errorf("order not found: %w", err)
	}

	return &order, nil
}
