package state

type HostState struct {
	FailCount    int
	SuccessCount int

	ActiveIP string

	Failed bool
}


type Transition int

const (
	NoChange Transition = iota
	Failed
	Recovered
)


func (s *HostState) MarkHealthy(successThreshold int) Transition {

	s.FailCount = 0
	s.SuccessCount++

	if s.Failed && s.SuccessCount >= successThreshold {

		s.Failed = false

		return Recovered
	}

	return NoChange
}


func (s *HostState) MarkFailed(failThreshold int) Transition {

	s.SuccessCount = 0
	s.FailCount++

	if !s.Failed && s.FailCount >= failThreshold {

		s.Failed = true

		return Failed
	}

	return NoChange
}