# co-change

[English](README.md) · **Español**

![CI](https://github.com/nicovegasr/co-change/actions/workflows/ci.yml/badge.svg)

Una GitHub Action que te dice, en cada pull request, **qué archivos suelen cambiar con los que has cambiado y se han quedado sin tocar**, con los commits que lo demuestran.

Cambiaste la API pero no su cliente. Tocaste la migración pero no el modelo. La historia de tu repo ya sabe que esos archivos viajan juntos; co-change la lee y deja un único comentario en el PR:

> **co-change** · files that usually change with this PR but are untouched:
>
> | File | Usually changes with | Co-change | Evidence |
> |---|---|---|---|
> | `CHANGELOG.md` | `main.go` | 90% (9 commits) | 614c265, 61d6cfa, ef02702 |
> | `internal/audit/audit_test.go` | `internal/audit/audit.go` | 75% (3 commits) | 8b6387b, d85798c, d0cc3f0 |
> | `go.sum` | `go.mod` | 75% (3 commits) | ee98508, a046d1a, cdac97a |
>
> <sub>Historical co-change before 521568b, not a dependency analysis.</sub>

Sin IA adivinando, sin configuración y sin que el código salga del runner: es historia de git pura, y cada fila enlaza a commits que puedes abrir.

---

## Úsala

```yaml
# .github/workflows/co-change.yml
name: co-change
on: pull_request

permissions:
  contents: read
  pull-requests: write

jobs:
  review:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
        with:
          fetch-depth: 0        # co-change necesita la historia completa
      - uses: nicovegasr/co-change@main
```

| Input | Por defecto | Qué es |
|---|---|---|
| `base` | la rama base del PR | Rama o commit al que apunta el PR |
| `min-support` | `3` | Mínimo de commits en que ambos archivos cambiaron juntos |
| `min-ratio` | `0.3` | Mínima proporción de commits del archivo cambiado que también tocaron la sugerencia |
| `limit` | `5` | Máximo de sugerencias |
| `token` | `github.token` | Token con permiso para comentar |

La revisión también se escribe en el resumen del job. Los PRs desde forks reciben un token de solo lectura, así que solo tienen el resumen.

---

## Cómo funciona

1. Indexa la historia del repo dentro del runner (`.git/co-change/`).
2. Toma los archivos que el PR cambia desde que se separó (el merge base).
3. Para cada uno, busca los archivos que cambiaron con él **en la historia anterior al merge base**, así los commits del propio PR nunca confirman sus sugerencias.
4. Se queda con los archivos sin tocar que siguen existiendo y pasan el filtro de ruido, ordenados dando más peso a la historia reciente (vida media de 90 días).
5. Crea un comentario y lo actualiza en cada push. Si no encuentra nada no publica nada, o actualiza su comentario anterior para decirlo.

Es **informativa**: nunca falla el check ni bloquea un merge.

### Quita ruido — `.cochangeignore`

Lockfiles, changelogs o código generado pueden co-cambiar con todo. Añade un `.cochangeignore` en la raíz con un glob o directorio por línea:

```gitignore
*.lock
package-lock.json
CHANGELOG.md
dist/
```

---

## ¿Funciona de verdad?

`co-change evaluate` esconde commits del pasado y pregunta, usando **solo la historia anterior** a cada uno, qué otros archivos cambiaron. Compara con dos referencias: contar sin filtros qué cambió junto (`unfiltered`) y sugerir los archivos más tocados del repo (`frequency`).

Precisión de las 5 primeras sugerencias en los commits más recientes de cada repo (los parámetros se ajustaron con commits más antiguos):

| Repo | co-change | unfiltered | frequency |
|---|---:|---:|---:|
| fzf (Go) | **30.6%** | 23.9% | 18.0% |
| Django (Python) | **13.1%** | 11.5% | 2.3% |
| OkHttp (Kotlin/Java) | 20.0% | **21.9%** | 1.9% |
| Flask (Python) | 18.6% | **19.3%** | 2.8% |
| ripgrep (Rust) | 13.3% | 13.3% | 4.2% |
| Astro (TypeScript) | 7.9% | 7.9% | 1.7% |
| Blog personal (Astro) | 22.3% | **23.3%** | 11.6% |
| media de 7 repos | **18.0%** | 17.3% | 6.1% |

Lectura honesta: fijarse en **lo que cambia con tu archivo** acierta de media unas 3 veces más que fijarse en lo que más cambia en general (desde 1,7× en fzf hasta 10× en OkHttp). Los filtros aportan poco sobre el co-cambio simple en estas muestras pequeñas (24–52 casos por repo); en muestras más grandes la ventaja es de unos 2,5 puntos. Estos números miden el ranking, no el comentario del PR: sus umbrales (`min-support 3`, `min-ratio 0.3`) son una primera elección para que no haga ruido, aún sin validar. Metodología en [docs/research/file-coupling-benchmarks.md](docs/research/file-coupling-benchmarks.md).

---

## Úsalo en local

Requiere Go 1.26+ y git.

```bash
git clone https://github.com/nicovegasr/co-change
cd co-change
make install                                   # instala co-change en ~/go/bin
```

Después, dentro de cualquier repo:

```bash
co-change review --base main                   # el comentario del PR de tu rama
co-change review --base main --json            # lo mismo, para otras herramientas
co-change coupled internal/ui/model.go         # qué cambia con un archivo, con evidencia
co-change coupled src/app.ts --before v1.0     # solo la historia anterior a un commit
co-change evaluate --prefix src/ --ext .ts     # mídelo en tu propio repo
```

`co-change` sin argumentos abre una herramienta de commits en la terminal (Conventional Commits en una línea, con los archivos relacionados mientras marcas); mira [docs/tui.es.md](docs/tui.es.md).

---

## Límites

- Un archivo es una **ruta**: los renombrados no unen su historia.
- Solo la historia alcanzable desde `HEAD`; los clones superficiales se rechazan.
- La Action compila co-change desde el código e indexa desde cero en cada ejecución (menos de un segundo en este repo; los repos grandes tardarán más hasta que llegue la caché).
- Co-cambio histórico no es dependencia ni predicción de bugs: es una pista.

---

## Documentación

| Documento | Contenido |
|---|---|
| [docs/adr/](docs/adr/) | Las decisiones, una por archivo (ADR-0019: el enfoque de CI) |
| [docs/research/file-coupling-benchmarks.md](docs/research/file-coupling-benchmarks.md) | Benchmarks y evaluaciones en repos reales |
| [docs/architecture.md](docs/architecture.md) | Paquetes, flujo y reglas de diseño |
| [docs/tui.es.md](docs/tui.es.md) | El TUI de commits |
| [docs/ROADMAP.md](docs/ROADMAP.md) | Qué viene después |
| [docs/contributing.md](docs/contributing.md) | Cómo trabajar en el repo |

---

## Desarrollo

```bash
make check   # formato, vet, lint y tests (pre-commit)
make ci      # todo lo anterior + cobertura ≥ 80% + integración con git real
make bench   # microbenchmarks del índice
```
