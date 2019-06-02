package main_test

import (
	"net/http"
	"encoding/json"
	"bytes"
	"testing"
	"fmt"
)

type Order struct {
	Id int64 `json:"id"`
	Distance int64 `json:"distance"`
	Status string `json:"status"`
}

type ErrorMessage struct {
	Error string `json:"error"`
}

var end_point = "http://localhost:8080/orders"

func TestPlaceOrder(t *testing.T) {
	payload := map[string][2]string{"origin": [2]string{"25.0330", "121.5654"}, "destination": [2]string{"22.9997", "120.2270"}}
	params, _ := json.Marshal(payload)
	resp, _ := http.Post(end_point, "application/json", bytes.NewBuffer(params))
	if resp.StatusCode != 200 {
		t.Errorf("Fail: Place an order.")
	}
}

func TestMissingArguments(t *testing.T) {
	payload := map[string][2]string{"origin": [2]string{"25.0330", "121.5654"}}
	params, _ := json.Marshal(payload)
	resp, _ := http.Post(end_point, "application/json", bytes.NewBuffer(params))
	var message ErrorMessage
	json.NewDecoder(resp.Body).Decode(&message)
	if message.Error != "Missing required arguments." {
		t.Errorf("Fail: Handle missing arguments.")
	}
}

func TestWrongArguments1(t *testing.T) {
	payload := map[string][2]string{"origin": [2]string{"0.0.0", "121.5654"}, "destination": [2]string{"22.9997", "120.2270"}}
	params, _ := json.Marshal(payload)
	resp, _ := http.Post(end_point, "application/json", bytes.NewBuffer(params))
	var message ErrorMessage
	json.NewDecoder(resp.Body).Decode(&message)
	if message.Error != "Invalid coordinates." {
		t.Errorf("Fail: Handle wrong arguments.")
	}
}

func TestWrongArguments2(t *testing.T) {
	payload := map[string][2]string{"origin": [2]string{"0", "0"}, "destination": [2]string{"22.9997", "120.2270"}}
	params, _ := json.Marshal(payload)
	resp, _ := http.Post(end_point, "application/json", bytes.NewBuffer(params))
	var message ErrorMessage
	json.NewDecoder(resp.Body).Decode(&message)
	if message.Error != "Can't get the distance." {
		t.Errorf("Fail: Handle wrong arguments.")
	}
}

func TestWrongArguments3(t *testing.T) {
	payload := map[string][1]string{"origin": [1]string{"0"}, "destination": [1]string{"0"}}
	params, _ := json.Marshal(payload)
	resp, _ := http.Post(end_point, "application/json", bytes.NewBuffer(params))
	var message ErrorMessage
	json.NewDecoder(resp.Body).Decode(&message)
	if message.Error != "Invalid coordinates." {
		t.Errorf("Fail: Handle wrong arguments.")
	}
}

func TestArgumentRanges(t *testing.T) {
	payload := map[string][2]string{"origin": [2]string{"121.5654", "25.0330"}, "destination": [2]string{"22.9997", "120.2270"}}
	params, _ := json.Marshal(payload)
	resp, _ := http.Post(end_point, "application/json", bytes.NewBuffer(params))
	var message ErrorMessage
	json.NewDecoder(resp.Body).Decode(&message)
	if message.Error != "Invalid coordinates." {
		t.Errorf("Fail: Handle value ranges of arguments")
	}
}

func TestTakeOrder(t *testing.T) {
	client := &http.Client{}
	payload := map[string]string{"status": "TAKEN"}
	params, _ := json.Marshal(payload)
	r, _ := http.NewRequest("PATCH", fmt.Sprintf("%s/1", end_point), bytes.NewBuffer(params))
	r.Header.Set("Content-Type", "application/json")
	resp, _ := client.Do(r)
	if resp.StatusCode != 200 {
		t.Errorf("Fail: Take an order.")
	}
}

func TestTakeOrderHasBeenTaken(t *testing.T) {
	client := &http.Client{}
	payload := map[string]string{"status": "TAKEN"}
	params, _ := json.Marshal(payload)
	r, _ := http.NewRequest("PATCH", fmt.Sprintf("%s/1", end_point), bytes.NewBuffer(params))
	r.Header.Set("Content-Type", "application/json")
	resp, _ := client.Do(r)
	var message ErrorMessage
	json.NewDecoder(resp.Body).Decode(&message)
	if message.Error != "Oops! The order has been taken." {
		t.Errorf("Fail: Handle order which is taken already.")
	}
}

func TestTakeOrderNotExist(t *testing.T) {
	client := &http.Client{}
	payload := map[string]string{"status": "TAKEN"}
	params, _ := json.Marshal(payload)
	r, _ := http.NewRequest("PATCH", fmt.Sprintf("%s/3", end_point), bytes.NewBuffer(params))
	r.Header.Set("Content-Type", "application/json")
	resp, _ := client.Do(r)
	var message ErrorMessage
	json.NewDecoder(resp.Body).Decode(&message)
	if message.Error != "The order doesn't exist." {
		t.Errorf("Fail: Handle order not exist.")
	}
}

func TestGetOrderList(t *testing.T) {
	// Place more orders
	for i := 0; i < 5; i++ {
		payload := map[string][2]string{"origin": [2]string{"25.0330", "121.5654"}, "destination": [2]string{"22.9997", "120.2270"}}
		params, _ := json.Marshal(payload)
		resp, _ := http.Post(end_point, "application/json", bytes.NewBuffer(params))
		if resp.StatusCode != 200 {
			t.Errorf("Fail: Place an order.")
		}
	}

	client := &http.Client{}
	page, limit := 0, 5
	r, _ := http.NewRequest("GET", fmt.Sprintf("%s?page=%d&limit=%d", end_point, page, limit), nil)
	r.Header.Set("Content-Type", "application/json")
	resp, _ := client.Do(r)
	var rows []Order
	json.NewDecoder(resp.Body).Decode(&rows)
	if len(rows)!=5 {
		t.Errorf("Fail: Get the right number of rows.")
	}
}

func TestInvalidPage(t *testing.T) {
	client := &http.Client{}
	page, limit := -1, 5
	r, _ := http.NewRequest("GET", fmt.Sprintf("%s?page=%d&limit=%d", end_point, page, limit), nil)
	r.Header.Set("Content-Type", "application/json")
	resp, _ := client.Do(r)
	var message ErrorMessage
	json.NewDecoder(resp.Body).Decode(&message)
	if message.Error != "Invalid values." {
		t.Errorf("Fail: Handle wrong value of page.")
	}
}

func TestInvalidLimit(t *testing.T) {
	client := &http.Client{}
	page, limit := 0, 0
	r, _ := http.NewRequest("GET", fmt.Sprintf("%s?page=%d&limit=%d", end_point, page, limit), nil)
	r.Header.Set("Content-Type", "application/json")
	resp, _ := client.Do(r)
	var message ErrorMessage
	json.NewDecoder(resp.Body).Decode(&message)
	if message.Error != "Invalid values." {
		t.Errorf("Fail: Handle wrong value of page.")
	}
}