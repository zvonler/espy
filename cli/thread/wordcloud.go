package thread

import (
	"flag"
	"fmt"
	"image/color"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/bbalet/stopwords"
	"github.com/psykhi/wordclouds"
	"github.com/spf13/cobra"
	"github.com/zvonler/espy/configuration"
	"github.com/zvonler/espy/database"
	"github.com/zvonler/espy/model"
	"gopkg.in/yaml.v2"
)

var path = flag.String("input", "input.yaml", "path to flat YAML like {\"word\":42,...}")
var config = flag.String("config", "config.yaml", "path to config file")
var output = flag.String("output", "output.png", "path to output image")

var DefaultColors = []color.RGBA{
	{0x1b, 0x1b, 0x1b, 0xff},
	{0x48, 0x48, 0x4B, 0xff},
	{0x59, 0x3a, 0xee, 0xff},
	{0x65, 0xCD, 0xFA, 0xff},
	{0x70, 0xD6, 0xBF, 0xff},
}

type Conf struct {
	FontMaxSize     int    `yaml:"font_max_size"`
	FontMinSize     int    `yaml:"font_min_size"`
	RandomPlacement bool   `yaml:"random_placement"`
	FontFile        string `yaml:"font_file"`
	Colors          []color.RGBA
	BackgroundColor color.RGBA `yaml:"background_color"`
	Width           int
	Height          int
	Mask            MaskConf
	SizeFunction    *string `yaml:"size_function"`
	Debug           bool
}

type MaskConf struct {
	File  string
	Color color.RGBA
}

var DefaultConf = Conf{
	FontMaxSize:     700,
	FontMinSize:     10,
	RandomPlacement: false,
	FontFile:        "./fonts/roboto/Roboto-Regular.ttf",
	Colors:          DefaultColors,
	BackgroundColor: color.RGBA{255, 255, 255, 255},
	Width:           4096,
	Height:          4096,
	Mask: MaskConf{"", color.RGBA{
		R: 0,
		G: 0,
		B: 0,
		A: 0,
	}},
	Debug: false,
}

func initWordcloudCommand() *cobra.Command {
	wordcloudCommand := &cobra.Command{
		Use:   "wordcloud <thread_id | thread_URL>...",
		Short: "Create a word cloud from the comments in the thread",
		Run:   runWordcloudCommand,
	}

	wordcloudCommand.Flags().StringVar(&startTime, "start-time", "", "Ignore comments before start-time")
	wordcloudCommand.Flags().StringVar(&endTime, "end-time", "", "Ignore comments after end-time")

	return wordcloudCommand
}

func runWordcloudCommand(cmd *cobra.Command, args []string) {
	var err error
	var sdb *database.ScraperDB
	var thread model.Thread
	var comments []model.Comment
	maxWords := 200

	wordRe := regexp.MustCompile("[A-Za-z]+")
	lightboxRe := regexp.MustCompile(`(?ms)\s+\{.*?lightbox_close.*?\}`)
	inputWords := map[string]int{}

	stopwords.LoadStopWordsFromFile("stopwords.txt", "en", "\n")

	dateTimeLayout := "20060102T15:04"

	var startTm time.Time
	var endTm time.Time

	if startTime != "" {
		startTm, _ = time.Parse(dateTimeLayout, startTime)
	}
	if endTime != "" {
		endTm, _ = time.Parse(dateTimeLayout, endTime)
	}

	if sdb, err = configuration.OpenExistingDatabase(); err == nil {
		defer sdb.Close()
		for _, threadRef := range args {
			if thread, err = sdb.FindThread(threadRef); err == nil {
				if comments, err = sdb.ThreadComments(thread.Id); err == nil {
					for _, c := range comments {
						if !startTm.IsZero() && c.Published.Before(startTm) {
							continue
						}
						if !endTm.IsZero() && c.Published.After(endTm) {
							continue
						}
						content := lightboxRe.ReplaceAllString(c.Content, "")
						relevant := stopwords.CleanString(content, "en", true)
						for _, w := range wordRe.FindAllString(relevant, -1) {
							lw := strings.ToLower(w)
							if len(lw) >= 3 {
								inputWords[strings.ToLower(w)] += 1
							}
						}
					}
				}
			}
		}
	}

	if err != nil {
		log.Fatal(err)
	}

	wordList := make([]string, len(inputWords))
	i := 0
	for w := range inputWords {
		wordList[i] = w
		i++
	}
	sort.Slice(wordList, func(i, j int) bool {
		return inputWords[wordList[i]] < inputWords[wordList[j]]
	})
	if len(wordList) > maxWords {
		wordList = wordList[len(wordList)-maxWords : len(wordList)]
	}

	displayWords := map[string]int{}
	for _, w := range wordList {
		displayWords[w] = inputWords[w]
	}
	fmt.Println(displayWords)

	// Load config
	conf := DefaultConf
	content, err := os.ReadFile(*config)
	if err == nil {
		err = yaml.Unmarshal(content, &conf)
		if err != nil {
			fmt.Printf("Failed to decode config, using defaults instead: %s\n", err)
		}
	} else {
		fmt.Println("No config file. Using defaults")
	}

	os.Chdir(filepath.Dir(*config))
	var boxes []*wordclouds.Box
	if conf.Mask.File != "" {
		boxes = wordclouds.Mask(
			conf.Mask.File,
			conf.Width,
			conf.Height,
			conf.Mask.Color)
	}

	colors := make([]color.Color, 0)
	for _, c := range conf.Colors {
		colors = append(colors, c)
	}

	start := time.Now()
	oarr := []wordclouds.Option{wordclouds.FontFile(conf.FontFile),
		wordclouds.FontMaxSize(conf.FontMaxSize),
		wordclouds.FontMinSize(conf.FontMinSize),
		wordclouds.Colors(colors),
		wordclouds.MaskBoxes(boxes),
		wordclouds.Height(conf.Height),
		wordclouds.Width(conf.Width),
		wordclouds.RandomPlacement(conf.RandomPlacement),
		wordclouds.BackgroundColor(conf.BackgroundColor)}
	if conf.SizeFunction != nil {
		oarr = append(oarr, wordclouds.WordSizeFunction(*conf.SizeFunction))
	}
	if conf.Debug {
		oarr = append(oarr, wordclouds.Debug())
	}
	w := wordclouds.NewWordcloud(displayWords,
		oarr...,
	)

	img := w.Draw()
	outputFile, err := os.Create(*output)
	if err != nil {
		panic(err)
	}

	// Encode takes a writer interface and an image interface
	// We pass it the File and the RGBA
	png.Encode(outputFile, img)

	// Don't forget to close files
	outputFile.Close()
	fmt.Printf("Done in %v\n", time.Since(start))
}
