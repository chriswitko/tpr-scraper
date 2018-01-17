package amp

import (
	"bytes"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
	xmlpath "gopkg.in/xmlpath.v2"
)

// TODO: Scrap OpenGraph data with image, if image availbe use fastimage to detect full info about image

var (
	canonical      = xmlpath.MustCompile("/html/head/link[@rel='canonical']/@href")
	amphtml        = xmlpath.MustCompile("/html/head/link[@rel='amphtml']/@href")
	imagesrc       = xmlpath.MustCompile("/html/*/meta[@property='og:image']/@content")
	imagesrcurl    = xmlpath.MustCompile("/html/*/meta[@property='og:image:url']/@content")
	imagesecureurl = xmlpath.MustCompile("/html/*/meta[@property='og:image:secure_url']/@content")
	imagewidth     = xmlpath.MustCompile("/html/*/meta[@property='og:image:width']/@content")
	imageheight    = xmlpath.MustCompile("/html/*/meta[@property='og:image:height']/@content")
	locale         = xmlpath.MustCompile("/html/*/meta[@property='og:locale']/@content")
)

// Links basic structure
type Links struct {
	Canonical   string
	AMP         string
	Image       string
	ImageWidth  int64
	ImageHeight int64
	Locale      string
	Valid       bool
}

// Validate link relations.
func Validate(urlStr string) (*Links, error) {
	src, err := Parse(urlStr)
	if err != nil {
		return src, err
	}
	l1, err := Parse(src.Canonical)
	if err != nil {
		return l1, err
	}
	l2, err := Parse(src.AMP)
	if err != nil {
		return l2, err
	}
	l3, err := Parse(src.Image)
	if err != nil {
		return l3, err
	}
	valid := reflect.DeepEqual(l1, l2)
	src.Valid = valid
	return src, nil
}

// Parse required links
func Parse(urlStr string) (*Links, error) {
	timeout := time.Duration(5 * time.Second)
	client := http.Client{
		Timeout: timeout,
	}
	resp, err := client.Get(urlStr)
	if err != nil {
		return nil, err
	}
	srcRoot, err := html.Parse(resp.Body)
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	html.Render(&b, srcRoot)
	fixedHTML := b.String()
	// fmt.Println("fixedHTML", fixedHTML)
	reader := strings.NewReader(fixedHTML)
	root, err := xmlpath.ParseHTML(reader)
	// fmt.Println("root", root)
	if err != nil {
		fmt.Println("my err", err)
		return nil, err
	}
	// fmt.Println("srcRoot", srcRoot)
	links := Links{}
	if canonicalURL, ok := canonical.String(root); ok {
		links.Canonical = canonicalURL
	}
	if ampURL, ok := amphtml.String(root); ok {
		links.AMP = ampURL
		links.Valid = true
	}
	if imageSrc, ok := imagesrc.String(root); ok {
		links.Image = imageSrc
	}
	if links.Image == "" {
		if imageSrcURL, ok := imagesrcurl.String(root); ok {
			links.Image = imageSrcURL
		}
		if imageSecureURL, ok := imagesecureurl.String(root); ok {
			links.Image = imageSecureURL
		}
	}
	if imageWidth, ok := imagewidth.String(root); ok {
		links.ImageWidth, _ = strconv.ParseInt(imageWidth, 0, 64)
	}
	if imageHeight, ok := imageheight.String(root); ok {
		links.ImageHeight, _ = strconv.ParseInt(imageHeight, 0, 64)
	}
	if locale, ok := locale.String(root); ok {
		links.Locale = locale
	}

	return &links, nil
}
