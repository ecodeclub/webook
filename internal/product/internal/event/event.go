package event

import "github.com/ecodeclub/webook/internal/product/internal/domain"

const (
	CreateProductTopic = "create_product"
)

type SPUEvent struct {
	UID       int64  `json:"uid"`
	ID        int64  `json:"id"`
	SN        string `json:"sn"`
	Name      string `json:"name"`
	Desc      string `json:"desc"`
	Category0 string `json:"category0"`
	Category1 string `json:"category1"`
	SKUs      []SKU  `json:"skus"`
}

type SKU struct {
	ID         int64  `json:"id"`
	SN         string `json:"sn"`
	Name       string `json:"name"`
	Desc       string `json:"desc"`
	Price      int64  `json:"price"`
	Stock      int64  `json:"stock"`
	StockLimit int64  `json:"stockLimit"`
	SaleType   uint8  `json:"saleType"`
	Attrs      string `json:"attrs"`
	Image      string `json:"image"`
}

func (s SPUEvent) ToDomain() domain.SPU {
	spu := domain.SPU{
		ID:        s.ID,
		SN:        s.SN,
		Name:      s.Name,
		Desc:      s.Desc,
		Category0: s.Category0,
		Category1: s.Category1,
	}
	skus := make([]domain.SKU, 0, len(s.SKUs))
	for _, sku := range s.SKUs {
		skus = append(skus, domain.SKU{
			ID:         sku.ID,
			SN:         sku.SN,
			Name:       sku.Name,
			Desc:       sku.Desc,
			Price:      sku.Price,
			Stock:      sku.Stock,
			StockLimit: sku.StockLimit,
			SaleType:   domain.SaleType(sku.SaleType),
			Attrs:      sku.Attrs,
			Image:      sku.Image,
		})
	}
	return spu
}
