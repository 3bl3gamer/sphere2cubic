package main

import (
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"math"
	"os"
	"runtime"
	"strings"
	"sync"
)

const (
	North = iota
	South
	West
	East
	Top
	Bottom
)

func init() {
	image.RegisterFormat("jpeg", "jpeg", jpeg.Decode, jpeg.DecodeConfig)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func readImage(path string) (*image.RGBA, error) {
	fmt.Print("reading " + path + "...")
	fd, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(fd)
	if err != nil {
		return nil, err
	}
	fmt.Println("OK")

	rgba, ok := img.(*image.RGBA)
	if !ok {
		fmt.Print("converting to rgba... ")

		rgba = image.NewRGBA(img.Bounds())
		nThreads := runtime.NumCPU()
		iStep := (img.Bounds().Dx() + nThreads - 1) / nThreads
		var wg sync.WaitGroup

		for k := 0; k < nThreads; k++ {
			wg.Add(1)
			go func(from, to int) {
				for i := from; i < to; i++ {
					for j := 0; j < img.Bounds().Dy(); j++ {
						rgba.Set(i, j, img.At(i, j))
					}
				}
				wg.Done()
			}(k*iStep, min((k+1)*iStep, img.Bounds().Dx()))
		}

		wg.Wait()
		fmt.Println("OK")
	}
	return rgba, nil
}

func fix(pos, w int) int {
	for pos >= w {
		pos -= w
	}
	return pos
}

func lnrp(buf []byte, w, h int, x, y int, fxk, fyk float64) (byte, byte, byte) {
	xki := int(fxk * 256)
	yki := int(fyk * 256)
	xk := 255 - xki
	yk := 255 - yki
	x0 := fix(x, w)
	y0 := fix(y, h)
	x1 := fix(x+1, w)
	y1 := fix(y+1, h)
	p0 := (x0 + y0*w) * 4
	p1 := (x1 + y0*w) * 4
	p2 := (x0 + y1*w) * 4
	p3 := (x1 + y1*w) * 4
	r0, g0, b0 := int(buf[p0]), int(buf[p0+1]), int(buf[p0+2])
	r1, g1, b1 := int(buf[p1]), int(buf[p1+1]), int(buf[p1+2])
	r2, g2, b2 := int(buf[p2]), int(buf[p2+1]), int(buf[p2+2])
	r3, g3, b3 := int(buf[p3]), int(buf[p3+1]), int(buf[p3+2])
	r := byte(((r0*xk+r1*xki)*yk + (r2*xk+r3*xki)*yki) / (256 * 256))
	g := byte(((g0*xk+g1*xki)*yk + (g2*xk+g3*xki)*yki) / (256 * 256))
	b := byte(((b0*xk+b1*xki)*yk + (b2*xk+b3*xki)*yki) / (256 * 256))
	return r, g, b
}

type sideFunc func(ri, rj, hRot, srcRWidth, srcRHeight float64) (xSrc, ySrc, xk, yk float64)

func vertSideFunc(ri, rj, hRot, srcRWidth, srcRHeight float64) (xSrc, ySrc, xk, yk float64) {
	dx := (math.Atan(ri) + hRot) / math.Pi / 2
	dy := (math.Atan2(rj, math.Sqrt(ri*ri+1))) / math.Pi
	xSrc, xk = math.Modf(srcRWidth * (1.5 - dx))
	ySrc, yk = math.Modf(srcRHeight * (0.5 - dy))
	return
}

func topSideFunc(ri, rj, hRot, srcRWidth, srcRHeight float64) (xSrc, ySrc, xk, yk float64) {
	dx := (math.Atan2(ri, rj) + hRot) / math.Pi / 2
	dy := (math.Atan(math.Sqrt(ri*ri + rj*rj))) / math.Pi
	xSrc, xk = math.Modf(srcRWidth * (1 + dx))
	ySrc, yk = math.Modf(srcRHeight * (dy))
	return
}

func bottomSideFunc(ri, rj, hRot, srcRWidth, srcRHeight float64) (xSrc, ySrc, xk, yk float64) {
	dx := (math.Atan2(ri, rj) + hRot) / math.Pi / 2
	dy := (math.Atan(math.Sqrt(ri*ri + rj*rj))) / math.Pi
	xSrc, xk = math.Modf(srcRWidth * (1.5 - dx))
	ySrc, yk = math.Modf(srcRHeight * (1 - dy))
	return
}

func extractSide(src *image.RGBA, width int, sf sideFunc, hRot float64) *image.RGBA {
	invWidth := 1 / float64(width)
	srcWidth := src.Bounds().Dx()
	srcHeight := src.Bounds().Dy()
	srcRWidth := float64(srcWidth)
	srcRHeight := float64(srcHeight)

	res := image.NewRGBA(image.Rect(0, 0, width, width))
	buf := res.Pix
	bufSrc := src.Pix

	for i := 0; i < width; i++ {
		for j := 0; j < width; j++ {
			ri := 1 - float64(i)*invWidth*2
			rj := 1 - float64(j)*invWidth*2
			xSrc, ySrc, xk, yk := sf(ri, rj, hRot, srcRWidth, srcRHeight)

			r, g, b := lnrp(bufSrc, srcWidth, srcHeight, int(xSrc), int(ySrc), xk, yk)
			o := res.PixOffset(i, j)
			buf[o], buf[o+1], buf[o+2] = r, g, b
		}
	}
	return res
}

func saveSide(img *image.RGBA, fname string) error {
	fd, err := os.Create(fname + ".jpg")
	if err != nil {
		return err
	}
	err = jpeg.Encode(fd, img, &jpeg.Options{jpeg.DefaultQuality})
	if err != nil {
		return err
	}
	return nil
}

func goAndSave(wg *sync.WaitGroup, fname string, src *image.RGBA, width int, sf sideFunc, hRot float64) {
	wg.Add(1)
	go func() {
		fmt.Println("processing: " + fname)
		img := extractSide(src, width, sf, hRot)
		err := saveSide(img, fname)
		if err != nil {
			fmt.Println(err.Error())
		}
		fmt.Println("saved: " + fname)
		wg.Done()
	}()
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	var srcPath string
	var prefix string
	var sidesStr string
	var hRot float64
	var width int

	flag.StringVar(&srcPath, "src", "sphere.jpg", "source file path")
	flag.StringVar(&prefix, "prefix", "cube_", "name prefix for output images (<prefix><sidename>.jpg, cube_north.jpg, cube_top.jpg etc)")
	flag.StringVar(&sidesStr, "sides", "north,south,west,east,top,bottom", "sides names for output images")
	flag.Float64Var(&hRot, "rot", 0, "additional rotation around vertical axis (degrees)")
	flag.IntVar(&width, "width", 256, "cube side width (in pixels)")
	flag.Parse()

	sides := strings.Split(sidesStr, ",")
	if len(sides) != 6 {
		fmt.Println("'sides' must contain six comma-separated names")
		return
	}

	img, err := readImage(srcPath)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	var wg sync.WaitGroup
	goAndSave(&wg, prefix+sides[North], img, width, vertSideFunc, 0)
	goAndSave(&wg, prefix+sides[South], img, width, vertSideFunc, math.Pi)
	goAndSave(&wg, prefix+sides[West], img, width, vertSideFunc, -math.Pi/2)
	goAndSave(&wg, prefix+sides[East], img, width, vertSideFunc, math.Pi/2)
	goAndSave(&wg, prefix+sides[Top], img, width, topSideFunc, 0)
	goAndSave(&wg, prefix+sides[Bottom], img, width, bottomSideFunc, 0)
	wg.Wait()
}
