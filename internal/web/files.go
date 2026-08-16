package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"

	"station/internal/filesvc"
	"station/internal/locks"
	"station/internal/models"
	"station/internal/session"
	"station/internal/storage"
	"station/internal/views"
)

func applyListingURL(sse *datastar.ServerSentEventGenerator, prefix string, push bool) {
	path := storage.PublicFolderPath(prefix)
	u := url.URL{Path: path}
	if !push {
		_ = sse.ReplaceURL(u)
		return
	}
	_ = sse.ExecuteScript(fmt.Sprintf(
		`if(location.pathname!==%s){history.pushState({prefix:%s},"",%s)}`,
		strconv.Quote(path),
		strconv.Quote(prefix),
		strconv.Quote(path),
	))
}

func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	if q := r.URL.Query().Get("prefix"); q != "" {
		http.Redirect(w, r, storage.PublicFolderPath(q), http.StatusMovedPermanently)
		return
	}
	http.Redirect(w, r, storage.PublicFolderPath(""), http.StatusSeeOther)
}

func (s *Server) filesRoot(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Datastar-Request") == "true" {
		s.listFiles(w, r)
		return
	}
	if q := r.URL.Query().Get("prefix"); q != "" {
		http.Redirect(w, r, storage.PublicFolderPath(q), http.StatusMovedPermanently)
		return
	}
	http.Redirect(w, r, storage.PublicFolderPath(""), http.StatusMovedPermanently)
}

func (s *Server) browser(w http.ResponseWriter, r *http.Request) {
	if q := r.URL.Query().Get("prefix"); q != "" {
		http.Redirect(w, r, storage.PublicFolderPath(q), http.StatusMovedPermanently)
		return
	}
	prefix := ""
	if p, ok := storage.PrefixFromRequestPath(r.URL.Path); ok {
		prefix = p
	} else {
		http.NotFound(w, r)
		return
	}
	if prefix != "" && !strings.HasSuffix(r.URL.Path, "/") {
		http.Redirect(w, r, storage.PublicFolderPath(prefix), http.StatusMovedPermanently)
		return
	}
	listing, err := s.files.List(r.Context(), prefix, false)
	if err != nil {
		s.log.Error("list", "err", err)
		http.Error(w, publicError(err), http.StatusBadRequest)
		return
	}
	_ = views.Browser(currentUser(r), listing, s.uiView(r)).Render(r.Context(), w)
}

func (s *Server) settings(w http.ResponseWriter, r *http.Request) {
	_ = views.Settings(currentUser(r)).Render(r.Context(), w)
}

func (s *Server) trashPage(w http.ResponseWriter, r *http.Request) {
	items, err := s.files.ListTrash(r.Context())
	if err != nil {
		s.log.Error("list trash", "err", err)
		http.Error(w, publicError(err), http.StatusBadGateway)
		return
	}
	_ = views.Trash(currentUser(r), items).Render(r.Context(), w)
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
	s.patchListing(w, r, listing, msg, sig.Prefix, sig.Nav && !sig.Refresh)
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
	s.patchListing(w, r, listing, "Folder created.", sig.Prefix, false)
}

func (s *Server) folderArchive(w http.ResponseWriter, r *http.Request) {
	arch, err := s.files.PrepareFolderArchive(r.Context(), r.URL.Query().Get("prefix"))
	if err != nil {
		s.log.Error("folder archive", "err", err)
		status := http.StatusBadRequest
		if errors.Is(err, locks.ErrLocked) {
			status = http.StatusForbidden
		}
		http.Error(w, publicError(err), status)
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", filesvc.AttachmentDisposition(arch.Filename))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, no-store")
	if err := s.files.WriteFolderArchive(r.Context(), arch, w); err != nil {
		s.log.Error("write folder archive", "prefix", arch.Prefix, "err", err)
	}
}

func (s *Server) protectFolder(w http.ResponseWriter, r *http.Request) {
	sig, err := s.readSignals(r)
	if err != nil {
		http.Error(w, "bad signals", http.StatusBadRequest)
		return
	}
	if err := s.files.ProtectFolder(r.Context(), sig.Prefix, sig.FolderPassword); err != nil {
		s.log.Error("protect folder", "err", err)
		s.flash(datastar.NewSSE(w, r), "bad", publicError(err))
		return
	}
	prefix, _ := normalizeSignalPrefix(sig.Prefix)
	if err := s.sessions.UnlockFolder(r.Context(), session.TokenFromRequest(r), prefix); err != nil {
		s.flash(datastar.NewSSE(w, r), "bad", publicError(err))
		return
	}
	listing, err := s.files.List(s.withUnlock(r, prefix).Context(), sig.Prefix, false)
	if err != nil {
		s.flash(datastar.NewSSE(w, r), "bad", publicError(err))
		return
	}
	s.patchListing(w, r, listing, "Folder is now password protected in the UI.", sig.Prefix, false)
}

func (s *Server) unprotectFolder(w http.ResponseWriter, r *http.Request) {
	sig, err := s.readSignals(r)
	if err != nil {
		http.Error(w, "bad signals", http.StatusBadRequest)
		return
	}
	if err := s.files.UnprotectFolder(r.Context(), sig.Prefix, sig.FolderPassword); err != nil {
		s.log.Error("unprotect folder", "err", err)
		s.flash(datastar.NewSSE(w, r), "bad", publicError(err))
		return
	}
	listing, err := s.files.List(r.Context(), sig.Prefix, false)
	if err != nil {
		s.flash(datastar.NewSSE(w, r), "bad", publicError(err))
		return
	}
	s.patchListing(w, r, listing, "Password removed. Anyone signed in can open this folder.", sig.Prefix, false)
}

func (s *Server) unlockFolder(w http.ResponseWriter, r *http.Request) {
	sig, err := s.readSignals(r)
	if err != nil {
		http.Error(w, "bad signals", http.StatusBadRequest)
		return
	}
	ok, err := s.sessions.AllowUnlock(r.Context(), session.ClientIP(r))
	if err != nil || !ok {
		s.flash(datastar.NewSSE(w, r), "bad", "Too many unlock attempts. Try again in a few minutes.")
		return
	}
	target := sig.TargetKey
	if target == "" {
		target = sig.Prefix
	}
	unlockedPrefix, err := s.files.UnlockFolder(r.Context(), target, sig.FolderPassword)
	if err != nil {
		s.log.Error("unlock folder", "err", err)
		s.flash(datastar.NewSSE(w, r), "bad", publicError(err))
		return
	}
	if err := s.sessions.UnlockFolder(r.Context(), session.TokenFromRequest(r), unlockedPrefix); err != nil {
		s.flash(datastar.NewSSE(w, r), "bad", publicError(err))
		return
	}
	listing, err := s.files.List(s.withUnlock(r, unlockedPrefix).Context(), target, false)
	if err != nil {
		s.flash(datastar.NewSSE(w, r), "bad", publicError(err))
		return
	}
	s.patchListing(w, r, listing, "Folder unlocked for this session.", sig.Prefix, false)
}

func (s *Server) withUnlock(r *http.Request, prefix string) *http.Request {
	current := append([]string{}, locks.Unlocked(r.Context())...)
	current = append(current, prefix)
	return r.WithContext(locks.WithUnlocked(r.Context(), current))
}

func normalizeSignalPrefix(prefix string) (string, error) {
	return storage.NormalizePrefix(prefix)
}

func (s *Server) folderPicker(w http.ResponseWriter, r *http.Request) {
	sig, err := s.readSignals(r)
	if err != nil {
		http.Error(w, "bad signals", http.StatusBadRequest)
		return
	}
	listing, err := s.files.ListFolders(r.Context(), sig.DestPrefix)
	if err != nil {
		s.log.Error("folder picker", "err", err)
		s.flash(datastar.NewSSE(w, r), "bad", publicError(err))
		return
	}
	sse := datastar.NewSSE(w, r)
	_ = sse.PatchElementTempl(views.MovePicker(listing, sig.TargetKey), datastar.WithSelector("#move-picker"), datastar.WithModeReplace())
}

func (s *Server) move(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Datastar-Request") == "true" {
		s.moveFromSignals(w, r)
		return
	}
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

func (s *Server) moveFromSignals(w http.ResponseWriter, r *http.Request) {
	sig, err := s.readSignals(r)
	if err != nil {
		http.Error(w, "bad signals", http.StatusBadRequest)
		return
	}
	if err := s.files.Move(r.Context(), sig.TargetKey, sig.DestPrefix); err != nil {
		s.log.Error("move", "err", err)
		s.flash(datastar.NewSSE(w, r), "bad", publicError(err))
		return
	}
	listing, err := s.files.List(r.Context(), sig.Prefix, false)
	if err != nil {
		s.flash(datastar.NewSSE(w, r), "bad", publicError(err))
		return
	}
	s.patchListing(w, r, listing, "Moved.", sig.Prefix, false)
}

func (s *Server) rename(w http.ResponseWriter, r *http.Request) {
	sig, err := s.readSignals(r)
	if err != nil {
		http.Error(w, "bad signals", http.StatusBadRequest)
		return
	}
	if err := s.files.Rename(r.Context(), sig.TargetKey, sig.RenameTo); err != nil {
		s.log.Error("rename", "err", err)
		s.flash(datastar.NewSSE(w, r), "bad", publicError(err))
		return
	}
	listing, err := s.files.List(r.Context(), sig.Prefix, false)
	if err != nil {
		s.flash(datastar.NewSSE(w, r), "bad", publicError(err))
		return
	}
	s.patchListing(w, r, listing, "Renamed.", sig.Prefix, false)
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
	s.patchListing(w, r, listing, "Moved to trash. Permanent delete is in Settings.", sig.Prefix, false)
}

func (s *Server) purgeCache(w http.ResponseWriter, r *http.Request) {
	if err := s.files.ClearCache(r.Context(), ""); err != nil {
		s.log.Error("purge cache", "err", err)
		s.flash(datastar.NewSSE(w, r), "bad", publicError(err))
		return
	}
	s.flash(datastar.NewSSE(w, r), "ok", "All folder listings were dropped from Postgres.")
}

func (s *Server) generateThumb(w http.ResponseWriter, r *http.Request) {
	sig, err := s.readSignals(r)
	if err != nil {
		http.Error(w, "bad signals", http.StatusBadRequest)
		return
	}
	if err := s.files.GenerateThumb(r.Context(), sig.TargetKey); err != nil {
		s.log.Error("generate thumb", "err", err)
		s.flash(datastar.NewSSE(w, r), "bad", publicError(err))
		return
	}
	listing, err := s.files.List(r.Context(), sig.Prefix, false)
	if err != nil {
		s.flash(datastar.NewSSE(w, r), "bad", publicError(err))
		return
	}
	bustThumbURLs(&listing, sig.TargetKey)
	s.patchListing(w, r, listing, "Thumbnail generated.", sig.Prefix, false)
}

func bustThumbURLs(listing *models.Listing, key string) {
	stamp := strconv.FormatInt(time.Now().Unix(), 10)
	for i := range listing.Entries {
		if listing.Entries[i].Key != key {
			continue
		}
		if listing.Entries[i].ThumbURL != "" {
			listing.Entries[i].ThumbURL += "?t=" + stamp
		}
		for j := range listing.Entries[i].ThumbURLs {
			listing.Entries[i].ThumbURLs[j] += "?t=" + stamp
		}
		return
	}
}

func (s *Server) purgeThumbs(w http.ResponseWriter, r *http.Request) {
	n, err := s.files.PurgeThumbs(r.Context())
	if err != nil {
		s.log.Error("purge thumbs", "err", err)
		s.flash(datastar.NewSSE(w, r), "bad", publicError(err))
		return
	}
	msg := "No generated thumbnails were in the bucket."
	if n == 1 {
		msg = "Deleted 1 thumbnail from the bucket."
	} else if n > 1 {
		msg = fmt.Sprintf("Deleted %d thumbnails from the bucket.", n)
	}
	s.flash(datastar.NewSSE(w, r), "ok", msg)
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
		Keys    []string `json:"keys"`
		Prefix  string   `json:"prefix"`
		Folders []string `json:"folders"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if err := s.files.CompleteUploads(r.Context(), req.Prefix, req.Keys, req.Folders); err != nil {
		s.log.Error("complete", "err", err)
		http.Error(w, publicError(err), http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) thumb(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(chi.URLParam(r, "*"), "/")
	thumbKey, err := storage.ThumbKeyFromPublic(rest)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	data, contentType, etag, err := s.files.ServeThumb(r.Context(), thumbKey)
	if err != nil {
		if errors.Is(err, locks.ErrLocked) {
			http.Error(w, "locked", http.StatusForbidden)
			return
		}
		s.log.Error("thumb", "key", thumbKey, "err", err)
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "private, max-age=604800")
	w.Header().Set("ETag", etag)
	if match := r.Header.Get("If-None-Match"); match != "" && match == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
