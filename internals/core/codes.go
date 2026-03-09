package core

// CLICommandType holds the string identifier for a CLI/daemon command.
type CLICommandType string

const (
	// Daemon lifecycle
	CommandDaemonShutdown CLICommandType = "DAEMON_SHUTDOWN"
	CommandDaemonKill     CLICommandType = "DAEMON_KILL"

	// Sessions
	CommandSessionCreate    CLICommandType = "SESSION_CREATE"
	CommandSessionCreateRun CLICommandType = "SESSION_CREATE_RUN"
	CommandSessionAttach    CLICommandType = "SESSION_ATTACH"
	CommandSessionStart     CLICommandType = "SESSION_START"
	CommandSessionStop      CLICommandType = "SESSION_STOP"
	CommandSessionRestart   CLICommandType = "SESSION_RESTART"
	CommandSessionResume    CLICommandType = "SESSION_RESUME"
	CommandSessionList      CLICommandType = "SESSION_LIST"
	CommandSessionState     CLICommandType = "SESSION_STATE"
	CommandSessionDelete    CLICommandType = "SESSION_DELETE"

	// Components
	CommandComponentList      CLICommandType = "COMPONENT_LIST"
	CommandComponentGet       CLICommandType = "COMPONENT_GET"
	CommandComponentDescribe  CLICommandType = "COMPONENT_DESCRIBE"
	CommandComponentLogs      CLICommandType = "COMPONENT_LOGS"
	CommandComponentLogPath   CLICommandType = "COMPONENT_LOG_PATH"
	CommandHealthCheck        CLICommandType = "HEALTH_CHECK"
	CommandHealthCheckSession CLICommandType = "HEALTH_CHECK_SESSION"
	CommandHealthCheckComp    CLICommandType = "HEALTH_CHECK_COMPONENT"
)

type ErrorCode = string

const (
	INVALID_PAYLOAD       ErrorCode = "INVALID_PAYLOAD"
	COMPONENT_NOT_FOUND   ErrorCode = "COMPONENT_NOT_FOUND"
	NATS_SUBSCRIBE_FAILED ErrorCode = "NATS_SUBSCRIBE_FAILED"

	INVALID_TARGET     ErrorCode = "INVALID_TARGET"
	INVALID_ENVELOPE   ErrorCode = "INVALID_ENVELOPE"
	UNEXPECTED_MESSAGE ErrorCode = "UNEXPECTED_MESSAGE"
	UNEXPECTED_STATE   ErrorCode = "UNEXPECTED_STATE"

	STATE_TRANSITION_FAILED ErrorCode = "STATE_TRANSITION_FAILED"

	SESSION_NOT_FOUND ErrorCode = "SESSION_NOT_FOUND"

	DECODE_ERROR    ErrorCode = "DECODE_ERROR"
	INVALID_COMMAND ErrorCode = "INVALID_COMMAND"

	MISSING_SESSION_TOKEN        ErrorCode = "MISSING_SESSION_TOKEN"
	WRONG_SESSION_TOKEN          ErrorCode = "WRONG_SESSION_TOKEN"
	SESSION_REGISTRY_UNAVAILABLE ErrorCode = "SESSION_REGISTRY_UNAVAILABLE"
	REGISTRATION_FAILED          ErrorCode = "REGISTRATION_FAILED"
	MISSING_COMPONENT_NAME       ErrorCode = "MISSING_COMPONENT_NAME"

	UNKNOWN_COMMAND    ErrorCode = "UNKNOWN_COMMAND"
	CONFIG_ACK_TIMEOUT ErrorCode = "CONFIG_ACK_TIMEOUT"

	NOT_IMPLEMENTED ErrorCode = "NOT_IMPLEMENTED"
	INTERNAL_ERROR  ErrorCode = "INTERNAL_ERROR"
)

type ForeignType = string

const (
	CommandRegister    ForeignType = "REGISTER"
	CommandRegistered  ForeignType = "REGISTERED"
	CommandStateUpdate ForeignType = "STATE_UPDATE"
	CommandShutdown    ForeignType = "SHUTDOWN"

	CommandACK  ForeignType = "ACK"
	CommandNACK ForeignType = "NACK"

	CommandConfigure  ForeignType = "CONFIGURE"
	CommandConfigured ForeignType = "CONFIGURED"

	CommandPing ForeignType = "PING"
	CommandPong ForeignType = "PONG"

	CommandRuntimeError       ForeignType = "RUNTIME_ERROR"
	CommandRegistrationFailed ForeignType = "REGISTRATION_FAILED"
	CommandAlreadyRegistered  ForeignType = "ALREADY_REGISTERED"

	ERROR ForeignType = "ERROR"
	OK    ForeignType = "OK"

	MessageTypeControl   ForeignType = "CONTROL"
	MessageTypeLifecycle ForeignType = "LIFECYCLE"
	MessageTypeConfig    ForeignType = "CONFIG"
	MessageTypeError     ForeignType = "ERROR"
	MessageTypeHeartbeat ForeignType = "HEARTBEAT"
	MessageTypeData      ForeignType = "DATA"
)

type ForeignComponentState string

const (
	ComponentStateInit         ForeignComponentState = "INIT"
	ComponentStateRegistered   ForeignComponentState = "REGISTERED"
	ComponentStateInitializing ForeignComponentState = "INITIALIZING"
	ComponentStateReady        ForeignComponentState = "READY"
	ComponentStateConfigured   ForeignComponentState = "CONFIGURED"
	ComponentStateRunning      ForeignComponentState = "RUNNING"
	ComponentStateWaiting      ForeignComponentState = "WAITING"
	ComponentStateError        ForeignComponentState = "ERROR"
	ComponentStateFinished     ForeignComponentState = "FINISHED"
	ComponentStateShutdown     ForeignComponentState = "SHUTDOWN"
)

type InternalComponentState string

const (
	NOT_FOUND                InternalComponentState = "NOT_FOUND"
	INVALID_STATE_TRANSITION InternalComponentState = "INVALID_STATE_TRANSITION"
)

type EnvelopeValidationTypes = string

const (
	MISSING_PROTOCOL_VERSION EnvelopeValidationTypes = "MISSING_PROTOCOL_VERSION"
	MISSING_MESSAGE_ID       EnvelopeValidationTypes = "MISSING_MESSAGE_ID"
	MISSING_SOURCE           EnvelopeValidationTypes = "MISSING_SOURCE"
	MISSING_TARGET           EnvelopeValidationTypes = "MISSING_TARGET"
	MISSING_TYPE             EnvelopeValidationTypes = "MISSING_TYPE"
	MISSING_COMMAND          EnvelopeValidationTypes = "MISSING_COMMAND"
	MISSING_PAYLOAD          EnvelopeValidationTypes = "MISSING_PAYLOAD"
)

type SessionStateType = string

const (
	SessionInitialized SessionStateType = "INITIALIZED"
	SessionStarting    SessionStateType = "STARTING"
	SessionRunning     SessionStateType = "RUNNING"
	SessionStopping    SessionStateType = "STOPPING"
	SessionStopped     SessionStateType = "STOPPED"
	SessionFinished    SessionStateType = "FINISHED"
	SessionError       SessionStateType = "ERROR"
)
