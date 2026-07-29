package log

type Level int

const (
	Debug Level = iota
	Info
	Warn
	Error
	Fatal
)

func (l Level) String() string {
	switch l {
	case Debug:
		return "DEBUG"
	case Info:
		return "INFO"
	case Warn:
		return "WARN"
	case Error:
		return "ERROR"
	case Fatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

func ParseLevel(level string) Level {
	switch level {
	case "debug":
		return Debug
	case "info":
		return Info
	case "warn":
		return Warn
	case "error":
		return Error
	case "fatal":
		return Fatal
	default:
		return Info
	}
}

type Component string

const (
	Account               Component = "account"
	AccountUsage          Component = "account-usage"
	App                   Component = "app"
	Auth                  Component = "auth"
	AuthUser              Component = "auth-user"
	AuthUserHandler       Component = "auth-user-handler"
	AuthAPIKey            Component = "auth-api-key"
	AuthToken             Component = "auth-token"
	Config                Component = "config"
	DB                    Component = "db"
	GinMiddleware         Component = "gin-middleware"
	HTTP                  Component = "http"
	Instance              Component = "instance"
	MessagingBroker       Component = "messaging-broker"
	MessagingChannel      Component = "messaging-channel"
	MessagingClient       Component = "messaging-client"
	MessagingConnection   Component = "messaging-connection"
	MessagingSubscription Component = "messaging-subscription"
	MessagingPublication  Component = "messaging-publication"
	Migration             Component = "migration"
	NATS                  Component = "nats"
	Project               Component = "project"
	ProjectLog            Component = "project-log"
	ProjectStat           Component = "project-stat"
	ProjectUsage          Component = "project-usage"
	PublicationHandler    Component = "publication-handler"
	Queue                 Component = "queue"
	Redis                 Component = "redis"
	Routes                Component = "routes"
	Seed                  Component = "seed"
	Services              Component = "services"
	Stats                 Component = "stats"
	Tasks                 Component = "tasks"
	User                  Component = "user"
	WebSocket             Component = "websocket"
)

// String implements the Stringer interface
func (c Component) String() string {
	return string(c)
}

const (
	DebugColor = "\033[38;5;111m" // #888CFF
	InfoColor  = "\033[38;5;33m"  // #2166FF
	WarnColor  = "\033[38;5;214m" // #FFB302
	ErrorColor = "\033[38;5;196m" // #FF4C61
	FatalColor = "\033[38;5;161m" // #D90040
)

func GetColorForLevel(level Level) string {
	switch level {
	case Debug:
		return DebugColor
	case Info:
		return InfoColor
	case Warn:
		return WarnColor
	case Error:
		return ErrorColor
	case Fatal:
		return FatalColor
	default:
		return ""
	}
}
