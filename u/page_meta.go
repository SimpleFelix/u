package u

// PageMeta 通用的分页请求参数模型
type PageMeta struct {
	// 从某个条件（一般是ID或日期）开始查询数据，和Page参数二选一
	Start interface{} `json:"start,omitempty" form:"start" uri:"start" bson:"start"`

	// 页数，和Start参数二选一
	Page *int64 `json:"page,omitempty" form:"page" uri:"page" bson:"page"`

	// 一页的条数
	Size int64 `json:"size,omitempty" form:"size" uri:"size" bson:"size"`

	// 模糊匹配
	Keyword string `json:"keyword,omitempty" form:"keyword" bson:"keyword"`

	// 排序 {"field": -1} -1, desc; 1, asc;
	SortBy map[string]interface{} `json:"sort_by,omitempty" form:"sort_by" bson:"sort_by"`

	// 匹配条件
	Match map[string]interface{} `json:"match,omitempty" form:"match" bson:"match"`

	// 模糊搜索条件
	Search map[string]interface{} `json:"search,omitempty" form:"search" bson:"search"`

	// 总共（约）有多少条记录。仅作为返回值。
	Total *int64 `json:"total,omitempty" form:"total" bson:"total"`
}

// PageMetaSort {"field": -1} -1, desc; 1, asc;
//type PageMetaSort struct {
//	Key   string
//	Value interface{}
//}

//todo wait go 1.18
//type PageResp[T any] struct {
//	*PageMeta
//	data T `json:"data"`
//}
