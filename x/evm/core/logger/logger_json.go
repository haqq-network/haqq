package logger

import (
	"encoding/json"
	"io"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"

	"github.com/haqq-network/haqq/x/evm/core/vm"
)

type JSONLogger struct {
	encoder *json.Encoder
	cfg     *Config
	env     *vm.EVM
}

// NewJSONLogger creates a new EVM tracer that prints execution steps as JSON objects
// into the provided stream.
func NewJSONLogger(cfg *Config, writer io.Writer) *JSONLogger {
	l := &JSONLogger{encoder: json.NewEncoder(writer), cfg: cfg}
	if l.cfg == nil {
		l.cfg = &Config{}
	}
	return l
}

func (l *JSONLogger) CaptureStart(env *vm.EVM, from, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	l.env = env
}

func (l *JSONLogger) CaptureFault(pc uint64, op vm.OpCode, gas uint64, cost uint64, scope *vm.ScopeContext, depth int, err error) {
	// TODO: Add rData to this interface as well
	l.CaptureState(pc, op, gas, cost, scope, nil, depth, err)
}

// CaptureState outputs state information on the logger.
func (l *JSONLogger) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	memory := scope.Memory
	stack := scope.Stack

	log := StructLog{
		Pc:            pc,
		Op:            op,
		Gas:           gas,
		GasCost:       cost,
		MemorySize:    memory.Len(),
		Depth:         depth,
		RefundCounter: l.env.StateDB.GetRefund(),
		Err:           err,
	}
	if l.cfg.EnableMemory {
		log.Memory = memory.Data()
	}
	if !l.cfg.DisableStack {
		log.Stack = stack.Data
	}
	if l.cfg.EnableReturnData {
		log.ReturnData = rData
	}
	l.encoder.Encode(log)
}

// CaptureEnd is triggered at end of execution.
func (l *JSONLogger) CaptureEnd(output []byte, gasUsed uint64, t time.Duration, err error) {
	type endLog struct {
		Output  string              `json:"output"`
		GasUsed math.HexOrDecimal64 `json:"gasUsed"`
		Time    time.Duration       `json:"time"`
		Err     string              `json:"error,omitempty"`
	}
	var errMsg string
	if err != nil {
		errMsg = err.Error()
	}
	l.encoder.Encode(endLog{common.Bytes2Hex(output), math.HexOrDecimal64(gasUsed), t, errMsg})
}

func (l *JSONLogger) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
}

func (l *JSONLogger) CaptureExit(output []byte, gasUsed uint64, err error) {}

func (l *JSONLogger) CaptureTxStart(gasLimit uint64) {}

func (l *JSONLogger) CaptureTxEnd(restGas uint64) {}
