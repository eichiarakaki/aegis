
## What is a session
Una sesion es como un container, donde pueden hacer muchos componentes.

Una sesion puede crear un aegis-data-stream-<id>.sock
y dentro de ese .sock, los datos se streamean por topics.

Session =
- Namespace aislado
- Registry de componentes
- Data stream socket dedicado
- Lifecycle manager

---

## Objectives
La estructura debe ser capaz de:
- correr el 'contenedor' de componentes (iniciar el envio de flujos de datos)
- poder ser controlado por aegis-cli session (stop/start/attach)

Estructura para lograr eso:

```go
type StatusType int

const (
	SessionCreated StatusType = iota
	SessionStarting
	SessionRunning
	SessionStopping
	SessionStopped
	SessionFinished
)

type Session struct {
	ID     string
	Name   string
	Mode   string // realtime | historical
	Status StatusType

	// Why map instead of slices? O(1) lookups by component name, easier to manage dynamic additions/removals
	Components map[string]*Component

	CreatedAt time.Time
	StartedAt *time.Time
	StoppedAt *time.Time

	mu sync.RWMutex
}

```