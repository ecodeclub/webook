package cache

import (
	"github.com/ecodeclub/ecache"
)

type CaseCache interface {
}

type caseCache struct {
	ec ecache.Cache
}

func NewCaseCache(ec ecache.Cache) CaseCache {
	return &caseCache{
		ec: &ecache.NamespaceCache{
			C:         ec,
			Namespace: "cases",
		},
	}
}
