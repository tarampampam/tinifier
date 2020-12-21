package pipeline

// CompressingPipelineOption allows to setup some internal pipeline properties from outside.
type CompressingPipelineOption func(*CompressingPipeline)

// WithPreWorkerRun setups pre worker run handler.
func WithPreWorkerRun(h TaskHandler) CompressingPipelineOption {
	return func(p *CompressingPipeline) { p.preWorkerRun = h }
}
