package services

import (
	"context"
	"encoding/json"
	"go-stock/backend/data"
	"go-stock/backend/db"
	"go-stock/backend/models"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/coocood/freecache"
)

// QueryBKDictService 板块字典查询服务
type QueryBKDictService struct{}

// QueryBKDict 执行板块字典查询
func (s *QueryBKDictService) QueryBKDict(code string) []any {
	api := data.NewMarketNewsApi()
	resp := api.EMDictCode(code, freecache.NewCache(100))
	return resp
}

// GetQueryBKDictService 获取板块字典查询服务实例
func GetQueryBKDictService() *QueryBKDictService {
	return &QueryBKDictService{}
}