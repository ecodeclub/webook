// Copyright 2023 ecodeclub
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package order

import (
	"github.com/ecodeclub/webook/internal/order"
)

type CategorizedItems struct {
	items           map[SPUCategory]map[SPUType][]order.Item
	categoryTypeSet CategoryTypeSet
}

func NewCategorizedItems() *CategorizedItems {
	return &CategorizedItems{
		items:           make(map[SPUCategory]map[SPUType][]order.Item),
		categoryTypeSet: make(CategoryTypeSet),
	}
}

func (c *CategorizedItems) AddItem(category SPUCategory, typ SPUType, item order.Item) {
	if c.items[category] == nil {
		c.items[category] = make(map[SPUType][]order.Item)
		c.categoryTypeSet[category] = make(map[SPUType]struct{})
	}
	c.items[category][typ] = append(c.items[category][typ], item)
	c.categoryTypeSet[category][typ] = struct{}{}
}

func (c *CategorizedItems) GetItems(category SPUCategory, typ SPUType) []order.Item {
	return c.items[category][typ]
}

func (c *CategorizedItems) CategoriesAndTypes() CategoryTypeSet {
	return c.categoryTypeSet
}
