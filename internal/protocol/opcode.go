package protocol

const (
	OpDATA  uint16 = 1
	OpPING  uint16 = 2
	OpPONG  uint16 = 3
	OpCLOSE uint16 = 4
	OpERROR uint16 = 5
)

func OpcodeName(op uint16) string {
	switch op {
	case OpDATA:
		return "DATA"
	case OpPING:
		return "PING"
	case OpPONG:
		return "PONG"
	case OpCLOSE:
		return "CLOSE"
	case OpERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

func KnownOpcode(op uint16) bool {
	return op >= OpDATA && op <= OpERROR
}
