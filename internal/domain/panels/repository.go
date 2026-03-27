package panels

import (
	"eclaim-workshop-deck-api/internal/models"
	"time"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetAllPanels() ([]models.Panel, error) {
	var panels []models.Panel

	err := r.db.
		Where("is_locked = ?", 0).
		Order("panel_name asc").
		Find(&panels).Error

	return panels, err
}

func (r *Repository) GetAllWorkshopPanels(id uint) ([]models.WorkshopPanels, error) {
	var workshopPanels []models.WorkshopPanels

	err := r.db.
		Preload("Panel").
		Where("is_locked = ?", 0).
		Where("(workshop_no IS NULL OR workshop_no = ?)", id).
		Find(&workshopPanels).Error

	return workshopPanels, err
}

func (r *Repository) GetMOUs(insID, woID, mouID uint, activeOnly bool) ([]models.MOU, error) {
	var mous []models.MOU
	query := r.db.Preload("InsurerUserProfile").
		Preload("WorkshopUserProfile").
		Preload("CreatedByUser").
		Where("is_locked = ?", 0)

	// Add filters only if the ID is provided (not 0)
	if insID != 0 {
		query = query.Where("insurer_no = ?", insID)
	}
	if woID != 0 {
		query = query.Where("workshop_no = ?", woID)
	}
	if mouID != 0 {
		query = query.Where("mou_no = ?", mouID)
	}

	// Add expiry filter only if requested
	if activeOnly {
		query = query.Where("(mou_expiry_date > ? OR mou_expiry_date IS NULL)", time.Now())
	}

	err := query.Find(&mous).Error
	return mous, err
}

func (r *Repository) GetPanelPricings(insID, woID, mouID uint) ([]models.PanelPricing, error) {
	var panelPricings []models.PanelPricing
	query := r.db.
		Joins("LEFT JOIN r_workshop_panels ON r_workshop_panels.workshop_panel_no = r_panel_pricing.workshop_panel_no").
		Preload("Insurer").
		Preload("Workshop").
		Preload("WorkshopPanels").
		Preload("Mou").
		Preload("CreatedByUser").
		Preload("LastModifiedByUser").
		Preload("Measurements", "is_locked = ?", false)

	// Add filters only if the ID is provided (not 0)
	if insID != 0 {
		query = query.Where("r_panel_pricing.insurer_no = ?", insID)
	}
	if woID != 0 {
		query = query.Where("r_panel_pricing.workshop_no = ?", woID)
	}
	if mouID != 0 {
		query = query.Where("r_panel_pricing.mou_no = ?", mouID)
	}

	err := query.
		Order("r_workshop_panels.panel_name ASC").
		Find(&panelPricings).Error

	return panelPricings, err
}

func (r *Repository) GetAllPanelPricingsForNegotiation(insID, woID, mouID uint) ([]models.PanelPricing, error) {
	var panelPricings []models.PanelPricing

	query := r.db.
		Joins("LEFT JOIN r_workshop_panels ON r_workshop_panels.workshop_panel_no = r_panel_pricing.workshop_panel_no").
		Preload("Insurer").
		Preload("Workshop").
		Preload("WorkshopPanels").
		Preload("Mou").
		Preload("CreatedByUser").
		Preload("LastModifiedByUser").
		Preload("Measurements", "is_locked = ?", false)

	// Add filters only if the ID is provided (not 0)
	if insID != 0 {
		query = query.Where("r_panel_pricing.insurer_no = ? or r_panel_pricing.insurer_no IS NULL", insID)
	}

	if woID != 0 {
		query = query.Where("r_panel_pricing.workshop_no = ?", woID)
	}

	if mouID != 0 {
		query = query.Where("r_panel_pricing.mou_no = ? OR r_panel_pricing.mou_no IS NULL", mouID)
	}

	err := query.
		Order("r_workshop_panels.panel_name ASC").
		Find(&panelPricings).Error

	return panelPricings, err

}

func (r *Repository) GetWorkshopPanelPricings(woID uint) ([]models.PanelPricing, error) {
	var panelPricings []models.PanelPricing
	err := r.db.
		Joins("LEFT JOIN r_workshop_panels ON r_workshop_panels.workshop_panel_no = r_panel_pricing.workshop_panel_no").
		Preload("Workshop").
		Preload("WorkshopPanels").
		Preload("CreatedByUser").
		Preload("LastModifiedByUser").
		Preload("Measurements", "is_locked = ?", false).
		Where("r_panel_pricing.insurer_no IS NULL AND r_panel_pricing.mou_no IS NULL AND r_panel_pricing.workshop_no = ? AND r_panel_pricing.is_locked = 0", woID).
		Order("r_workshop_panels.panel_name ASC").
		Find(&panelPricings).Error

	return panelPricings, err
}

func (r *Repository) FindMOUByID(id uint) (*models.MOU, error) {
	var mou models.MOU

	err := r.db.
		Preload("InsurerUserProfile").
		Preload("WorkshopUserProfile").
		Preload("CreatedByUser").
		Where("is_locked = ?", 0).
		Where("mou_no", id).
		First(&mou).Error

	return &mou, err
}

func (r *Repository) FindPanelPricingById(id uint) (*models.PanelPricing, error) {
	var panelPricing models.PanelPricing

	err := r.db.
		Preload("Insurer").
		Preload("Workshop").
		Preload("CreatedByUser").
		Preload("Mou").
		Preload("WorkshopPanels").
		Preload("Measurements", "is_locked = ?", false).
		Where("panel_pricing_no", id).
		First(&panelPricing).Error

	return &panelPricing, err
}

func (r *Repository) FindWorkshopPanelById(id uint) (*models.WorkshopPanels, error) {
	var workshopPanel models.WorkshopPanels

	err := r.db.
		Preload("Workshop").
		Preload("Panel").
		Where("workshop_panel_no = ? AND is_locked = 0", id).
		First(&workshopPanel).Error

	return &workshopPanel, err
}

func (r *Repository) CreateMOU(mou *models.MOU) error {
	return r.db.Create(mou).Error
}

func (r *Repository) CreatePanelPricing(panelPricing *models.PanelPricing) error {
	return r.db.Create(panelPricing).Error
}

func (r *Repository) CreateWorkshopPanel(workshopPanel *models.WorkshopPanels) error {
	return r.db.Create(workshopPanel).Error
}

func (r *Repository) CreateMeasurement(measurement *models.Measurement) error {
	return r.db.Create(measurement).Error
}

// Update
func (r *Repository) UpdatePanelPricing(panelPricing *models.PanelPricing) error {
	return r.db.Save(panelPricing).Error
}

// Delete
func (r *Repository) SoftDeleteMeasurementsByPanelPricingNo(panelPricingNo uint) error {
	return r.db.Model(&models.Measurement{}).
		Where("panel_pricing_no = ? AND is_locked = ?", panelPricingNo, 0).
		Update("is_locked", 1).Error
}
