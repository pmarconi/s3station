package web

import (
	"encoding/json"
	"net/http"

	"github.com/starfederation/datastar-go/datastar"

	"station/internal/views"
)

func (s *Server) browser(w http.ResponseWriter, r *http.Request) {
	prefix := r.URL.Query().Get("prefix")
	listing, err := s.files.List(r.Context(), prefix, false)
	if err != nil {
		s.log.Error("list", "err", err)
		http.Error(w, publicError(err), http.StatusBadRequest)
		return
	}
	_ = views.Browser(currentUser(r), listing).Render(r.Context(), w)
}

func (s *Server) settings(w http.ResponseWriter, r *http.Request) {
	items, err := s.files.ListTrash(r.Context())
	if err != nil {
		s.log.Error("list trash", "err", err)
		http.Error(w, publicError(err), http.StatusBadGateway)
		return
	}
	_ = views.Settings(currentUser(r), items).Render(r.Context(), w)
}

func (s *Server) listFiles(w http.ResponseWriter, r *http.Request) {
	sig, err := s.readSignals(r)
	if err != nil {
		http.Error(w, "bad signals", http.StatusBadRequest)
		return
	}
	listing, err := s.files.List(r.Context(), sig.Prefix, sig.Refresh)
	if err != nil {
		s.log.Error("list", "err", err)
		s.flash(datastar.NewSSE(w, r), "bad", publicError(err))
		return
	}
	msg := ""
	if sig.Refresh {
		msg = "Folder reloaded from the bucket."
	}
	s.patchListing(w, r, listing, msg)
}

func (s *Server) createFolder(w http.ResponseWriter, r *http.Request) {
	sig, err := s.readSignals(r)
	if err != nil {
		http.Error(w, "bad signals", http.StatusBadRequest)
		return
	}
	if err := s.files.CreateFolder(r.Context(), sig.Prefix, sig.NewFolderName); err != nil {
		s.log.Error("mkdir", "err", err)
		s.flash(datastar.NewSSE(w, r), "bad", publicError(err))
		return
	}
	listing, err := s.files.List(r.Context(), sig.Prefix, false)
	if err != nil {
		s.flash(datastar.NewSSE(w, r), "bad", publicError(err))
		return
	}
	s.patchListing(w, r, listing, "Folder created.")
}

func (s *Server) move(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Key        string `json:"key"`
		DestPrefix string `json:"destPrefix"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if err := s.files.Move(r.Context(), req.Key, req.DestPrefix); err != nil {
		s.log.Error("move", "err", err)
		http.Error(w, publicError(err), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) trash(w http.ResponseWriter, r *http.Request) {
	sig, err := s.readSignals(r)
	if err != nil {
		http.Error(w, "bad signals", http.StatusBadRequest)
		return
	}
	if err := s.files.Trash(r.Context(), sig.TargetKey); err != nil {
		s.log.Error("trash", "err", err)
		s.flash(datastar.NewSSE(w, r), "bad", publicError(err))
		return
	}
	listing, err := s.files.List(r.Context(), sig.Prefix, false)
	if err != nil {
		s.flash(datastar.NewSSE(w, r), "bad", publicError(err))
		return
	}
	s.patchListing(w, r, listing, "Moved to trash. Permanent delete is in Settings.")
}

func (s *Server) purgeCache(w http.ResponseWriter, r *http.Request) {
	if err := s.files.ClearCache(r.Context(), ""); err != nil {
		s.log.Error("purge cache", "err", err)
		s.flash(datastar.NewSSE(w, r), "bad", publicError(err))
		return
	}
	s.flash(datastar.NewSSE(w, r), "ok", "All folder listings were dropped from Postgres.")
}

func (s *Server) restore(w http.ResponseWriter, r *http.Request) {
	sig, err := s.readSignals(r)
	if err != nil {
		http.Error(w, "bad signals", http.StatusBadRequest)
		return
	}
	if err := s.files.Restore(r.Context(), sig.TargetBatch); err != nil {
		s.log.Error("restore", "err", err)
		s.flash(datastar.NewSSE(w, r), "bad", publicError(err))
		return
	}
	items, err := s.files.ListTrash(r.Context())
	if err != nil {
		s.flash(datastar.NewSSE(w, r), "bad", publicError(err))
		return
	}
	s.patchTrash(w, r, items, "Restored to the original path.")
}

func (s *Server) purgeTrash(w http.ResponseWriter, r *http.Request) {
	sig, err := s.readSignals(r)
	if err != nil {
		http.Error(w, "bad signals", http.StatusBadRequest)
		return
	}
	if err := s.files.PurgeTrash(r.Context(), sig.TargetBatch); err != nil {
		s.log.Error("purge trash", "err", err)
		s.flash(datastar.NewSSE(w, r), "bad", publicError(err))
		return
	}
	items, err := s.files.ListTrash(r.Context())
	if err != nil {
		s.flash(datastar.NewSSE(w, r), "bad", publicError(err))
		return
	}
	s.patchTrash(w, r, items, "Permanently deleted from the bucket.")
}

func (s *Server) emptyTrash(w http.ResponseWriter, r *http.Request) {
	if err := s.files.EmptyTrash(r.Context()); err != nil {
		s.log.Error("empty trash", "err", err)
		s.flash(datastar.NewSSE(w, r), "bad", publicError(err))
		return
	}
	s.patchTrash(w, r, nil, "Trash emptied. Those objects are gone from the bucket.")
}

func (s *Server) presign(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		ContentType string `json:"contentType"`
		Size        int64  `json:"size"`
		Prefix      string `json:"prefix"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	spec, err := s.files.PresignUpload(r.Context(), req.Prefix, req.Name, req.ContentType, req.Size)
	if err != nil {
		s.log.Error("presign", "err", err)
		http.Error(w, publicError(err), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(spec)
}

func (s *Server) complete(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Keys   []string `json:"keys"`
		Prefix string   `json:"prefix"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if err := s.files.CompleteUploads(r.Context(), req.Keys); err != nil {
		s.log.Error("complete", "err", err)
		http.Error(w, publicError(err), http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
