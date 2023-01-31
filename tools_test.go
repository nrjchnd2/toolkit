package toolkit

import "testing"

func TestTools_RandomString(t *testing.T) {
	var testTools Tools
	if s := testTools.RandomString(10); len(s) != 10 {
		t.Error("Wrong length returned.")
	}

}