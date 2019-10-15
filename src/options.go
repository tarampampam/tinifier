package main

type Options struct {
	Verbose        bool     `short:"v" description:"Show verbose debug information"`
	ShowVersion    bool     `short:"V" long:"version" description:"Show version and exit"`
	FileExtensions []string `short:"e" long:"ext" default:"jpg,JPG,jpeg,JPEG,png,PNG" description:"Target file extensions"`
	ApiKey         string   `short:"k" long:"api-key" env:"TINYPNG_API_KEY" description:"API key <https://tinypng.com/dashboard/api>"`
	Threads        int      `short:"t" long:"threads" default:"5" description:"Threads processing count"`
	Targets        struct {
		Path []string `positional-arg-name:"files-and-directories"`
	} `positional-args:"yes" required:"true"`
}

var options Options
