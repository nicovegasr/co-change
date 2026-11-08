# El TUI de commits

[English](tui.md) · **Español**

`co-change` sin argumentos abre una herramienta de terminal para escribir [Conventional Commits](https://www.conventionalcommits.org/) en una línea. Mientras marcas archivos, te enseña qué otros suelen cambiar con ellos. Es anterior al enfoque de CI (ADR-0019): sigue funcionando pero no recibe funcionalidades nuevas.

![TUI de co-change: marca un archivo y aparecen sus relacionados](../assets/tui.gif)

## El TUI — commitear en una línea

| Acción | Teclado | Ratón |
|---|---|---|
| Marcar / desmarcar un archivo | `espacio` | click en el archivo |
| Moverse por la lista | `j` / `k` o flechas | — |
| Elegir el tipo (`feat`, `fix`, …) | `tab` + la inicial | click en `[ feat ▾ ]` |
| Escribir el mensaje y commitear | escribir + `enter` | click en el mensaje |
| Volver atrás / salir | `esc` · `q` · `ctrl+c` | — |

Debajo de la línea de commit verás el subject exacto que se va a crear (`→ feat: … .`), con el punto final ya puesto. Hace `git add` **solo de los archivos marcados**, nunca `git add .`.

La sesión no se cierra al commitear: la lista y el git log se recargan cada 2 s, así que lo que cambies en el editor, los `git add` o los commits que hagas desde otra terminal aparecen solos. Tus marcas se mantienen.

### Relacionados

Al marcar archivos, la columna derecha muestra los cinco que más suelen cambiar con ellos:

- `+` en amarillo: está **modificado y sin marcar**. Probablemente se te olvida.
- `·` en gris: suele cambiar con lo marcado, pero no lo has tocado.
- `↔` en la lista de archivos: el mismo aviso, junto al archivo.

El porcentaje es la proporción de commits del archivo marcado que también tocaron ese otro. El índice se crea solo la primera vez (en un repo grande tarda unos segundos, en segundo plano) y se mantiene al día tras cada commit, checkout o rebase. Si algo falla, el panel se queda en una línea gris y puedes commitear igual.

Con menos de 80 columnas de ancho, la columna derecha se oculta para no estorbar al flujo mínimo.

---

## Comandos dentro del TUI — `/`

Escribe `/` con el mensaje vacío (o haz click en `/ comandos`) para abrir la paleta, teclea unas letras y pulsa `enter`. `esc` te devuelve a la vista de commit. Puedes lanzar co-change desde cualquier subcarpeta del repo.

### Revisa tus cambios — `/changes`

![/changes: los archivos cambiados y el diff del seleccionado](../assets/changes.png)

A la izquierda los archivos cambiados; a la derecha el archivo entero con lo añadido en verde y lo borrado en rojo. Pensado para revisar lo que ha cambiado un agente o una sesión larga sin abrir un IDE.

| Acción | Teclado | Ratón |
|---|---|---|
| Filtrar archivos por ruta | `/` y escribir | click en el buscador |
| Cambiar de archivo / scroll del diff | `j` / `k` (`tab` cambia de panel) | click · rueda |
| Cambio siguiente / anterior | `n` / `p` | las marcas de color de la barra |
| Media página | `ctrl+d` / `ctrl+u` | arrastrar la barra |
| Marcar el archivo para el commit | `espacio` | — |
| Editar el archivo en ese cambio | `e` | — |

`e` le pasa la terminal a tu editor, abierto en el cambio que estás viendo, y al salir co-change recarga el diff. El editor es `$VISUAL` o `$EDITOR` (p. ej. `export EDITOR=vim` en tu `.zshrc`); si no hay ninguno, `micro`, `nano` o `vi`. co-change nunca edita archivos por sí mismo.

### Tus commits por tipo — `/stats`

![/stats: commits por tipo y actividad diaria](../assets/stats.png)

Los commits de la rama actual por tipo y su actividad: por día hasta 30 días, por semana a partir de ahí. `d` o un click en el título cambia entre 7 · 30 · 90 · 365 días. Los commits sin prefijo `tipo:` cuentan como `sin tipo`; el culpable habitual es el título de squash de GitHub (`Feat/audit (#13)`), así que ponle a tus PR un título convencional.

### Los archivos que más cambian — `/audit`

![/audit: archivos ordenados por número de commits](../assets/audit.png)

Archivos ordenados por **cuántos commits los tocaron** (los hotspots de Adam Tornhill); las líneas cambiadas son solo contexto. Los que pasan el umbral de `/settings` salen como `refactor?` y los que se acercan, `vigilar`. `d` cambia la ventana. Los lockfiles y el build se ignoran; añade tus patrones en un `.auditignore` en la raíz, con el mismo formato que `.cochangeignore`.

### Apaga lo que no uses — `/settings`

El panel de git log, el de relacionados, la captura del ratón (apagada = selección y scroll nativos de la terminal), el alcance del log (rama actual o todas), cada cuánto se recarga y los valores por defecto de `/audit`. Se aplica al momento y se guarda en `~/.config/co-change/settings.toml`.
