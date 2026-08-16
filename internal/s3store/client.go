package s3store

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"

	"station/internal/config"
	"station/internal/models"
	"station/internal/storage"
)

type Client struct {
	api     *s3.Client
	presign *s3.PresignClient
	bucket  string
	ttl     time.Duration
}

func New(ctx context.Context, cfg config.Config) (*Client, error) {
	api, err := newS3(ctx, cfg, cfg.S3Endpoint)
	if err != nil {
		return nil, err
	}

	presignAPI := api
	if cfg.S3PublicURL != "" && cfg.S3PublicURL != cfg.S3Endpoint {
		presignAPI, err = newS3(ctx, cfg, cfg.S3PublicURL)
		if err != nil {
			return nil, fmt.Errorf("public s3 client: %w", err)
		}
	}

	c := &Client{
		api:     api,
		presign: s3.NewPresignClient(presignAPI),
		bucket:  cfg.S3Bucket,
		ttl:     cfg.PresignTTL,
	}

	deadline := time.Now().Add(45 * time.Second)
	for {
		err = c.verifyBucket(ctx)
		if err == nil {
			return c, nil
		}
		if time.Now().After(deadline) {
			return nil, err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Second):
		}
	}
}

func newS3(ctx context.Context, cfg config.Config, endpoint string) (*s3.Client, error) {
	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.AWSRegion),
	}
	if cfg.AWSAccessKey != "" && cfg.AWSSecretKey != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AWSAccessKey, cfg.AWSSecretKey, ""),
		))
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("aws config: %w", err)
	}

	return s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
		}
		o.UsePathStyle = cfg.S3PathStyle
	}), nil
}

func (c *Client) verifyBucket(ctx context.Context) error {
	_, err := c.api.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(c.bucket)})
	if err == nil {
		return nil
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) && (apiErr.ErrorCode() == "NotFound" || apiErr.ErrorCode() == "NoSuchBucket" || apiErr.ErrorCode() == "404") {
		return fmt.Errorf("bucket %q was not found; set S3_BUCKET to an existing bucket", c.bucket)
	}
	return fmt.Errorf("cannot access bucket %q: %w", c.bucket, err)
}

func (c *Client) List(ctx context.Context, prefix string) ([]models.Entry, error) {
	input := &s3.ListObjectsV2Input{
		Bucket:    aws.String(c.bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String("/"),
	}

	seen := map[string]struct{}{}
	entries := make([]models.Entry, 0)
	paginator := s3.NewListObjectsV2Paginator(c.api, input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("list objects: %w", err)
		}
		for _, p := range page.CommonPrefixes {
			key := aws.ToString(p.Prefix)
			if key == "" || key == prefix || storage.IsReservedName(storage.BaseName(key)) {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			entries = append(entries, models.Entry{
				Key:  key,
				Name: storage.BaseName(key),
				IsDir: true,
				Kind:  models.KindFolder,
			})
		}
		for _, obj := range page.Contents {
			key := aws.ToString(obj.Key)
			if key == "" || key == prefix {
				continue
			}
			if strings.HasSuffix(key, "/") {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			name := storage.BaseName(key)
			entries = append(entries, models.Entry{
				Key:          key,
				Name:         name,
				IsDir:        false,
				Size:         aws.ToInt64(obj.Size),
				ETag:         strings.Trim(aws.ToString(obj.ETag), `"`),
				LastModified: aws.ToTime(obj.LastModified),
				Kind:         storage.KindFromName(name, false),
				ContentType:  storage.ContentTypeFromName(name, ""),
			})
		}
	}
	return entries, nil
}

func (c *Client) PutFolder(ctx context.Context, key string) error {
	_, err := c.api.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(nil),
		ContentType: aws.String("application/x-directory"),
	})
	if err != nil {
		return fmt.Errorf("create folder: %w", err)
	}
	return nil
}

func (c *Client) Delete(ctx context.Context, key string, recursive bool) error {
	if !recursive {
		_, err := c.api.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(c.bucket),
			Key:    aws.String(key),
		})
		return err
	}

	prefix := key
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	paginator := s3.NewListObjectsV2Paginator(c.api, &s3.ListObjectsV2Input{
		Bucket: aws.String(c.bucket),
		Prefix: aws.String(prefix),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return err
		}
		if len(page.Contents) == 0 {
			continue
		}
		objs := make([]types.ObjectIdentifier, 0, len(page.Contents)+1)
		for _, obj := range page.Contents {
			objs = append(objs, types.ObjectIdentifier{Key: obj.Key})
		}
		if _, err := c.api.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(c.bucket),
			Delete: &types.Delete{Objects: objs, Quiet: aws.Bool(true)},
		}); err != nil {
			return err
		}
	}
	_, _ = c.api.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(strings.TrimRight(key, "/") + "/"),
	})
	return nil
}

func (c *Client) Head(ctx context.Context, key string) (models.Entry, error) {
	out, err := c.api.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return models.Entry{}, err
	}
	name := storage.BaseName(key)
	return models.Entry{
		Key:          key,
		Name:         name,
		IsDir:        storage.IsFolderKey(key),
		Size:         aws.ToInt64(out.ContentLength),
		ETag:         strings.Trim(aws.ToString(out.ETag), `"`),
		ContentType:  storage.ContentTypeFromName(name, aws.ToString(out.ContentType)),
		LastModified: aws.ToTime(out.LastModified),
		Kind:         storage.ResolveKind(name, aws.ToString(out.ContentType), storage.IsFolderKey(key)),
	}, nil
}

func (c *Client) Get(ctx context.Context, key string) ([]byte, string, error) {
	body, contentType, err := c.Open(ctx, key)
	if err != nil {
		return nil, "", err
	}
	defer body.Close()
	data, err := io.ReadAll(io.LimitReader(body, 32<<20))
	if err != nil {
		return nil, "", err
	}
	return data, contentType, nil
}

func (c *Client) Open(ctx context.Context, key string) (io.ReadCloser, string, error) {
	out, err := c.api.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, "", err
	}
	return out.Body, aws.ToString(out.ContentType), nil
}

func (c *Client) Put(ctx context.Context, key string, body []byte, contentType string) error {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	_, err := c.api.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(body),
		ContentType: aws.String(contentType),
	})
	return err
}

func (c *Client) PresignPut(ctx context.Context, key, contentType string) (models.Presign, error) {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	out, err := c.presign.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(c.ttl))
	if err != nil {
		return models.Presign{}, fmt.Errorf("presign put: %w", err)
	}
	return models.Presign{
		URL:    out.URL,
		Key:    key,
		Method: out.Method,
		Headers: map[string]string{
			"Content-Type": contentType,
		},
	}, nil
}

func (c *Client) PresignGet(ctx context.Context, key string) (string, error) {
	out, err := c.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(c.ttl))
	if err != nil {
		return "", fmt.Errorf("presign get: %w", err)
	}
	return out.URL, nil
}

func (c *Client) PresignDownload(ctx context.Context, key, filename string) (string, error) {
	out, err := c.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket:                     aws.String(c.bucket),
		Key:                        aws.String(key),
		ResponseContentDisposition: aws.String(contentDisposition(filename)),
	}, s3.WithPresignExpires(c.ttl))
	if err != nil {
		return "", fmt.Errorf("presign download: %w", err)
	}
	return out.URL, nil
}

func contentDisposition(filename string) string {
	filename = strings.ReplaceAll(filename, `"`, `'`)
	filename = strings.ReplaceAll(filename, "\n", "")
	filename = strings.ReplaceAll(filename, "\r", "")
	if filename == "" {
		filename = "download"
	}
	return fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, filename, url.PathEscape(filename))
}

type Object struct {
	Key          string
	Size         int64
	ETag         string
	LastModified time.Time
}

func (c *Client) ListAll(ctx context.Context, prefix string) ([]Object, error) {
	var out []Object
	paginator := s3.NewListObjectsV2Paginator(c.api, &s3.ListObjectsV2Input{
		Bucket: aws.String(c.bucket),
		Prefix: aws.String(prefix),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("list objects: %w", err)
		}
		for _, obj := range page.Contents {
			key := aws.ToString(obj.Key)
			if key == "" {
				continue
			}
			out = append(out, Object{
				Key:          key,
				Size:         aws.ToInt64(obj.Size),
				ETag:         strings.Trim(aws.ToString(obj.ETag), `"`),
				LastModified: aws.ToTime(obj.LastModified),
			})
		}
	}
	return out, nil
}

func (c *Client) Copy(ctx context.Context, src, dest string) error {
	_, err := c.api.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(c.bucket),
		Key:        aws.String(dest),
		CopySource: aws.String(c.bucket + "/" + encodeCopySource(src)),
	})
	if err != nil {
		return fmt.Errorf("copy %s -> %s: %w", src, dest, err)
	}
	return nil
}

func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	_, err := c.api.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err == nil {
		return true, nil
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) && (apiErr.ErrorCode() == "NotFound" || apiErr.ErrorCode() == "NoSuchKey" || apiErr.ErrorCode() == "404") {
		return false, nil
	}
	return false, err
}

func (c *Client) DeleteKeys(ctx context.Context, keys []string) error {
	for i := 0; i < len(keys); i += 1000 {
		end := i + 1000
		if end > len(keys) {
			end = len(keys)
		}
		chunk := keys[i:end]
		objs := make([]types.ObjectIdentifier, 0, len(chunk))
		for _, key := range chunk {
			k := key
			objs = append(objs, types.ObjectIdentifier{Key: &k})
		}
		if _, err := c.api.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(c.bucket),
			Delete: &types.Delete{Objects: objs, Quiet: aws.Bool(true)},
		}); err != nil {
			return err
		}
	}
	return nil
}

func encodeCopySource(key string) string {
	parts := strings.Split(key, "/")
	for i, part := range parts {
		parts[i] = strings.ReplaceAll(url.PathEscape(part), "+", "%20")
	}
	return strings.Join(parts, "/")
}
