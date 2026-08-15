package models

import "time"

type Kind string

const (
	KindFolder  Kind = "folder"
	KindImage   Kind = "image"
	KindAudio   Kind = "audio"
	KindVideo   Kind = "video"
	KindPDF     Kind = "pdf"
	KindText    Kind = "text"
	KindArchive Kind = "archive"
	KindFile    Kind = "file"
)

type Entry struct {
	Key          string    `json:"key"`
	Name         string    `json:"name"`
	IsDir        bool      `json:"isDir"`
	Size         int64     `json:"size"`
	ETag         string    `json:"etag"`
	ContentType  string    `json:"contentType"`
	LastModified time.Time `json:"lastModified"`
	Kind         Kind      `json:"kind"`
	PreviewURL   string    `json:"previewUrl,omitempty"`
	ThumbURL     string    `json:"thumbUrl,omitempty"`
	ThumbURLs    []string  `json:"thumbUrls,omitempty"`
	DownloadURL  string    `json:"downloadUrl,omitempty"`
}

type Listing struct {
	Prefix    string
	Entries   []Entry
	FromCache bool
	FetchedAt time.Time
}

type Crumb struct {
	Name   string
	Prefix string
}

type Presign struct {
	URL     string            `json:"url"`
	Key     string            `json:"key"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
}

type TrashItem struct {
	BatchID     string
	OriginalKey string
	Name        string
	IsDir       bool
	Size        int64
	Count       int
	TrashedAt   time.Time
	TrashKeys   []string
	Kind        Kind
	PreviewURL  string
}
