```
cmd/
 ├─ aegis          # daemon principal, distribuye datos
 ├─ aegisd         # CLI para controlar daemon
 └─ aegis-fetcher  # fetch histórico y limpieza/preparación

internals/
 ├─ core/          # modelos de datos, entidades, tipos puros
 ├─ services/      # lógica de negocio (validaciones, healthchecks, adaptaciones)
 └─ handlers/      # wrappers para sockets/CLI (para aegis y aegis-fetcher si hace algo de net)

config/
 ├─ aegis.yaml     # rutas a datos, parámetros generales
 └─ globals.yaml   # rutas, flags comunes
```