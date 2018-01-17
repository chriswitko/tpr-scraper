package amp

import (
	"testing"
)

var testURLs = []string{
	"https://www.nytimes.com/2017/11/26/us/politics/john-conyers-steps-down-judiciary-committee-sexual-harassment.html?hp&action=click&pgtype=Homepage&clickSource=story-heading&module=first-column-region&region=top-news&WT.nav=top-news",
	"http://www.sport.pl/mundial/56,154361,22703018,kluby-z-najwyzszymi-srednimi-pensjami-zawodnikow-w-pierwszym.html#MTstream",
}

func TestParse(t *testing.T) {
	for _, u := range testURLs {
		links, err := parse(u)
		if err != nil {
			t.Errorf("%v", err)
		}
		if !links.Valid {
			t.Errorf("UNO canonical: %v, amphtml: %v", links.Canonical, links.AMP)
		}
	}
}

func TestValidate(t *testing.T) {
	for _, u := range testURLs {
		links, err := Validate(u)
		if err != nil {
			t.Fatalf("%v", err)
		}
		if !links.Valid {
			t.Fatalf("DUO canonical: %v, amphtml: %v", links.Canonical, links.AMP)
		}
	}
}
