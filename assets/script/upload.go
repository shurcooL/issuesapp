package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gopherjs/gopherjs/js"
	"honnef.co/go/js/dom"
)

const (
	//host = "http://virtivia.com:27080"
	host = "http://localhost:27080"
)

func init() {
	document.AddEventListener("DOMContentLoaded", false, func(_ dom.Event) { setup() })
}

func setup() {
	textArea := document.GetElementByID("comment-editor").(*dom.HTMLTextAreaElement)

	textArea.AddEventListener("paste", false, func(e dom.Event) {
		ce := e.(*dom.ClipboardEvent)

		// Get a File.
		items := ce.Get("clipboardData").Get("items")
		if items.Length() == 0 {
			return
		}
		item := items.Index(0)
		js.Global.Get("console").Call("log", item)
		if item.Get("kind").String() != "file" {
			return
		}
		if item.Get("type").String() != "image/png" {
			return
		}
		f := item.Call("getAsFile")
		js.Global.Get("console").Call("log", f)

		go func() {
			body := blobToBytes(f)
			fmt.Println("got here:", len(body))

			resp, err := http.Get(host + "/api/getfilename?ext=" + "png") // TODO.
			if err != nil {
				log.Println(err)
				return
			}
			defer resp.Body.Close()
			filename, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Println(err)
				return
			}

			url := host + "/" + string(filename)
			fmt.Println(url)
			insertText(textArea, "![Image]("+url+")\n")

			req, err := http.NewRequest("PUT", url, bytes.NewReader(body))
			if err != nil {
				log.Println(err)
				return
			}
			req.Header.Set("Content-Type", "application/octet-stream")
			resp, err = http.DefaultClient.Do(req)
			if err != nil {
				log.Println(err)
				return
			}
			_ = resp.Body.Close()
			fmt.Println("done")
		}()
	})
}

func insertText(t *dom.HTMLTextAreaElement, inserted string) {
	value, start, end := t.Value, t.SelectionStart, t.SelectionEnd
	t.Value = value[:start] + inserted + value[end:]
	t.SelectionStart, t.SelectionEnd = start+len(inserted), start+len(inserted)
}

// blobToBytes converts a Blob to []byte.
func blobToBytes(blob *js.Object) []byte {
	var b = make(chan []byte)
	fileReader := js.Global.Get("FileReader").New()
	fileReader.Set("onload", func(e *js.Object) {
		b <- js.Global.Get("Uint8Array").New(e.Get("target").Get("result")).Interface().([]byte)
	})
	fileReader.Call("readAsArrayBuffer", blob)
	return <-b
}
