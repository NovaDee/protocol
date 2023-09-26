package logger

type zl struct {
	logger Logger
}
type zl1 interface {
	GetLogger() Logger
}

func (p *zl) GetLogger() Logger {
	return p.logger
}

// logger helpers
func LoggerWithParticipant(l Logger, identity string, sid string, isRemote bool) Logger {
	values := make([]interface{}, 0, 4)
	if identity != "" {
		values = append(values, "participant", identity)
	}
	if sid != "" {
		values = append(values, "pID", sid)
	}
	values = append(values, "remote", isRemote)
	// enable sampling per participant
	return l.WithValues(values...)
}

func InitL() *zl {
	participant := LoggerWithParticipant(GetLogger(), "1", "2", true)
	z := &zl{
		logger: participant.WithValues("", ""),
	}

	return z

}
