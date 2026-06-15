package main

import (
	"testing"

	"github.com/ctdk/goiardi/pedant"
)

// --- Depsolver Tests ---

func TestDepsolverEmptyPayload(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("depsolv_env")
	env := pedant.NewEnvironment(envName, map[string]interface{}{
		"cookbook_versions": map[string]string{
			"qux": "> 4.0.0",
			"foo": ">= 0.1.0",
			"bar": "< 4.0.0",
		},
	})
	defer client.Delete("/environments/" + envName)

	resp, err := client.Post("/environments", env)
	if err != nil {
		t.Fatalf("POST /environments: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Empty payload
	resp, err = client.Post("/environments/"+envName+"/cookbook_versions", map[string]interface{}{})
	if err != nil {
		t.Fatalf("POST /environments/%s/cookbook_versions: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 405)
}

func TestDepsolverEmptyRunList(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("depsolv_empty")
	env := pedant.NewEnvironment(envName, map[string]interface{}{
		"cookbook_versions": map[string]string{
			"foo": ">= 0.1.0",
		},
	})
	defer client.Delete("/environments/" + envName)

	resp, err := client.Post("/environments", env)
	if err != nil {
		t.Fatalf("POST /environments: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Empty run_list
	payload := map[string]interface{}{
		"run_list": []string{},
	}
	resp, err = client.Post("/environments/"+envName+"/cookbook_versions", payload)
	if err != nil {
		t.Fatalf("POST /environments/%s/cookbook_versions: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	if len(body) != 0 {
		t.Errorf("expected empty response for empty run_list, got %d entries", len(body))
	}
}

func TestDepsolverNonExistentCookbook(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("depsolv_missing")
	env := pedant.NewEnvironment(envName, map[string]interface{}{
		"cookbook_versions": map[string]string{
			"foo": ">= 0.1.0",
		},
	})
	defer client.Delete("/environments/" + envName)

	resp, err := client.Post("/environments", env)
	if err != nil {
		t.Fatalf("POST /environments: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	payload := map[string]interface{}{
		"run_list": []string{"this_does_not_exist"},
	}
	resp, err = client.Post("/environments/"+envName+"/cookbook_versions", payload)
	if err != nil {
		t.Fatalf("POST /environments/%s/cookbook_versions: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 412)
}

func TestDepsolverNonExistentCookbookDefaultEnv(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)

	payload := map[string]interface{}{
		"run_list": []string{"this_does_not_exist"},
	}
	resp, err := client.Post("/environments/_default/cookbook_versions", payload)
	if err != nil {
		t.Fatalf("POST /environments/_default/cookbook_versions: %v", err)
	}
	pedant.AssertStatus(t, resp, 412)
}

func TestDepsolverExistingCookbook(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("depsolv_cb")
	cbVersion := "1.2.3"
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	envName := pedant.UniqueName("depsolv_good")
	env := pedant.NewEnvironment(envName, map[string]interface{}{
		"cookbook_versions": map[string]string{
			cbName: ">= 0.1.0",
		},
	})
	defer client.Delete("/environments/" + envName)

	resp, err = client.Post("/environments", env)
	if err != nil {
		t.Fatalf("POST /environments: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	depsPayload := map[string]interface{}{
		"run_list": []string{cbName},
	}
	resp, err = client.Post("/environments/"+envName+"/cookbook_versions", depsPayload)
	if err != nil {
		t.Fatalf("POST /environments/%s/cookbook_versions: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	cbInfo, ok := body[cbName].(map[string]interface{})
	if !ok {
		t.Fatalf("expected cookbook %q in response, got: %v", cbName, body)
	}
	if cbInfo["version"] != cbVersion {
		t.Errorf("expected version %q, got %v", cbVersion, cbInfo["version"])
	}
}

func TestDepsolverExistingCookbookDefaultEnv(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("depsolv_def")
	cbVersion := "1.2.3"
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	depsPayload := map[string]interface{}{
		"run_list": []string{cbName},
	}
	resp, err = client.Post("/environments/_default/cookbook_versions", depsPayload)
	if err != nil {
		t.Fatalf("POST /environments/_default/cookbook_versions: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	cbInfo, ok := body[cbName].(map[string]interface{})
	if !ok {
		t.Fatalf("expected cookbook %q in response, got: %v", cbName, body)
	}
	if cbInfo["version"] != cbVersion {
		t.Errorf("expected version %q, got %v", cbVersion, cbInfo["version"])
	}
}

func TestDepsolverNonExistentVersion(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("depsolv_ver")
	cbVersion := "1.2.3"
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Request a version that doesn't exist
	depsPayload := map[string]interface{}{
		"run_list": []string{cbName + "@2.0.0"},
	}
	resp, err = client.Post("/environments/_default/cookbook_versions", depsPayload)
	if err != nil {
		t.Fatalf("POST /environments/_default/cookbook_versions: %v", err)
	}
	pedant.AssertStatus(t, resp, 412)
}

func TestDepsolverFilteredByEnvironment(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("depsolv_filter")
	cbVersion := "1.2.3"
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Environment that requires a version that doesn't exist
	envName := pedant.UniqueName("depsolv_filter_env")
	env := pedant.NewEnvironment(envName, map[string]interface{}{
		"cookbook_versions": map[string]string{
			cbName: "= 400.0.0",
		},
	})
	defer client.Delete("/environments/" + envName)

	resp, err = client.Post("/environments", env)
	if err != nil {
		t.Fatalf("POST /environments: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	depsPayload := map[string]interface{}{
		"run_list": []string{cbName},
	}
	resp, err = client.Post("/environments/"+envName+"/cookbook_versions", depsPayload)
	if err != nil {
		t.Fatalf("POST /environments/%s/cookbook_versions: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 412)
}

func TestDepsolverMultipleNonExistent(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("depsolv_multi")
	env := pedant.NewEnvironment(envName, map[string]interface{}{
		"cookbook_versions": map[string]string{
			"foo": ">= 0.1.0",
		},
	})
	defer client.Delete("/environments/" + envName)

	resp, err := client.Post("/environments", env)
	if err != nil {
		t.Fatalf("POST /environments: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	depsPayload := map[string]interface{}{
		"run_list": []string{"this_does_not_exist", "also_this_one"},
	}
	resp, err = client.Post("/environments/"+envName+"/cookbook_versions", depsPayload)
	if err != nil {
		t.Fatalf("POST /environments/%s/cookbook_versions: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 412)
}

func TestDepsolverInvalidEnvironment(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	depsPayload := map[string]interface{}{
		"run_list": []string{},
	}
	resp, err := client.Post("/environments/not@environment/cookbook_versions", depsPayload)
	if err != nil {
		t.Fatalf("POST /environments/not@environment/cookbook_versions: %v", err)
	}
	pedant.AssertStatus(t, resp, 404)
}

func TestDepsolverNonArrayRunList(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("depsolv_narray")
	env := pedant.NewEnvironment(envName)
	defer client.Delete("/environments/" + envName)

	resp, err := client.Post("/environments", env)
	if err != nil {
		t.Fatalf("POST /environments: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	// run_list as string instead of array
	payload := map[string]interface{}{
		"run_list": "foo",
	}
	resp, err = client.Post("/environments/"+envName+"/cookbook_versions", payload)
	if err != nil {
		t.Fatalf("POST /environments/%s/cookbook_versions: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 400)
}

func TestDepsolverIntInRunList(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("depsolv_int")
	env := pedant.NewEnvironment(envName)
	defer client.Delete("/environments/" + envName)

	resp, err := client.Post("/environments", env)
	if err != nil {
		t.Fatalf("POST /environments: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	// run_list with int instead of string
	payload := map[string]interface{}{
		"run_list": []interface{}{12},
	}
	resp, err = client.Post("/environments/"+envName+"/cookbook_versions", payload)
	if err != nil {
		t.Fatalf("POST /environments/%s/cookbook_versions: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 400)
}

func TestDepsolverWithDependencies(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbFoo := pedant.UniqueName("depsolv_foo")
	cbBar := pedant.UniqueName("depsolv_bar")

	// bar depends on foo
	payloadFoo := newCookbookPayload(cbFoo, "1.0.0")
	payloadBar := newCookbookPayload(cbBar, "2.0.0", map[string]interface{}{
		"metadata": map[string]interface{}{
			"version":      "2.0.0",
			"name":         cbBar,
			"dependencies": map[string]string{cbFoo: "> 0.0.0"},
		},
	})

	defer func() {
		client.Delete("/cookbooks/" + cbFoo + "/1.0.0")
		client.Delete("/cookbooks/" + cbBar + "/2.0.0")
	}()

	resp, err := client.Put("/cookbooks/"+cbFoo+"/1.0.0", payloadFoo)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/1.0.0: %v", cbFoo, err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.Put("/cookbooks/"+cbBar+"/2.0.0", payloadBar)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/2.0.0: %v", cbBar, err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Resolve dependencies for bar
	depsPayload := map[string]interface{}{
		"run_list": []string{cbBar},
	}
	resp, err = client.Post("/environments/_default/cookbook_versions", depsPayload)
	if err != nil {
		t.Fatalf("POST /environments/_default/cookbook_versions: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)

	// Both cookbooks should be in the response
	if _, ok := body[cbFoo]; !ok {
		t.Errorf("expected cookbook %q in depsolver response, got: %v", cbFoo, body)
	}
	if _, ok := body[cbBar]; !ok {
		t.Errorf("expected cookbook %q in depsolver response, got: %v", cbBar, body)
	}
}

func TestDepsolverMissingDependency(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("depsolv_missdep")

	// Cookbook with a dependency that doesn't exist
	payload := newCookbookPayload(cbName, "1.2.3", map[string]interface{}{
		"metadata": map[string]interface{}{
			"version":      "1.2.3",
			"name":         cbName,
			"dependencies": map[string]string{"this_does_not_exist": ">= 0.0.0"},
		},
	})
	defer client.Delete("/cookbooks/" + cbName + "/1.2.3")

	resp, err := client.Put("/cookbooks/"+cbName+"/1.2.3", payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/1.2.3: %v", cbName, err)
	}
	pedant.AssertStatus(t, resp, 201)

	depsPayload := map[string]interface{}{
		"run_list": []string{cbName},
	}
	resp, err = client.Post("/environments/_default/cookbook_versions", depsPayload)
	if err != nil {
		t.Fatalf("POST /environments/_default/cookbook_versions: %v", err)
	}
	pedant.AssertStatus(t, resp, 412)
}

func TestDepsolverImpossibleDependency(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbFoo := pedant.UniqueName("depsolv_imp_foo")
	cbBar := pedant.UniqueName("depsolv_imp_bar")

	// foo depends on bar > 2.0.0, bar depends on foo > 3.0.0
	// This creates a circular impossible constraint
	payloadFoo := newCookbookPayload(cbFoo, "1.2.3", map[string]interface{}{
		"metadata": map[string]interface{}{
			"version":      "1.2.3",
			"name":         cbFoo,
			"dependencies": map[string]string{cbBar: "> 2.0.0"},
		},
	})
	payloadBar := newCookbookPayload(cbBar, "2.0.0", map[string]interface{}{
		"metadata": map[string]interface{}{
			"version":      "2.0.0",
			"name":         cbBar,
			"dependencies": map[string]string{cbFoo: "> 3.0.0"},
		},
	})

	defer func() {
		client.Delete("/cookbooks/" + cbFoo + "/1.2.3")
		client.Delete("/cookbooks/" + cbBar + "/2.0.0")
	}()

	resp, err := client.Put("/cookbooks/"+cbFoo+"/1.2.3", payloadFoo)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/1.2.3: %v", cbFoo, err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.Put("/cookbooks/"+cbBar+"/2.0.0", payloadBar)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/2.0.0: %v", cbBar, err)
	}
	pedant.AssertStatus(t, resp, 201)

	depsPayload := map[string]interface{}{
		"run_list": []string{cbFoo},
	}
	resp, err = client.Post("/environments/_default/cookbook_versions", depsPayload)
	if err != nil {
		t.Fatalf("POST /environments/_default/cookbook_versions: %v", err)
	}
	pedant.AssertStatus(t, resp, 412)
}

func TestDepsolverDatestampVersion(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("depsolv_ds")
	cbVersion := "1.2.20130730201745"
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	envName := pedant.UniqueName("depsolv_ds_env")
	env := pedant.NewEnvironment(envName, map[string]interface{}{
		"cookbook_versions": map[string]string{
			cbName: ">= 1.2.20130730200000",
		},
	})
	defer client.Delete("/environments/" + envName)

	resp, err = client.Post("/environments", env)
	if err != nil {
		t.Fatalf("POST /environments: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	depsPayload := map[string]interface{}{
		"run_list": []string{cbName},
	}
	resp, err = client.Post("/environments/"+envName+"/cookbook_versions", depsPayload)
	if err != nil {
		t.Fatalf("POST /environments/%s/cookbook_versions: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	if _, ok := body[cbName]; !ok {
		t.Errorf("expected cookbook %q in depsolver response, got: %v", cbName, body)
	}
}
