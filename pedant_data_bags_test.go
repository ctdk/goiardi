package main

import (
	"fmt"
	"github.com/ctdk/goiardi/pedant"
	"testing"
)

func TestDataBagsListEmpty(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	resp, err := client.GetOrg("/data")
	if err != nil {
		t.Fatalf("GET /data: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	if len(body) != 0 {
		t.Errorf("expected empty data bag list, got %d entries", len(body))
	}
}

func TestDataBagsCreateAndRead(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	bagName := pedant.UniqueName("test_bag")
	bag := pedant.NewDataBag(bagName)
	defer client.DeleteOrg("/data/" + bagName)

	resp, err := client.PostOrg("/data", bag)
	if err != nil {
		t.Fatalf("POST /data: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)
	pedant.AssertBodyContains(t, resp, "/data/"+bagName)

	resp, err = client.GetOrg("/data/" + bagName)
	if err != nil {
		t.Fatalf("GET /data/%s: %v", bagName, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestDataBagsCreateDuplicate(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	bagName := pedant.UniqueName("dup_bag")
	bag := pedant.NewDataBag(bagName)
	defer client.DeleteOrg("/data/" + bagName)

	resp, err := client.PostOrg("/data", bag)
	if err != nil {
		t.Fatalf("first POST /data: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.PostOrg("/data", bag)
	if err != nil {
		t.Fatalf("second POST /data: %v", err)
	}
	pedant.AssertErrorResponse(t, resp, 409, "already exists")
}

func TestDataBagsDelete(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	bagName := pedant.UniqueName("del_bag")
	bag := pedant.NewDataBag(bagName)

	resp, err := client.PostOrg("/data", bag)
	if err != nil {
		t.Fatalf("POST /data: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.DeleteOrg("/data/" + bagName)
	if err != nil {
		t.Fatalf("DELETE /data/%s: %v", bagName, err)
	}
	pedant.AssertStatus(t, resp, 200)

	resp, err = client.GetOrg("/data/" + bagName)
	if err != nil {
		t.Fatalf("GET /data/%s: %v", bagName, err)
	}
	pedant.AssertStatus(t, resp, 404)
}

func TestDataBagsNameValidation(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)

	tests := []struct {
		name  string
		valid bool
	}{
		{"pedant", true},
		{"pedant-bag", true},
		{"pedant_bag", true},
		{"pedant_bag-foo", true},
		{"1234567890", true},
		{"pedant99", true},
		{"pedant:with:colons", true},
		{"pedant.with.dots", true},
		{"pedant_badName!!$$$$_oh_very+bad", false},
		{"pedant-does-not-like-punctuation!!!!", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bag := pedant.NewDataBag(tt.name)
			resp, err := client.PostOrg("/data", bag)
			if err != nil {
				t.Fatalf("POST /data: %v", err)
			}
			if tt.valid {
				pedant.AssertStatus(t, resp, 201)
				client.DeleteOrg("/data/" + tt.name)
			} else {
				pedant.AssertStatus(t, resp, 400)
			}
		})
	}
}

func TestDataBagItemsCreateAndRead(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	bagName := pedant.UniqueName("item_bag")
	bag := pedant.NewDataBag(bagName)
	defer client.DeleteOrg("/data/" + bagName)

	resp, err := client.PostOrg("/data", bag)
	if err != nil {
		t.Fatalf("POST /data: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Create item
	itemID := pedant.UniqueName("item")
	item := pedant.NewDataBagItem(itemID, map[string]interface{}{"answer": float64(42)})
	defer client.DeleteOrg("/data/" + bagName + "/" + itemID)

	resp, err = client.PostOrg("/data/"+bagName, item)
	if err != nil {
		t.Fatalf("POST /data/%s: %v", bagName, err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Read item
	resp, err = client.GetOrg("/data/" + bagName + "/" + itemID)
	if err != nil {
		t.Fatalf("GET /data/%s/%s: %v", bagName, itemID, err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	if body["id"] != itemID {
		t.Errorf("expected id %q, got %q", itemID, body["id"])
	}
	if body["answer"] != float64(42) {
		t.Errorf("expected answer 42, got %v", body["answer"])
	}
}

func TestDataBagItemsUpdate(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	bagName := pedant.UniqueName("upd_item_bag")
	bag := pedant.NewDataBag(bagName)
	defer client.DeleteOrg("/data/" + bagName)

	resp, err := client.PostOrg("/data", bag)
	if err != nil {
		t.Fatalf("POST /data: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	itemID := pedant.UniqueName("upd_item")
	item := pedant.NewDataBagItem(itemID, map[string]interface{}{"value": "original"})
	defer client.DeleteOrg("/data/" + bagName + "/" + itemID)

	resp, err = client.PostOrg("/data/"+bagName, item)
	if err != nil {
		t.Fatalf("POST /data/%s: %v", bagName, err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Update
	updated := pedant.NewDataBagItem(itemID, map[string]interface{}{"value": "updated"})
	resp, err = client.PutOrg("/data/"+bagName+"/"+itemID, updated)
	if err != nil {
		t.Fatalf("PUT /data/%s/%s: %v", bagName, itemID, err)
	}
	pedant.AssertStatus(t, resp, 200)

	// Verify
	resp, err = client.GetOrg("/data/" + bagName + "/" + itemID)
	if err != nil {
		t.Fatalf("GET /data/%s/%s: %v", bagName, itemID, err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	if body["value"] != "updated" {
		t.Errorf("expected value 'updated', got %v", body["value"])
	}
}

func TestDataBagItemsDelete(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	bagName := pedant.UniqueName("del_item_bag")
	bag := pedant.NewDataBag(bagName)
	defer client.DeleteOrg("/data/" + bagName)

	resp, err := client.PostOrg("/data", bag)
	if err != nil {
		t.Fatalf("POST /data: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	itemID := pedant.UniqueName("del_item")
	item := pedant.NewDataBagItem(itemID)

	resp, err = client.PostOrg("/data/"+bagName, item)
	if err != nil {
		t.Fatalf("POST /data/%s: %v", bagName, err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.DeleteOrg("/data/" + bagName + "/" + itemID)
	if err != nil {
		t.Fatalf("DELETE /data/%s/%s: %v", bagName, itemID, err)
	}
	pedant.AssertStatus(t, resp, 200)

	resp, err = client.GetOrg("/data/" + bagName + "/" + itemID)
	if err != nil {
		t.Fatalf("GET /data/%s/%s: %v", bagName, itemID, err)
	}
	pedant.AssertStatus(t, resp, 404)
}

func TestDataBagItemsNoID(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	bagName := pedant.UniqueName("noid_bag")
	bag := pedant.NewDataBag(bagName)
	defer client.DeleteOrg("/data/" + bagName)

	resp, err := client.PostOrg("/data", bag)
	if err != nil {
		t.Fatalf("POST /data: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Item without ID
	item := map[string]interface{}{"answer": float64(42)}
	resp, err = client.PostOrg("/data/"+bagName, item)
	if err != nil {
		t.Fatalf("POST /data/%s: %v", bagName, err)
	}
	pedant.AssertStatus(t, resp, 400)
}

func TestDataBagItemsInvalidID(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	bagName := pedant.UniqueName("badid_bag")
	bag := pedant.NewDataBag(bagName)
	defer client.DeleteOrg("/data/" + bagName)

	resp, err := client.PostOrg("/data", bag)
	if err != nil {
		t.Fatalf("POST /data: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	invalidIDs := []string{"pedant_badId!!", "^$@^*  pedant"}
	for _, id := range invalidIDs {
		t.Run(id, func(t *testing.T) {
			item := pedant.NewDataBagItem(id)
			resp, err := client.PostOrg("/data/"+bagName, item)
			if err != nil {
				t.Fatalf("POST /data/%s: %v", bagName, err)
			}
			pedant.AssertStatus(t, resp, 400)
		})
	}
}

func TestDataBagDeleteBagWithItems(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	bagName := pedant.UniqueName("del_bag_items")
	bag := pedant.NewDataBag(bagName)

	resp, err := client.PostOrg("/data", bag)
	if err != nil {
		t.Fatalf("POST /data: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Create items
	for i := 0; i < 3; i++ {
		itemID := fmt.Sprintf("item_%d", i)
		item := pedant.NewDataBagItem(itemID)
		resp, err := client.PostOrg("/data/"+bagName, item)
		if err != nil {
			t.Fatalf("POST /data/%s: %v", bagName, err)
		}
		pedant.AssertStatus(t, resp, 201)
	}

	// Delete the bag (should delete all items)
	resp, err = client.DeleteOrg("/data/" + bagName)
	if err != nil {
		t.Fatalf("DELETE /data/%s: %v", bagName, err)
	}
	pedant.AssertStatus(t, resp, 200)

	// Verify bag is gone
	resp, err = client.GetOrg("/data/" + bagName)
	if err != nil {
		t.Fatalf("GET /data/%s: %v", bagName, err)
	}
	pedant.AssertStatus(t, resp, 404)
}

// --- Client tests ---
