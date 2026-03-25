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
			// Create photo records for ALL uploads
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

			// Insert all photos in one call
			if err := s.repo.CreateRepairPhotosTx(tx, repairPhotoRecords); err != nil {
				return fmt.Errorf("failed to create repair photos for panel %d: %w", req.OrderPanelNo, err)
			}
		}

		// Update order panel status
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
		err := s.repo.WithTransaction(func(tx *gorm.DB) error {
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
