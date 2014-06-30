// Copyright (c) 2014 Piotr S. Staszewski. All rights reserved.
// See LICENSE.txt for licensing information.

package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"camlistore.org/pkg/blob"
	"camlistore.org/pkg/cacher"
	"camlistore.org/pkg/client"
	"camlistore.org/pkg/schema"
	"camlistore.org/pkg/search"
)

const (
	ADD = iota
	DEL
)

var (
	flgWorkers = flag.Int("workers", 1, "number of parallel workers")
	flgCache   = flag.Bool("cache", false, "use disk cache fetcher")
	cl         *client.Client
	wg         sync.WaitGroup
	args       chan string
	tags       []string
	mode       int
)

func main() {
	flag.Usage = usage
	// seems to be required for proper client functioning
	client.AddFlags()
	flag.Parse()

	if len(flag.Args()) < 4 {
		usage()
	}

	if flag.Arg(2) != "--" {
		usage()
	}

	switch flag.Arg(0) {
	case "add":
		mode = ADD
	case "del":
		mode = DEL
	default:
		usage()
	}

	tags = strings.Split(flag.Arg(1), ",")
	if len(tags) < 1 {
		usage()
	}

	cl = client.NewOrFail()

	// seems to be required for proper client functioning
	tr := cl.TransportForConfig(&client.TransportConfig{})
	cl.SetHTTPClient(&http.Client{Transport: tr})

	// disk cache fetcher
	if *flgCache {
		dcf, err := cacher.NewDiskCache(cl)
		if err != nil {
			fmt.Fprintf(os.Stderr, "CACHER: %v\n", err)
		}
		defer dcf.Clean()
	}

	args = make(chan string)

	for i := 0; i < *flgWorkers; i++ {
		wg.Add(1)
		go worker()
	}

	for _, arg := range flag.Args()[3:] {
		args <- arg
	}
	close(args)
	wg.Wait()
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [options] add|del tag,tag... -- file file...\n\n", os.Args[0])
	flag.PrintDefaults()
	fmt.Fprintf(
		os.Stderr,
		"\nFiles are matched with permanodes by filename and size (exact match).\n\n"+
			"Results format: path blobref optag optag...\n"+
			"            or: path error\n\n"+
			"Op legend: + tag has been added       - tag has been removed\n"+
			"           = no change for that tag   ! there was an error\n",
	)

	os.Exit(1)
}

func worker() {
	var blobs, candidates, permanodes []*blob.Ref
	var output bytes.Buffer

	for arg := range args {
		output.Reset()
		output.WriteString(arg)
		output.WriteString(" ")

		stat, err := os.Stat(arg)
		if err != nil {
			output.WriteString(fmt.Sprintln(err))
			goto finish
		}

		if stat.IsDir() {
			output.WriteString("DIRECTORY\n")
			goto finish
		}

		blobs, err = findBlobs(filepath.Base(arg), stat.Size())
		if err != nil {
			output.WriteString(fmt.Sprintln(err))
			goto finish
		}

		permanodes = nil
		for _, blob := range blobs {
			candidates, err = findPermanodes(blob)
			if err != nil {
				output.WriteString(fmt.Sprintln(err))
				goto finish
			}

			permanodes = append(permanodes, candidates...)
		}

		if len(permanodes) != 1 {
			output.WriteString(fmt.Sprintln(len(permanodes), "PERMANODES FOUND"))
			goto finish
		}

		output.WriteString(fmt.Sprintln(permanodes[0], doClaims(permanodes[0])))

	finish:
		fmt.Print(output.String())
	}

	wg.Done()

	return
}

func doSearch(cs *search.Constraint) (*search.SearchResult, error) {
	req := &search.SearchQuery{
		Constraint: cs,
	}

	return cl.Search(req)
}

func findBlobs(name string, size int64) (blobs []*blob.Ref, err error) {
	cs := &search.Constraint{
		File: &search.FileConstraint{
			FileName: &search.StringConstraint{
				Equals: name,
			},
			FileSize: &search.IntConstraint{
				Max: size,
				Min: size,
			},
		},
	}

	res, err := doSearch(cs)
	if err != nil {
		return
	}

	for _, br := range res.Blobs {
		blobs = append(blobs, &br.Blob)
	}

	return
}

func findPermanodes(blob *blob.Ref) (blobs []*blob.Ref, err error) {
	cs := &search.Constraint{
		Permanode: &search.PermanodeConstraint{
			Attr:  "camliContent",
			Value: blob.String(),
		},
	}

	res, err := doSearch(cs)
	if err != nil {
		return
	}

	for _, br := range res.Blobs {
		blobs = append(blobs, &br.Blob)
	}

	return
}

func getTags(blob *blob.Ref) ([]string, error) {
	req := &search.DescribeRequest{
		BlobRef: *blob,
	}

	res, err := cl.Describe(req)
	if err != nil {
		return nil, err
	}

	db := res.Meta.Get(*blob)
	if db == nil {
		return nil, errors.New("Nil blob")
	}

	if db.Permanode == nil {
		return nil, errors.New("Nil permanode")
	}

	return db.Permanode.Attr["tag"], nil
}

func hasKey(ary *[]string, key *string) bool {
	for _, str := range *ary {
		if str == *key {
			return true
		}
	}

	return false
}

func doClaims(blob *blob.Ref) string {
	var builder *schema.Builder
	var status bytes.Buffer

	itemTags, err := getTags(blob)
	if err != nil {
		return fmt.Sprint(err)
	}

	for _, tag := range tags {
		var err error
		var op string

		switch mode {
		case ADD:
			if hasKey(&itemTags, &tag) {
				op = "="
				goto finish
			}

			builder = schema.NewAddAttributeClaim(*blob, "tag", tag)
			op = "+"
		case DEL:
			if !hasKey(&itemTags, &tag) {
				op = "="
				goto finish
			}

			builder = schema.NewDelAttributeClaim(*blob, "tag", tag)
			op = "-"
		default:
			panic("not reachable")
		}

		_, err = cl.UploadAndSignBlob(builder)
		if err != nil {
			op = "!"
		}

	finish:
		status.WriteString(op)
		status.WriteString(tag)
		status.WriteString(" ")
	}

	return status.String()
}

// vim: ts=4 sw=4 sts=4
