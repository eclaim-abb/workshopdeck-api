package orders

import "eclaim-workshop-deck-api/internal/models"

func (s *Service) GetInvoicedOrders(workshopId uint) ([]models.Order, error) {
	return s.repo.GetInvoicedOrders(workshopId)
}
