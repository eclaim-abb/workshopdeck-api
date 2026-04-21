package posts

import "gorm.io/gorm"

type Repository struct {
	db    *gorm.DB
	dbAlt *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(post *Post) error {
	return r.db.Create(post).Error
}

func (r *Repository) FindAll() ([]Post, error) {
	var posts []Post
	err := r.db.Preload("User").Order("created_date desc").Find(&posts).Error
	return posts, err
}

func (r *Repository) FindByUserID(userID uint) ([]Post, error) {
	var posts []Post
	err := r.db.Where("user_no = ?", userID).Order("created_date desc").Find(&posts).Error
	return posts, err
}

func (r *Repository) FindByID(id uint) (*Post, error) {
	var post Post
	err := r.db.Preload("User").First(&post, id).Error
	return &post, err
}

func (r *Repository) Update(post *Post) error {
	return r.db.Save(post).Error
}

func (r *Repository) Delete(id uint) error {
	return r.db.Delete(&Post{}, id).Error
}
