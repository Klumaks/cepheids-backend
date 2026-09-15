package repository

import "fmt"

const (
	StatusPublished = "published"
	StatusDraft     = "draft"
	StatusDeleted   = "deleted"
)

// SpectralClass — «услуга» по теме: спектральный класс цефеид
type SpectralClass struct {
	ID          int
	Name        string
	Description string // короткая строка для ленты
	Details     string // текст под «… Больше»
	Status      string
	ImageKey    string
	VideoKey    string
	PlSlope     float64 // наклон a
	PlIntercept float64 // свободный член b
	Likes       []int
}

type Repository struct{ classes []SpectralClass }

func likes(n int) []int {
	s := make([]int, n)
	for i := range s {
		s[i] = i + 1
	}
	return s
}

func NewRepository() (*Repository, error) {
	return &Repository{classes: []SpectralClass{
		{1, "Класс F (F6 Ib)", "Классическая цефеида в полосе нестабильности.", "Зависимость «период–светимость» используется для измерения внегалактических расстояний.", StatusPublished, "class_f.jpg", "class_f.mp4", -2.78, -1.35, likes(24)},
		{2, "Класс G (G2 V)", "Цефеида солнечной температуры.", "Пульсирует с периодом несколько суток, устойчивая фаза.", StatusPublished, "class_g.jpg", "class_g.mp4", -2.65, -1.21, likes(18)},
		{3, "Класс M (M4 III)", "Холодный красный сверхгигант.", "Поздние стадии эволюции, сильные пульсации оболочки.", StatusPublished, "class_m.jpg", "class_m.mp4", -2.92, -1.58, likes(31)},
		{4, "Класс K (K2 Ib)", "Яркая звезда-сверхгигант со стабильными пульсациями.", "Стабильные пульсации, используется для калибровки.", StatusPublished, "class_k.jpg", "class_k.mp4", -2.81, -1.43, likes(12)},
		// Пустой черновик: страница «Добавление» откроется с чистыми полями.
		{5, "", "", "", StatusDraft, "", "", 0, 0, likes(0)},
	}}, nil
}

func (r *Repository) published() []SpectralClass {
	res := []SpectralClass{}
	for _, c := range r.classes {
		if c.Status == StatusPublished {
			res = append(res, c)
		}
	}
	return res
}

func (r *Repository) GetPublishedClasses() ([]SpectralClass, error) { return r.published(), nil }

// GetFiltered — фильтрация ЧИСЛАМИ на сервере: наклон a и свободный член b в диапазоне
func (r *Repository) GetFiltered(minSlope, maxSlope, minB, maxB *float64) ([]SpectralClass, error) {
	res := []SpectralClass{}
	for _, c := range r.published() {
		if minSlope != nil && c.PlSlope < *minSlope {
			continue
		}
		if maxSlope != nil && c.PlSlope > *maxSlope {
			continue
		}
		if minB != nil && c.PlIntercept < *minB {
			continue
		}
		if maxB != nil && c.PlIntercept > *maxB {
			continue
		}
		res = append(res, c)
	}
	return res, nil
}

func (r *Repository) GetClassByID(id int) (SpectralClass, error) {
	for _, c := range r.published() {
		if c.ID == id {
			return c, nil
		}
	}
	return SpectralClass{}, fmt.Errorf("класс %d не найден или не опубликован", id)
}

func (r *Repository) GetNextClass(id int) (SpectralClass, error) {
	pub := r.published()
	for i, c := range pub {
		if c.ID == id {
			return pub[(i+1)%len(pub)], nil
		}
	}
	return SpectralClass{}, fmt.Errorf("next: id %d не найден", id)
}

func (r *Repository) GetDraftClass() (SpectralClass, error) {
	for _, c := range r.classes {
		if c.Status == StatusDraft {
			return c, nil
		}
	}
	return SpectralClass{}, fmt.Errorf("черновик не найден")
}
