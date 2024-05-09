package domain

type Interactive struct {
	Biz        string
	BizId      int64
	ViewCnt    int
	LikeCnt    int
	CollectCnt int
	Liked      bool
	Collected  bool
}
