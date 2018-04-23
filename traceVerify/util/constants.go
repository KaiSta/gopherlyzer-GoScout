package util

const (
	PREPARE   = 1 << iota
	COMMIT    = 1 << iota
	SEND      = 1 << iota
	RCV       = 1 << iota
	CLS       = 1 << iota
	SIG       = 1 << iota
	WAIT      = 1 << iota
	WRITE     = 1 << iota
	READ      = 1 << iota
	LOCK      = 1 << iota
	UNLOCK    = 1 << iota
	RLOCK     = 1 << iota
	RUNLOCK   = 1 << iota
	RMVSEND   = 0xF ^ SEND
	RMVRCV    = 0xF ^ RCV
	RMVPREP   = 0xF ^ PREPARE
	RMVCOT    = 0xF ^ COMMIT
	NOPARTNER = "-"
)

const (
	EXCLUSIVE  = 1 << iota
	READSHARED = 1 << iota
	SHARED     = 1 << iota
)
