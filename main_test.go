package main

import (
	"testing"
)

func TestCheckUrl( t *testing.T ) {
	url , err := checkUrl( "http://blog.94say.com/content/templates/huxiu/style/common.css" , "http://blog.94say.com" )
	if err != nil {
		t.Error( url )
	}

	url2 , err := checkUrl( "://blog.94say.com/content/templates/huxiu/style/common.css" , "http://blog.94say.com" )
	if url2 != "http://blog.94say.com/content/templates/huxiu/style/common.css" {
		t.Error( err )
	}

	url3 , err := checkUrl( "/content/templates/huxiu/style/common.css" , "http://blog.94say.com" )
	if url3 != "http://blog.94say.com/content/templates/huxiu/style/common.css" {
		t.Error( url3 )
	}

	url4 , err := checkUrl( "../templates/huxiu/style/common.css" , "http://blog.94say.com/content/templates" )
	if url4 != "http://blog.94say.com/templates/huxiu/style/common.css" {
		t.Error( url4 )
	}

	url5 , err := checkUrl( "data:image/png;base64,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" , "http://blog.94say.com/content/templates" )
	if url5 != "" {
		t.Error( url5 )
	}
}