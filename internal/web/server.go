package web

import (
	"context"
	"crypto/subtle"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/starfederation/datastar-go/datastar"

	"station/internal/config"
	"station/internal/filesvc"
	"station/internal/models"
	"station/internal/session"
	"station/internal/storage"
	"station/internal/views"
)

type Signals struct {
	Prefix        string `json:"prefix"`
	NewFolderName string `json:"newFolderName"`
	TargetKey     string `json:"targetKey"`
	TargetBatch   string `json:"targetBatch"`
	Refresh       bool   `json:"refresh"`
}

type Server struct {
	cfg      config.Config
	sessions *session.Store
	files    *filesvc.Service
	log      *slog.Logger
}

func New(cfg config.Config, sessions *session.Store, files *filesvc.Service, log *slog.Logger) http.Handler {
	s := &Server{cfg: cfg, sessions: sessions, files: files, log: log}
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer)

	static, err := fs.Sub(assetsFS, "assets")
	if err != nil {
		panic(err)
	}
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(static))))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	r.Get("/login", s.loginPage)
	r.Post("/login", s.login)
	r.Post("/logout", s.logout)

	r.Group(func(r chi.Router) {
		r.Use(s.requireAuth)
		r.Get("/", s.browser)
		r.Get("/settings", s.settings)
		r.Get("/files", s.listFiles)
		r.Post("/folders", s.createFolder)
		r.Post("/files/trash", s.trash)
		r.Post("/files/move", s.move)
		r.Post("/cache/purge", s.purgeCache)
		r.Post("/trash/restore", s.restore)
		r.Post("/trash/purge", s.purgeTrash)
		r.Post("/trash/empty", s.emptyTrash)
		r.Post("/uploads/presign", s.presign)
		r.Post("/uploads/complete", s.complete)
	})
	return r
}

type userKey struct{}

func currentUser(r *http.Request) string {
	user, _ := r.Context().Value(userKey{}).(string)
	return user
}

func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok, err := s.sessions.Username(r.Context(), session.TokenFromRequest(r))
		if err != nil {
			http.Error(w, "session error", http.StatusInternalServerError)
			return
		}
		if !ok {
			if r.Header.Get("Datastar-Request") == "true" {
				sse := datastar.NewSSE(w, r)
				_ = sse.Redirect("/login")
				return
			}
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey{}, user)))
	})
}

func (s *Server) loginPage(w http.ResponseWriter, r *http.Request) {
	if _, ok, _ := s.sessions.Username(r.Context(), session.TokenFromRequest(r)); ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	msg := ""
	switch r.URL.Query().Get("error") {
	case "1":
		msg = "Wrong username or password."
	case "rate":
		msg = "Too many attempts. Try again in a few minutes."
	}
	_ = views.Login(msg).Render(r.Context(), w)
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/login?error=1", http.StatusSeeOther)
		return
	}
	ok, err := s.sessions.AllowLogin(r.Context(), session.ClientIP(r))
	if err != nil || !ok {
		http.Redirect(w, r, "/login?error=rate", http.StatusSeeOther)
		return
	}
	if !s.checkCreds(r.FormValue("username"), r.FormValue("password")) {
		http.Redirect(w, r, "/login?error=1", http.StatusSeeOther)
		return
	}
	token, err := s.sessions.Create(r.Context(), s.cfg.Username)
	if err != nil {
		http.Error(w, "could not start session", http.StatusInternalServerError)
		return
	}
	s.sessions.SetCookie(w, token)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	_ = s.sessions.Delete(r.Context(), session.TokenFromRequest(r))
	s.sessions.ClearCookie(w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *Server) checkCreds(user, pass string) bool {
	uOK := subtle.ConstantTimeCompare([]byte(user), []byte(s.cfg.Username)) == 1
	pOK := subtle.ConstantTimeCompare([]byte(pass), []byte(s.cfg.Password)) == 1
	return uOK && pOK
}

func (s *Server) readSignals(r *http.Request) (Signals, error) {
	var sig Signals
	if err := datastar.ReadSignals(r, &sig); err != nil {
		return sig, err
	}
	return sig, nil
}

func (s *Server) flash(sse *datastar.ServerSentEventGenerator, kind, msg string) {
	_ = sse.MarshalAndPatchSignals(map[string]any{
		"_flash":     msg,
		"_flashKind": kind,
		"_busy":      false,
		"refresh":    false,
	})
}

func (s *Server) patchListing(w http.ResponseWriter, r *http.Request, listing models.Listing, msg string) {
	sse := datastar.NewSSE(w, r)
	_ = sse.PatchElementTempl(views.FilePanel(listing), datastar.WithSelector("#file-panel"), datastar.WithModeReplace())
	_ = sse.PatchElementTempl(views.Crumbs(listing.Prefix), datastar.WithSelector("#crumbs"), datastar.WithModeReplace())
	_ = sse.PatchElementTempl(views.MetaBar(listing), datastar.WithSelector("#meta-bar"), datastar.WithModeReplace())
	_ = sse.MarshalAndPatchSignals(map[string]any{
		"prefix":         listing.Prefix,
		"refresh":        false,
		"newFolderName":  "",
		"_showNewFolder": false,
		"_showDelete":    false,
		"_busy":          false,
		"_flash":         msg,
		"_flashKind":     "ok",
	})
	u := *r.URL
	u.Path = "/"
	if listing.Prefix != "" {
		q := url.Values{}
		q.Set("prefix", listing.Prefix)
		u.RawQuery = q.Encode()
	} else {
		u.RawQuery = ""
	}
	_ = sse.ReplaceURL(u)
}

func (s *Server) patchTrash(w http.ResponseWriter, r *http.Request, items []models.TrashItem, msg string) {
	sse := datastar.NewSSE(w, r)
	_ = sse.PatchElementTempl(views.TrashPanel(items), datastar.WithSelector("#trash-panel"), datastar.WithModeReplace())
	_ = sse.MarshalAndPatchSignals(map[string]any{
		"targetBatch":     "",
		"_showDelete":     false,
		"_showEmptyTrash": false,
		"_flash":          msg,
		"_flashKind":      "ok",
	})
}

func publicError(err error) string {
	switch {
	case errors.Is(err, storage.ErrInvalidPath):
		return "That path is not allowed."
	case errors.Is(err, storage.ErrReserved):
		return "That name is reserved for trash."
	default:
		return "Something went wrong."
	}
}

