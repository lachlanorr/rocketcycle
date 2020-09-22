package txn

type StepStatus int

const (
	Pending  StepStatus = 0
	Complete            = 1
	Error               = 2
)

type MsgId struct {
	Topic     string
	Partition uint64
	Offset    uint64
}

type TxnStep struct {
	Command string
	Params  []string
	Status  stepStatus
	MsgId   MsgId
	Errors  []string
}

type Direction int

const (
	Forward Direction = 0
	Reverse           = 1
)

type TxnDesc struct {
	Direction Direction
	Steps     []TxnStep
}

type StepHandler struct {
	Command string
	Topic   string
	Handler func(params string) error
}

type Processor struct {
	Handlers       map[string]StepHandler
	CommitWriter   func(*TxnDesc) error
	RollbackWriter func(*TxnDesc) error
}
