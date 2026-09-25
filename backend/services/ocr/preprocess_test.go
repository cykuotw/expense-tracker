package ocr

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreprocessRasterStripsJPEGMetadata(t *testing.T) {
	imageBytes := encodeTestJPEG(t, 8, 6)
	sensitiveMarker := []byte("Exif\x00\x00SENSITIVE_GPS_METADATA")
	segmentLength := len(sensitiveMarker) + 2
	app1 := []byte{0xff, 0xe1, byte(segmentLength >> 8), byte(segmentLength)}
	withMetadata := append([]byte{}, imageBytes[:2]...)
	withMetadata = append(withMetadata, app1...)
	withMetadata = append(withMetadata, sensitiveMarker...)
	withMetadata = append(withMetadata, imageBytes[2:]...)

	processed, err := PreprocessRaster(bytes.NewReader(withMetadata), DefaultPreprocessOptions())
	require.NoError(t, err)
	assert.Equal(t, "jpeg", processed.SourceFormat)
	assert.Equal(t, 8, processed.Width)
	assert.Equal(t, 6, processed.Height)
	assert.NotContains(t, string(processed.Bytes), "SENSITIVE_GPS_METADATA")
	_, format, err := image.Decode(bytes.NewReader(processed.Bytes))
	require.NoError(t, err)
	assert.Equal(t, "jpeg", format)
}

func TestPreprocessRasterRejectsSourceByteLimit(t *testing.T) {
	_, err := PreprocessRaster(bytes.NewReader([]byte("too large")), PreprocessOptions{MaxSourceBytes: 4})
	assert.ErrorIs(t, err, ErrInputTooLarge)
}

func TestPreprocessRasterRejectsDimensionLimit(t *testing.T) {
	imageBytes := encodeTestJPEG(t, 9, 2)
	_, err := PreprocessRaster(bytes.NewReader(imageBytes), PreprocessOptions{MaxDimension: 8})
	assert.ErrorIs(t, err, ErrDimensionsTooLarge)
}

func TestPreprocessRasterRejectsOutputByteLimit(t *testing.T) {
	imageBytes := encodeTestJPEG(t, 8, 8)
	_, err := PreprocessRaster(bytes.NewReader(imageBytes), PreprocessOptions{MaxOutputBytes: 8})
	assert.ErrorIs(t, err, ErrOutputTooLarge)
}

func TestPreprocessRasterRejectsMalformedContent(t *testing.T) {
	_, err := PreprocessRaster(bytes.NewReader([]byte("not an image")), DefaultPreprocessOptions())
	assert.Error(t, err)
	assert.False(t, errors.Is(err, ErrUnsupportedFormat))
}

func encodeTestJPEG(t *testing.T, width, height int) []byte {
	t.Helper()
	fixture := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			fixture.Set(x, y, color.RGBA{R: uint8(x * 10), G: uint8(y * 10), B: 128, A: 255})
		}
	}
	var encoded bytes.Buffer
	require.NoError(t, jpeg.Encode(&encoded, fixture, nil))
	return encoded.Bytes()
}
