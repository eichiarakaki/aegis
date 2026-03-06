
Flujo completo de un componente desde attach hasta running:aegisctl session attach asd --path ./market_data
HandleSessionAttach llama AttachComponents
Por cada path, se genera un componentID pre-asignado y se crea un Component{State: INIT} en el registry
El path y el ID quedan guardados juntos en session.ComponentEntries
El CTL ya puede ver el componente en INIT con component list/get/describe
aegisctl session start asd
LaunchComponents lee ComponentEntries y lanza cada binario pasando AEGIS_COMPONENT_ID=cmp-xxx via env var
El proceso arranca y conecta al socket del daemon enviando REGISTER con el ID que recibió
HandleComponentConnection ve que el ID ya existe en el registry con State == INIT, y en vez de crear un componente nuevo llama UpdateFromRegister para hidratarlo con el nombre real, version y capabilities
El componente transiciona INIT → REGISTERED → INITIALIZING → READY → CONFIGURED → RUNNING via STATE_UPDATE messages
El heartbeat monitor ignora componentes en INIT, REGISTERED e INITIALIZING — solo empieza a hacer ping cuando el componente ya completó el handshake
Por qué este diseño:

Antes: los componentes no existían en el registry hasta que el proceso se conectaba, lo que hacía imposible inspeccionarlos antes de session start
Ahora: el registry es la fuente de verdad desde el momento del attach. El proceso solo completa la información que ya existe.