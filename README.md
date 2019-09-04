# Read JPEG quality from bytes or os.File

Inpiration from [Estimating Quality](http://fotoforensics.com/tutorial-estq.php)

Code Borrowed from [jhead](http://www.sentex.net/~mwandel/jhead/)

Code based on [liut/jpegquality](https://github.com/liut/jpegquality)

Test image comes from [recurser/exif-orientation-examples](https://github.com/recurser/exif-orientation-examples)

## usage:

````go

	file, err := os.Open("file.jpg")
	if err != nil {
		log.Fatal(err)
	}
	j, err := jpegquality.New(file) // or NewWithBytes([]byte)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("jpeg quality %d", j.Quality())
````
