package builtin

import (
	"github.com/packetmind/packetmind/internal/agent/tools"
	"github.com/packetmind/packetmind/internal/storage"
)

var (
	NewGetRequestHandler               = newGetRequestHandler
	NewSearchByHeaderHandler           = newSearchByHeaderHandler
	NewSearchByBodyHandler             = newSearchByBodyHandler
	NewSearchByResponseHandler         = newSearchByResponseHandler
	NewAnalyzeEncodingHandler          = newAnalyzeEncodingHandler
	NewSearchAllFieldsHandler          = newSearchAllFieldsHandler
	NewFindPriorResponseSourcesHandler = newFindPriorResponseSourcesHandler
	NewFindLaterRequestUsagesHandler   = newFindLaterRequestUsagesHandler
	NewTraceValueFlowHandler           = newTraceValueFlowHandler
	NewDiffRequestsHandler             = newDiffRequestsHandler
	NewBatchExecuteHandler             = newBatchExecuteHandler
	NewBashHandler                     = newBashHandler
	NewSummarizeSessionHandler         = newSummarizeSessionHandler
	NewClassifyRequestsHandler         = newClassifyRequestsHandler
	NewTraceFlowSequenceHandler        = newTraceFlowSequenceHandler
	NewTestHypothesisHandler           = newTestHypothesisHandler
)

type BuiltinHandlerFunc = tools.BuiltinHandler
type Executor = tools.Executor
type Storage = storage.Storage