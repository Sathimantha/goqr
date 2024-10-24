// certificate/utils.go
package certificate

import (
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"log"
	"os"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// TextRenderer handles text operations on images
type TextRenderer struct {
	img *image.RGBA
}

// NewTextRenderer creates a new text renderer
func NewTextRenderer(img *image.RGBA) *TextRenderer {
	return &TextRenderer{img: img}
}

// LoadFont loads a TrueType font from a file
func LoadFont(fontPath string, scale float64) font.Face {
	fontBytes, err := ioutil.ReadFile(fontPath)
	if err != nil {
		log.Fatalf("failed to read font file: %v", err)
	}

	f, err := opentype.Parse(fontBytes)
	if err != nil {
		log.Fatalf("failed to parse font: %v", err)
	}

	const dpi = 72 // standard DPI
	fontFace, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size:    scale,
		DPI:     dpi,
		Hinting: font.HintingNone,
	})
	if err != nil {
		log.Fatalf("failed to create font face: %v", err)
	}

	return fontFace
}

// AddText adds text to an image
func (tr *TextRenderer) AddText(text string, imgWidth, yPos int, fontScale float64, fontPath string) {
	// Load the font face using LoadFont function
	face := LoadFont(fontPath, fontScale)

	fg := image.NewUniform(color.RGBA{255, 0, 0, 255})

	// Create a new font drawer with the loaded face
	d := &font.Drawer{
		Dst:  tr.img,
		Src:  fg,
		Face: face,
	}

	// Measure the width of the text
	textWidth := d.MeasureString(text).Ceil()

	// Calculate the available margin and max width allowed for the text
	availableWidth := imgWidth
	maxWidth := int(float64(availableWidth) * 0.93)

	// If the text width exceeds the max width, adjust the font scale
	if textWidth > maxWidth {
		// Calculate the new font scale to fit the text within the max width
		scaleFactor := float64(maxWidth) / float64(textWidth)
		fontScale *= scaleFactor

		// Load the font face again with the new font scale
		face = LoadFont(fontPath, fontScale)

		// Update the font drawer with the new face
		d.Face = face

		// Measure the text width again with the new font scale
		textWidth = d.MeasureString(text).Ceil()
	}

	// Calculate the starting point to center the text
	point := fixed.Point26_6{
		X: fixed.Int26_6((availableWidth - textWidth) / 2 * 64), // Center the text
		Y: fixed.Int26_6(yPos * 64),
	}

	d.Dot = point      // Set the starting point for the text
	d.DrawString(text) // Draw the string
}

// LoadQRCode loads a QR code image from file
func LoadQRCode(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return png.Decode(file)
}

// SaveAsJPEG saves an RGBA image as JPEG
func SaveAsJPEG(img *image.RGBA, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return jpeg.Encode(file, img, nil)
}
