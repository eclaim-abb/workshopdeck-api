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

func (s *Service) UpdateOrderPanelRepairStatus(req *AddOrderPanelRepairStatus, files []*multipart.FileHeader, uploadFn func(file multipart.File, header *multipart.FileHeader, folder string) (string, error)) (*models.Order, error) {
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

			err = s.repo.UpdateOrderPanelTx(tx, orderPanel)
			if err != nil {
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

// validateSparePartFiles checks MIME type and file size for all uploaded files.
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

// uploadSparePartPhotos uploads all photos referenced by photoMeta and returns
// a map of orderPanelNo → list of uploaded URLs+captions.
func uploadSparePartPhotos(
	photos []SparePartPhotoMetadata,
	files []*multipart.FileHeader,
	folder string,
	uploadFn uploadFnType,
) (map[uint][]struct {
	url     string
	caption string
}, error) {
	result := make(map[uint][]struct {
		url     string
		caption string
	})

	for _, meta := range photos {
		if meta.FileIndex < 0 || meta.FileIndex >= len(files) {
			return nil, fmt.Errorf("photo file_index %d is out of range", meta.FileIndex)
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

		result[meta.OrderPanelNo] = append(result[meta.OrderPanelNo], struct {
			url     string
			caption string
		}{url: photoURL, caption: meta.PhotoCaption})
	}

	return result, nil
}

// RequestSparePart handles spare part requests sent to the insurer.
// At least one entry in req.Requests must be present; each must have a
// description, a qty ≥ 1, and at least one photo.
func (s *Service) RequestSparePart(
	req *RequestOrderSparePartRequest,
	files []*multipart.FileHeader,
	uploadFn uploadFnType,
) (*models.Order, error) {
	// ---- Basic guards ----
	if req.CreatedBy == 0 {
		return nil, errors.New("created_by is needed")
	}
	if len(req.Requests) == 0 {
		return nil, errors.New("at least one request entry is required")
	}
	if len(req.Photos) != len(files) {
		return nil, fmt.Errorf("file count mismatch: expected %d files based on photos metadata, got %d", len(req.Photos), len(files))
	}

	// ---- Validate each request entry ----
	// Track which order panel nos have photos declared
	photoPanelNos := make(map[uint]bool)
	for _, p := range req.Photos {
		photoPanelNos[p.OrderPanelNo] = true
	}

	for _, r := range req.Requests {
		if r.OrderPanelNo == 0 {
			return nil, errors.New("order_panel_no is required for every request entry")
		}
		if r.Description == "" {
			return nil, fmt.Errorf("description is required for order panel %d", r.OrderPanelNo)
		}
		if r.Qty == 0 {
			return nil, fmt.Errorf("qty must be at least 1 for order panel %d", r.OrderPanelNo)
		}
		if !photoPanelNos[r.OrderPanelNo] {
			return nil, fmt.Errorf("at least one photo is required for order panel %d", r.OrderPanelNo)
		}
	}

	// ---- Validate files ----
	if err := validateSparePartFiles(files); err != nil {
		return nil, err
	}

	// ---- Derive order from first panel ----
	firstPanel, err := s.repo.FindOrderPanelById(req.Requests[0].OrderPanelNo)
	if err != nil {
		return nil, fmt.Errorf("order panel %d not found: %w", req.Requests[0].OrderPanelNo, err)
	}
	workOrder, err := s.repo.FindWorkOrderById(firstPanel.WorkOrderNo)
	if err != nil {
		return nil, fmt.Errorf("work order not found: %w", err)
	}

	// ---- Upload photos ----
	now := time.Now()
	folder := fmt.Sprintf(
		"spare-parts/request/%d/%d%02d%02d",
		workOrder.OrderNo,
		now.Year(),
		now.Month(),
		now.Day(),
	)

	uploadedPhotos, err := uploadSparePartPhotos(req.Photos, files, folder, uploadFn)
	if err != nil {
		return nil, err
	}

	// ---- Persist (transaction) ----
	// TODO: replace with real model creates once models.SparePartRequest exists.
	// For now we log and return the order so the handler compiles.
	_ = uploadedPhotos

	order, err := s.repo.FindOrderById(workOrder.OrderNo)
	if err != nil {
		return nil, fmt.Errorf("order not found: %w", err)
	}

	return order, nil
}

// OrderSparePart places a direct spare part order with one or more suppliers.
// Each entry in req.Orders must have a description, qty ≥ 1, price_per_unit > 0,
// at least one supplier, and at least one photo.
func (s *Service) OrderSparePart(
	req *RequestOrderSparePartRequest,
	files []*multipart.FileHeader,
	uploadFn uploadFnType,
) (*models.Order, error) {
	if req.CreatedBy == 0 {
		return nil, errors.New("created_by is needed")
	}
	if len(req.Orders) == 0 {
		return nil, errors.New("at least one order entry is required")
	}
	if len(req.Photos) != len(files) {
		return nil, fmt.Errorf("file count mismatch: expected %d files based on photos metadata, got %d", len(req.Photos), len(files))
	}

	// ---- Validate each order entry ----
	photoPanelNos := make(map[uint]bool)
	for _, p := range req.Photos {
		photoPanelNos[p.OrderPanelNo] = true
	}

	for _, o := range req.Orders {
		if o.OrderPanelNo == 0 {
			return nil, errors.New("order_panel_no is required for every order entry")
		}
		if o.Description == "" {
			return nil, fmt.Errorf("description is required for order panel %d", o.OrderPanelNo)
		}
		if o.Qty == 0 {
			return nil, fmt.Errorf("qty must be at least 1 for order panel %d", o.OrderPanelNo)
		}
		if o.PricePerUnit == 0 {
			return nil, fmt.Errorf("price_per_unit must be greater than 0 for order panel %d", o.OrderPanelNo)
		}
		if len(o.Suppliers) == 0 {
			return nil, fmt.Errorf("at least one supplier is required for order panel %d", o.OrderPanelNo)
		}
		if !photoPanelNos[o.OrderPanelNo] {
			return nil, fmt.Errorf("at least one photo is required for order panel %d", o.OrderPanelNo)
		}

		// Verify each supplier exists
		for _, supplierID := range o.Suppliers {
			if _, err := s.repo.FindSupplierFromID(supplierID); err != nil {
				return nil, fmt.Errorf("supplier %d not found for order panel %d", supplierID, o.OrderPanelNo)
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

	// ---- Upload photos ----
	now := time.Now()
	folder := fmt.Sprintf(
		"spare-parts/order/%d/%d%02d%02d",
		workOrder.OrderNo,
		now.Year(),
		now.Month(),
		now.Day(),
	)

	uploadedPhotos, err := uploadSparePartPhotos(req.Photos, files, folder, uploadFn)
	if err != nil {
		return nil, err
	}

	// ---- Persist (transaction) ----
	// TODO: replace with real model creates once models.SparePartOrder exists.
	_ = uploadedPhotos

	order, err := s.repo.FindOrderById(workOrder.OrderNo)
	if err != nil {
		return nil, fmt.Errorf("order not found: %w", err)
	}

	return order, nil
}

// GetSparePartsTracking returns all spare part requests/orders for a workshop.
// TODO: implement once the tracking models are ready.
func (s *Service) GetSparePartsTracking(workshopID uint) (interface{}, error) {
	if workshopID == 0 {
		return nil, errors.New("workshop_no is required")
	}
	// Placeholder — return empty list until models are added
	return []interface{}{}, nil
}
