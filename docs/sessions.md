
## Que es una sesion

Una sesion es como un container, donde pueden hacer muchos componentes.

Una sesion puede crear un aegis-data-stream-<id>.sock
y dentro de ese .sock, los datos se streamean por topics.

Session =
- Namespace aislado
- Registry de componentes
- Data stream socket dedicado
- Lifecycle manager