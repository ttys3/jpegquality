package jpegquality

import (
	"io/ioutil"
	"log"
	"testing"
)

func init() {
	log.SetFlags(log.Ltime | log.Lshortfile)
}

func TestJpegQuality(t *testing.T) {
	jpeg_data, _:= ioutil.ReadFile("./testdata/Landscape_3.jpg")
	j, err := NewWithBytes(jpeg_data)
	if err != nil {
		t.Fatal(err)
	}

	//infact it is 73
	t.Logf("jpeg quality %d", j.Quality())
	if j.Quality() != 73 {
		t.FailNow()
	}
}
