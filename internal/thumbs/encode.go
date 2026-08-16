package thumbs

import (
	"bytes"
	"image"

	nativewebp "github.com/HugoSmits86/nativewebp"
)

func encodeWebP(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := nativewebp.Encode(&buf, img, nil); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
