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
	items           map[SPUCategory]map[SPUCategory][]order.Item
	categoryTypeSet CategoryTypeSet
}

func NewCategorizedItems() *CategorizedItems {
	return &CategorizedItems{
		items:           make(map[SPUCategory]map[SPUCategory][]order.Item),
		categoryTypeSet: make(CategoryTypeSet),
	}
}

func (c *CategorizedItems) AddItem(category0 SPUCategory, category1 SPUCategory, item order.Item) {
	if c.items[category0] == nil {
		c.items[category0] = make(map[SPUCategory][]order.Item)
		c.categoryTypeSet[category0] = make(map[SPUCategory]struct{})
	}
	c.items[category0][category1] = append(c.items[category0][category1], item)
	c.categoryTypeSet[category0][category1] = struct{}{}
}

func (c *CategorizedItems) GetItems(category0 SPUCategory, category1 SPUCategory) []order.Item {
	return c.items[category0][category1]
}

func (c *CategorizedItems) CategoriesAndTypes() CategoryTypeSet {
	return c.categoryTypeSet
}
