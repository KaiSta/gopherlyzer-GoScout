package traceReplay

import "fmt"

type SyncPair struct {
	T1         string
	T2         string
	AsyncSend  bool
	AsyncRcv   bool
	Closed     bool
	DoClose    bool
	GoStart    bool
	DataAccess bool
	Sync       bool
	IsSelect   bool
	Idx        int
	T2Idx      int
	IsGoStart  bool
}

func (s *SyncPair) String() string {
	if s.AsyncSend {
		return fmt.Sprintf("Thread %v send async to chan %v", s.T1, s.T2)
	} else if s.AsyncRcv {
		return fmt.Sprintf("Thread %v received async from chan %v", s.T1, s.T2)
	} else if s.Closed {
		return fmt.Sprintf("Thread %v operates on closed chan %v", s.T1, s.T2)
	} else if s.DoClose {
		return fmt.Sprintf("Thread %v closed channel %v", s.T1, s.T2)
	} else if s.DataAccess {
		return fmt.Sprintf("Thread %v accessed var %v", s.T1, s.T2)
	} else if s.Sync {
		return fmt.Sprintf("Thread %v send %v a message", s.T2, s.T1)
	}
	return "UNKNOWN"
}
