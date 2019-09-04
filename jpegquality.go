package jpegquality

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	//"math"
)

var (
	ErrInvalidJPEG = errors.New("Invalid JPEG content")
	ErrWrongTable  = errors.New("ERROR: Wrong size for quantization table")
)

//--------------------------------------------------------------------------
// @author HuangYeWuDeng
// This file handles guessing of jpeg quality from quantization table
// Using code from jhead http://www.sentex.net/~mwandel/jhead/
// golang Code based on [liut/jpegquality](https://github.com/liut/jpegquality)
//--------------------------------------------------------------------------

//idct.go
const blockSize = 64 // A DCT block is 8x8.

type block [blockSize]int32

const (
	//from  /usr/lib/go/src/image/jpeg/reader.go
	dhtMarker  = 0xc4 // Define Huffman Table.
	dqtMarker  = 0xdb // Define Quantization Table.
	maxTq   = 3
)

var quant      [maxTq + 1]block // Quantization tables, in zig-zag order.

// for the DQT marker -- start --
// Sample quantization tables from JPEG spec --- only needed for
// guesstimate of quality factor.  Note these are in zigzag order.

var std_luminance_quant_tbl = [64]int{
16,  11,  12,  14,  12,  10,  16,  14,
13,  14,  18,  17,  16,  19,  24,  40,
26,  24,  22,  22,  24,  49,  35,  37,
29,  40,  58,  51,  61,  60,  57,  51,
56,  55,  64,  72,  92,  78,  64,  68,
87,  69,  55,  56,  80, 109,  81,  87,
95,  98, 103, 104, 103,  62,  77, 113,
121, 112, 100, 120,  92, 101, 103,  99,
}

var std_chrominance_quant_tbl = [64]int{
17,  18,  18,  24,  21,  24,  47,  26,
26,  47,  99,  66,  56,  66,  99,  99,
99,  99,  99,  99,  99,  99,  99,  99,
99,  99,  99,  99,  99,  99,  99,  99,
99,  99,  99,  99,  99,  99,  99,  99,
99,  99,  99,  99,  99,  99,  99,  99,
99,  99,  99,  99,  99,  99,  99,  99,
99,  99,  99,  99,  99,  99,  99,  99,
}

var deftabs = [2][64]int{
	std_luminance_quant_tbl, std_chrominance_quant_tbl,
}
// jpeg_zigzag_order[i] is the zigzag-order position of the i'th element
// of a DCT block read in natural order (left to right, top to bottom).

var jpeg_zigzag_order = [64]int{
0,  1,  5,  6, 14, 15, 27, 28,
2,  4,  7, 13, 16, 26, 29, 42,
3,  8, 12, 17, 25, 30, 41, 43,
9, 11, 18, 24, 31, 40, 44, 53,
10, 19, 23, 32, 39, 45, 52, 54,
20, 22, 33, 38, 46, 51, 55, 60,
21, 34, 37, 47, 50, 56, 59, 61,
35, 36, 48, 49, 57, 58, 62, 63,
}
// for the DQT marker -- end --


type jpegReader struct {
	rs      io.ReadSeeker
	quality int
}

func NewWithBytes(buf []byte) (jr *jpegReader, err error) {
	return New(bytes.NewReader(buf))
}

func New(rs io.ReadSeeker) (jr *jpegReader, err error) {
	jr = &jpegReader{rs: rs}
	_, err = jr.rs.Seek(0, 0)
	if err != nil {
		return
	}

	var (
		sign = make([]byte, 2)
	)
	_, err = jr.rs.Read(sign)
	if err != nil {
		return
	}
	if sign[0] != 0xff && sign[1] != 0xd8 {
		err = ErrInvalidJPEG
	}

	var q int
	q, err = jr.readQuality()
	if err == nil {
		jr.quality = q
	}
	return
}

func (this *jpegReader) readQuality() (q int, err error) {
	for {
		mark := this.readMarker()
		if mark == 0 {
			err = ErrInvalidJPEG
			return
		}
		var (
			length, index int
			sign          = make([]byte, 2)
			//qualityAvg    = make([]float64, 3)
		)
		_, err = this.rs.Read(sign)
		if err != nil {
			log.Printf("read err %s", err)
			return
		}

		//ref to func (d *decoder) decode(r io.Reader, configOnly bool) (image.Image, error) {
		length = int(sign[0])<<8 + int(sign[1]) - 2
		if length < 0 {
			err = fmt.Errorf("short segment length")
			return
		}

		// 0xdb Define Quantization Table.
		if (mark & 0xff) != dqtMarker { // not a quantization table
			_, err = this.rs.Seek(int64(length), 1)
			if err != nil {
				log.Printf("seek err %s", err)
				return
			}
			continue
		}

		//yes, we got the dqtMarker
		if length%65 != 0 {
			log.Printf("ERROR: Wrong size for quantization table -- this contains %d bytes (%d bytes short or %d bytes long)\n", length, 65-length%65, length%65)
			err = ErrWrongTable
			return
		}

		log.Printf("length %d", length)
		log.Print("Quantization table")

		var tabuf = make([]byte, length)
		_, err = this.rs.Read(tabuf)
		if err != nil {
			log.Printf("read err %s", err)
			return
		}

		//tableindex
		index = int(tabuf[0] & 0x0f)

		//we only process DQT
		if index != 0 {
			continue
		}

		var allones int
		var cumsf, cumsf2 float64
		buf := tabuf[0:65]

		//tableindex
		index = int(buf[0] & 0x0f)
		//precision: (c>>4) ? 16 : 8
		precision := 8
		if int8(buf[0])>>4 > 0 {
			precision = 16
		}

		reftable := deftabs[index]
		log.Printf("  Precision=%d; Table index=%d (%s)\n", precision, index, getTableName(index))

		a := 2
		for coefindex := 0; coefindex < 64 && a < length; coefindex++ {
			var val int

			if index>>4 != 0 {
				temp := int(buf[a])
				a++
				temp *= 256;
				val = int(buf[a]) + temp;
				a++
			} else {
				val = int(buf[a])
				a++
			}

			// scaling factor in percent
			x := 100.0 * float64(val) / float64(reftable[coefindex])
			cumsf += x;
			cumsf2 += x * x;
			// separate check for all-ones table (Q 100)
			if val != 1 {
				allones = 0
			}
		}

		var qual float64
		cumsf /= 64.0; // mean scale factor
		cumsf2 /= 64.0;
		//var2 = cumsf2 - (cumsf * cumsf); // variance
		if allones == 1 { // special case for all-ones table
			qual = 100.0;
		} else if (cumsf <= 100.0) {
			qual = (200.0 - cumsf) / 2.0;
		} else {
			qual = 5000.0 / cumsf
		}
		q = (int)(qual + 0.5)
		log.Printf("aver_quality %#v", q)
		break
	}
	return
}

func getTableName(index int) string {
	if index > 0 {
		return "chrominance"
	}
	return "luminance"
}

func (this *jpegReader) readMarker() int {
	var (
		mark = make([]byte, 2)
		err  error
	)

ReadAgain:
	_, err = this.rs.Read(mark)
	if err != nil {
		return 0
	}
	if mark[0] != 0xff || mark[1] == 0xff || mark[1] == 0x00 {
		goto ReadAgain
	}

	// log.Printf("get marker %x", mark)
	return int(mark[0])*256 + int(mark[1])
}

func (this *jpegReader) Quality() int {
	return this.quality
}
