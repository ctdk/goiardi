package util

import (
	"testing"
	"net/http"
)

type testObj struct {
	Name string `json:"name"`
	UrlType string `json:"url_type"`
	Normal map[string]interface{} `json:"normal"`
	RunList []string `json:"run_list"`
}

func (to *testObj) GetName() string {
	return to.Name
}

func (to *testObj) URLType() string {
	return to.UrlType
}

// The strange URLs are because the config doesn't get parsed here, so it ends
// up using the really-really default settings.

func TestObjURL(t *testing.T){
	obj := &testObj{ Name: "foo", UrlType: "bar" }
	url := ObjURL(obj)
	expectedUrl := "http://:0/bar/foo"
	if url != expectedUrl {
		t.Errorf("expected %s, got %s", expectedUrl, url)
	}
}

func TestCustomObjUrl(t *testing.T){
	obj := &testObj{ Name: "foo", UrlType: "bar" }
	url := CustomObjURL(obj, "/baz")
	expectedUrl := "http://:0/bar/foo/baz"
	if url != expectedUrl {
		t.Errorf("expected %s, got %s", expectedUrl, url)
	}
}

func TestCustomURL(t *testing.T){
	initUrl := "/foo/bar"
	url := CustomURL(initUrl)
	expectedUrl := "http://:0/foo/bar"
	if url != expectedUrl {
		t.Errorf("expected %s, got %s", expectedUrl, url)
	}
	initUrl = "foo/bar"
	url = CustomURL(initUrl)
	if url != expectedUrl {
		t.Errorf("expected %s, got %s", expectedUrl, url)
	}
}

func TestGerror(t *testing.T){
	errmsg := "foo bar"
	err := Errorf(errmsg)
	if err.Error() != errmsg {
		t.Errorf("expected %s to match %s", err.Error(), errmsg)
	}
	if err.Status() != http.StatusBadRequest {
		t.Errorf("err.Status() did not return expected default")
	}
	err.SetStatus(http.StatusNotFound)
	if err.Status() != http.StatusNotFound {
		t.Errorf("SetStatus did not set Status correctly")
	}
}

func TestFlatten(t *testing.T){
	rl := []string{ "recipe[foo]", "role[bar]" }
	normmap := make(map[string]interface{})
	normmap["foo"] = "bar"
	normmap["baz"] = "buz"
	normmap["slice"] = []string{ "fee", "fie", "fo" }
	normmap["map"] = make(map[string]interface{})
	normmap["map"].(map[string]interface{})["first"] = "mook"
	normmap["map"].(map[string]interface{})["second"] = "nork"
	obj := &testObj{ Name: "foo", UrlType: "bar", RunList: rl, Normal: normmap }
	flattened := FlattenObj(obj)
	if _, ok := flattened["name"]; !ok {
		t.Errorf("obj name was not flattened correctly")
	}
	if flattened["name"].(string) != obj.Name {
		t.Errorf("flattened name not correct, wanted %s got %v", obj.Name, flattened["name"])
	}
	if _, ok := flattened["foo"]; !ok {
		t.Errorf("Foo should have been set, but it wasn't")
	}
	if _, ok := flattened["normal"]; ok {
		t.Errorf("The 'normal' field was set, but shouldn't have been.")
	}
	if _, ok := flattened["map_first"]; !ok {
		t.Errorf("normal -> map -> second should have been flattened to map_second, but it wasn't")
	}
	if r, ok := flattened["recipe"]; ok {
		if r.([]string)[0] != "foo" {
			t.Errorf("recipe list should have included foo, but it had %v instead", r.([]string)[0])
		}
	} else {
		t.Errorf("No recipe list")
	}
	if r, ok := flattened["role"]; ok {
		if r.([]string)[0] != "bar" {
			t.Errorf("role list should have included bar, but it had %v instead", r.([]string)[0])
		}
	} else {
		t.Errorf("No role list")
	}
}

func TestMapify(t *testing.T){

}

func TestIndexify(t *testing.T){

}
