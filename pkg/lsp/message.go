package lsp

type Message struct {
	JSONRPC string `json:"jsonrpc"`
}

type RequestMessage struct {
	ID     string      `json:"id"`
	Method string      `json:"method"`
	Params interface{} `json:"params,omitempty"`

	Message
}

type ResponseMessage struct {
	ID     string      `json:"id"`
	Result interface{} `json:"result,omitempty"`
	Error  *Error      `json:"error,omitempty"`

	Message
}

type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omityempty"`
}

type ErrorCode int

const (
	ParseError           ErrorCode = -32700
	InvalidRequest       ErrorCode = -32600
	MethodNotFound       ErrorCode = -32601
	InvalidParams        ErrorCode = -32602
	InternalError        ErrorCode = -32603
	serverErrorStart     ErrorCode = -32099
	serverErrorEnd       ErrorCode = -32000
	ServerNotInitialized ErrorCode = -32002
	UnknownErrorCode     ErrorCode = -32001

	// Defined by the language server protocol.
	RequestCancelled ErrorCode = -32800
)

type NotificationMessage struct {
	Method string      `json:"method"`
	Params interface{} `json:"params,omitempty"`
}

type CancelParams struct {
	ID string `json:"id"`
}
