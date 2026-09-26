package ocr

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png"
	"io"
)

const (
	DefaultMaxSourceBytes int64 = 20 << 20
	DefaultMaxOutputBytes       = 5 << 20
	DefaultMaxDimension         = 8_192
	DefaultMaxPixels      int64 = 25_000_000
	defaultJPEGQuality          = 90
)

var (
	ErrInputTooLarge      = errors.New("image input exceeds byte limit")
	ErrOutputTooLarge     = errors.New("normalized image exceeds provider byte limit")
	ErrUnsupportedFormat  = errors.New("unsupported image format")
	ErrDimensionsTooLarge = errors.New("image dimensions exceed limit")
	ErrPreprocessTimeout  = errors.New("image preprocessing deadline exceeded")
)

// PreprocessOptions bounds image decoding and provider payload size.
type PreprocessOptions struct {
	MaxSourceBytes int64
	MaxOutputBytes int
	MaxDimension   int
	MaxPixels      int64
	JPEGQuality    int
}

// PreprocessedImage is a metadata-free JPEG ready for OCR.
type PreprocessedImage struct {
	Bytes        []byte
	SourceFormat string
	Width        int
	Height       int
}

// DefaultPreprocessOptions returns conservative limits for receipt photos.
func DefaultPreprocessOptions() PreprocessOptions {
	return PreprocessOptions{
		MaxSourceBytes: DefaultMaxSourceBytes,
		MaxOutputBytes: DefaultMaxOutputBytes,
		MaxDimension:   DefaultMaxDimension,
		MaxPixels:      DefaultMaxPixels,
		JPEGQuality:    defaultJPEGQuality,
	}
}

// PreprocessRaster validates, decodes, flattens, and re-encodes a JPEG or PNG.
// Re-encoding intentionally drops EXIF and other source metadata.
func PreprocessRaster(reader io.Reader, options PreprocessOptions) (PreprocessedImage, error) {
	return PreprocessRasterContext(context.Background(), reader, options)
}

// PreprocessRasterContext applies the same bounded transformation while
// honoring cancellation between the bounded decode and encode stages.
func PreprocessRasterContext(ctx context.Context, reader io.Reader, options PreprocessOptions) (PreprocessedImage, error) {
	options = withPreprocessDefaults(options)
	if err := preprocessingContextError(ctx); err != nil {
		return PreprocessedImage{}, err
	}
	limited := io.LimitReader(reader, options.MaxSourceBytes+1)
	source, err := io.ReadAll(limited)
	if err != nil {
		return PreprocessedImage{}, fmt.Errorf("read image: %w", err)
	}
	if int64(len(source)) > options.MaxSourceBytes {
		return PreprocessedImage{}, ErrInputTooLarge
	}
	if err := preprocessingContextError(ctx); err != nil {
		return PreprocessedImage{}, err
	}

	config, format, err := image.DecodeConfig(bytes.NewReader(source))
	if err != nil {
		return PreprocessedImage{}, fmt.Errorf("decode image configuration: %w", err)
	}
	if format != "jpeg" && format != "png" {
		return PreprocessedImage{}, fmt.Errorf("%w: %s", ErrUnsupportedFormat, format)
	}
	if config.Width <= 0 || config.Height <= 0 ||
		config.Width > options.MaxDimension || config.Height > options.MaxDimension ||
		int64(config.Width)*int64(config.Height) > options.MaxPixels {
		return PreprocessedImage{}, ErrDimensionsTooLarge
	}
	if err := preprocessingContextError(ctx); err != nil {
		return PreprocessedImage{}, err
	}

	decoded, _, err := image.Decode(bytes.NewReader(source))
	if err != nil {
		return PreprocessedImage{}, fmt.Errorf("decode image: %w", err)
	}
	bounds := decoded.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width != config.Width || height != config.Height || width <= 0 || height <= 0 ||
		width > options.MaxDimension || height > options.MaxDimension ||
		int64(width)*int64(height) > options.MaxPixels {
		return PreprocessedImage{}, ErrDimensionsTooLarge
	}
	if err := preprocessingContextError(ctx); err != nil {
		return PreprocessedImage{}, err
	}
	flattened := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(flattened, flattened.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.Draw(flattened, flattened.Bounds(), decoded, bounds.Min, draw.Over)
	if err := preprocessingContextError(ctx); err != nil {
		return PreprocessedImage{}, err
	}

	var output bytes.Buffer
	if err := jpeg.Encode(&output, flattened, &jpeg.Options{Quality: options.JPEGQuality}); err != nil {
		return PreprocessedImage{}, fmt.Errorf("encode normalized image: %w", err)
	}
	if output.Len() > options.MaxOutputBytes {
		return PreprocessedImage{}, ErrOutputTooLarge
	}
	if err := preprocessingContextError(ctx); err != nil {
		return PreprocessedImage{}, err
	}

	return PreprocessedImage{
		Bytes:        output.Bytes(),
		SourceFormat: format,
		Width:        bounds.Dx(),
		Height:       bounds.Dy(),
	}, nil
}

func withPreprocessDefaults(options PreprocessOptions) PreprocessOptions {
	defaults := DefaultPreprocessOptions()
	if options.MaxSourceBytes <= 0 {
		options.MaxSourceBytes = defaults.MaxSourceBytes
	}
	if options.MaxOutputBytes <= 0 {
		options.MaxOutputBytes = defaults.MaxOutputBytes
	}
	if options.MaxDimension <= 0 {
		options.MaxDimension = defaults.MaxDimension
	}
	if options.MaxPixels <= 0 {
		options.MaxPixels = defaults.MaxPixels
	}
	if options.JPEGQuality <= 0 || options.JPEGQuality > 100 {
		options.JPEGQuality = defaults.JPEGQuality
	}
	return options
}

func preprocessingContextError(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("%w: %w", ErrPreprocessTimeout, err)
	}
	return nil
}
