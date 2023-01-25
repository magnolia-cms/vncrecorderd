package encoder

/**
* XXX: Ugly workaround for https://github.com/amitbet/vnc2video/issues/10. I've copied the file and build a
* X264ImageCustomEncoder. Once this is merged, we can drop the encoder.go file again.
 */

import (
	"errors"
	"fmt"
	vnc "github.com/amitbet/vnc2video"
	"github.com/amitbet/vnc2video/encoders"
	"github.com/magnolia-cms/vncrecorder/log"
	"image"
	"image/color"
	"io"
	"os"
	"os/exec"
	"strconv"
	"time"
)

func encodePPMforRGBA(w io.Writer, img *image.RGBA) error {
	maxvalue := 255
	size := img.Bounds()
	// write ppm header
	_, err := fmt.Fprintf(w, "P6\n%d %d\n%d\n", size.Dx(), size.Dy(), maxvalue)
	if err != nil {
		return err
	}

	if convImage == nil {
		convImage = make([]uint8, size.Dy()*size.Dx()*3)
	}

	rowCount := 0
	for i := 0; i < len(img.Pix); i++ {
		if (i % 4) != 3 {
			convImage[rowCount] = img.Pix[i]
			rowCount++
		}
	}

	if _, err = w.Write(convImage); err != nil {
		return err
	}

	return nil
}

func encodePPMGeneric(w io.Writer, img image.Image) error {
	maxvalue := 255
	size := img.Bounds()
	// write ppm header
	_, err := fmt.Fprintf(w, "P6\n%d %d\n%d\n", size.Dx(), size.Dy(), maxvalue)
	if err != nil {
		return err
	}

	// write the bitmap
	colModel := color.RGBAModel
	row := make([]uint8, size.Dx()*3)
	for y := size.Min.Y; y < size.Max.Y; y++ {
		i := 0
		for x := size.Min.X; x < size.Max.X; x++ {
			clr := colModel.Convert(img.At(x, y)).(color.RGBA)
			row[i] = clr.R
			row[i+1] = clr.G
			row[i+2] = clr.B
			i += 3
		}
		if _, err = w.Write(row); err != nil {
			return err
		}
	}
	return nil
}

var convImage []uint8

func encodePPM(w io.Writer, img image.Image) error {
	if img == nil {
		return errors.New("nil image")
	}
	img1, isRGBImage := img.(*vnc.RGBImage)
	img2, isRGBA := img.(*image.RGBA)
	if isRGBImage {
		return encodePPMforRGBImage(w, img1)
	} else if isRGBA {
		return encodePPMforRGBA(w, img2)
	}
	return encodePPMGeneric(w, img)
}
func encodePPMforRGBImage(w io.Writer, img *vnc.RGBImage) error {
	maxvalue := 255
	size := img.Bounds()
	// write ppm header
	_, err := fmt.Fprintf(w, "P6\n%d %d\n%d\n", size.Dx(), size.Dy(), maxvalue)
	if err != nil {
		return err
	}

	if _, err = w.Write(img.Pix); err != nil {
		return err
	}
	return nil
}

type X264ImageCustomEncoder struct {
	encoders.X264ImageEncoder
	VideoFileName      string
	ffmpegPath         string
	cmd                *exec.Cmd
	input              io.WriteCloser
	closed             bool
	Framerate          int
	ConstantRateFactor int
	Abort              bool
}

func (enc *X264ImageCustomEncoder) Init() {
	if enc.Framerate == 0 {
		enc.Framerate = 12
	}

	cmd := exec.Command(enc.ffmpegPath,
		"-f", "image2pipe",
		"-vcodec", "ppm",
		"-r", strconv.Itoa(enc.Framerate),
		"-an", // no audio
		"-y",
		"-i", "-",
		"-vcodec", "libx264",
		"-preset", "veryfast",
		"-g", "250",
		"-crf", strconv.Itoa(enc.ConstantRateFactor),
		enc.VideoFileName,
		"-hide_banner",
	)

	encInput, err := cmd.StdinPipe()
	enc.input = encInput
	if err != nil {
		log.Errorf("can't get ffmpeg input pipe. Reason: %s", err)
	}
	enc.cmd = cmd
}
func (enc *X264ImageCustomEncoder) Run() error {
	ffmpegPath, err := exec.LookPath("ffmpeg")

	if err != nil {
		log.Error("ffmpeg binary not found.")
		return err
	}

	enc.ffmpegPath = ffmpegPath
	if _, err = os.Stat(enc.ffmpegPath); os.IsNotExist(err) {
		log.Errorf("ffmpeg binary not found at %s", enc.ffmpegPath)
		return err
	}

	enc.Init()
	err = enc.cmd.Start()

	if err != nil && !enc.closed {
		log.Errorf("error while launching ffmpeg: %v. Reason: %s", enc.cmd.Args, err)
		return err
	} else {
		log.Infof("[FFMPEG] Recording %s", enc.VideoFileName)
	}

	return nil
}
func (enc *X264ImageCustomEncoder) Encode(img image.Image) bool {
	if enc.input == nil || enc.closed {
		return false
	}

	err := encodePPM(enc.input, img)

	if err != nil && !enc.closed {
		log.Warnf("error while encoding image. Reason: %s. Target file: %s", err, enc.VideoFileName)
		return false
	}
	return true
}

func (enc *X264ImageCustomEncoder) Close() {
	t := time.Now()

	if enc.closed {
		return
	}

	enc.closed = true

	if enc.input != nil {
		err := enc.input.Close()
		if err != nil {
			log.Errorf("Could not close input. Reason: %s", err)
		}

		err = enc.cmd.Wait()

		if err != nil {
			log.Errorf("Could not shut down ffmpeg. Reason: %s", err)
		}
	}

	log.Infof("[FFMPEG] Shutdown %s (%s)", enc.VideoFileName, time.Now().Sub(t))
	enc.Post()
}

func (enc *X264ImageCustomEncoder) Post() {

	//if enc.Abort

	exec.Command(enc.ffmpegPath,
		"-y",
		"-i", enc.VideoFileName,
		"-vcodec", "libx265",
		"-preset", "veryfast",
		"-tag:v", "hvc1",
		"post"+enc.VideoFileName,
		"-hide_banner",
	).Start()

}
