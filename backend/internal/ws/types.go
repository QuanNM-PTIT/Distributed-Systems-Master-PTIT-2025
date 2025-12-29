package ws

var allowedSignalTypes = map[string]struct{}{
	"signal.offer":        {},
	"signal.answer":       {},
	"signal.ice":          {},
	"chat.busy":           {},
	"group.signal.offer":  {},
	"group.signal.answer": {},
	"group.signal.ice":    {},
}

func isAllowedSignalType(t string) bool {
	_, ok := allowedSignalTypes[t]
	return ok
}

func isGroupSignal(t string) bool {
	switch t {
	case "group.signal.offer", "group.signal.answer", "group.signal.ice":
		return true
	default:
		return false
	}
}
