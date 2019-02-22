package errors

//OK everything is fine
const OK = 0

//DB errors pack 5**
const (
	LostConnectToDB = 500 + iota
	RowNotFound
	FailedToValidate
	AlreadyUsed
	CantCreate
	CantSave
)
