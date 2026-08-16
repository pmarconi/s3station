# Datastar in this app

This app uses [Datastar](https://data-star.dev) for in-page UI: folder navigation, modals, flashes, and trash/settings actions. The script is local: `/static/js/datastar.js` (embedded). Official reference: `docs/context/datastar-docs.md` and https://data-star.dev/docs

Vanilla JS (`upload.js`, `gallery.js`, `nav.js`) handles S3 uploads, drag-and-drop, the image gallery, and browser back/forward (`popstate` → `station-navigate`). Those are not Datastar.

## Page loads vs patches

Full HTML (templ `Layout`): `GET /files/`, `GET /files/{folder}/…`, `GET /trash`, `GET /settings`, `GET /login`. `GET /` redirects to `/files/`.

Datastar SSE patches (same page, morph/replace fragments):

| Client | Server | Patches |
| --- | --- | --- |
| `@get('/files')` | `listFiles` (Datastar `GET /files`) | `#file-panel`, `#crumbs`, `#meta-bar` + signals + folder URL (`pushState` when `$nav`) |
| `@post('/folders')` | `createFolder` | same as listing |
| `@post('/folders/unlock'\|'protect'\|'unprotect')` | lock handlers | same as listing |
| `@post('/files/trash')` | `trash` | listing |
| `@post('/cache/purge')` | `purgeCache` | flash only |
| `@post('/trash/restore'\|'purge'\|'empty')` | trash handlers | `#trash-panel` + signals |
| `@post('/thumbs/purge')` | `purgeThumbs` | flash |

Depot objects are stored under `files/` in the bucket (`FILES_PREFIX`). The browser URL is `/files/` plus that relative path (`/files/photos/italy/`). Other app pages stay at the root (`/trash`, `/settings`, …). A non-Datastar `GET /files` redirects to `/files/`.

Do **not** use `@get` / `@post` for binary or JSON APIs:

- `GET /folders/archive` — zip stream (`<a href>`)
- `GET /thumbs/*` — image bytes
- `POST /uploads/presign`, `POST /uploads/complete`, `POST /files/move` — `fetch` + JSON from `upload.js`

If `Datastar-Request: true` and the session is dead, redirect with `sse.Redirect("/login")`, not `http.Redirect`.

## Signals

Initialized once on the page root with `data-signals={ InitialSignals(...) }` in `internal/views/helpers.go`.

Backend reads a subset via `datastar.ReadSignals` into `web.Signals`: `prefix`, `newFolderName`, `targetKey`, `targetBatch`, `folderPassword`, `refresh`, `nav`. `$nav` is true only for in-page folder clicks (history `pushState`); reloads and mutations `replaceState` the same path.

The rest is frontend-only. Names starting with `_` are UI state (`_busy`, `_flash`, `_showDelete`, `_locked`, …).

Keep new signals in `InitialSignals` **and** reset them in `patchListing` / `patchTrash` / `flash` when the action should close a modal or clear busy/flash.

## Attributes we use

- `data-on:click` / `data-on:submit` — expressions; `@get` / `@post` send **all** current signals
- `data-on:event__window` — listen on `window` (Escape, `station-uploaded`, …)
- `data-bind:name` — two-way input (`query`, `newFolderName`, `folderPassword`)
- `data-show="expr"` — hide/show (also set `style="display: none"` for first paint)
- `data-text="$targetName"` — text content from a signal
- `data-class:btn-active="$view === 'grid'"`
- `data-attr:disabled="$_busy \|\| $_locked"` and `data-attr:href="… + $prefix"`
- `data-attr:open="$_showNewFolder"` on DaisyUI `<dialog class="modal">`
- `data-indicator:_busy` — sets `$_busy` for the duration of the request

Expressions that need a Go value (object key, name) are built in helpers (`NavigateExpr`, `TrashExpr`, …) with `jsString` so quotes are safe:

```
$_busy = true; $prefix = "photos/"; $refresh = false; $nav = true; @get('/files')
```

## Server patches

```go
sse := datastar.NewSSE(w, r)
_ = sse.PatchElementTempl(views.FilePanel(listing),
    datastar.WithSelector("#file-panel"),
    datastar.WithModeReplace())
_ = sse.MarshalAndPatchSignals(map[string]any{ /* … */ })
applyListingURL(sse, listing.Prefix, push) // /files/photos/italy/ via pushState or replaceState
```

The patched root must keep a stable `id` (`#file-panel`, `#trash-panel`, `#crumbs`, `#meta-bar`). If the control that should update lives **outside** that id (for example Empty trash), put it inside the panel or patch a second selector.

`WithModeReplace` replaces the node. Morph is the Datastar default for HTML responses without SSE; the Go handlers here always use SSE + replace.

## Vanilla JS bridge

`upload.js` / `gallery.js` dispatch `CustomEvent`s. Datastar listens on the drawer root:

```
data-on:station-uploaded__window="$_busy = false; $_flash = evt.detail.message; $_flashKind = 'ok'; @get('/files')"
```

After a successful upload or internal move, refresh the listing with `@get('/files')`. Do not try to patch the grid from those scripts.

## Templ pitfalls

- Do **not** use `else if` on HTML **attributes**. templ emits a literal ` else` into the markup. Use two separate `if` blocks.
- `data-on:click={ Expr(e) }` is a Go expression that returns JS. A raw `@post('/x')` cannot pass a per-row key unless that key is already a signal (`$targetKey`).
- After `.templ` or `assets/js` changes: `templ generate` and a Go rebuild (`go:embed`).
- CSS is a local Tailwind/DaisyUI build (`task css`). CDN Tailwind will not see server-rendered classes.

## New UI action checklist

1. Add any new signal to `InitialSignals`.
2. Wire `data-on:*` / `data-bind` in templ.
3. If the backend needs values, add fields to `web.Signals`.
4. Handler: `readSignals` → service call → `patchListing` or `patchTrash` (or `flash` on error).
5. Reset `_show*` / `_busy` / targets in that patch.
6. If the action downloads a file or talks to S3 from the browser, use `<a>` or `fetch`, not `@get`.
