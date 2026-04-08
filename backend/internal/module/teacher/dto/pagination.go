package dto

// PaginationQuery 定义通用分页参数。
type PaginationQuery struct {
	Page     int `form:"page" binding:"omitempty,min=1"`
	PageSize int `form:"pageSize" binding:"omitempty,min=1,max=100"`
}

// Normalize 补齐默认分页参数。
// 教师端列表统一限制单页最大 100 条，避免误传超大页大小时放大数据库扫描成本。
func (q PaginationQuery) Normalize() PaginationQuery {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = 20
	}
	if q.PageSize > 100 {
		q.PageSize = 100
	}
	return q
}
