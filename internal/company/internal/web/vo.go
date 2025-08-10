package web

type SaveCompanyReq struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type CompanyVO struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Ctime int64  `json:"ctime"`
	Utime int64  `json:"utime"`
}

type IdReq struct {
	Id int64 `json:"id"`
}

type Page struct {
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

type ListCompanyResp struct {
	List  []CompanyVO `json:"list"`
	Total int64       `json:"total"`
}
