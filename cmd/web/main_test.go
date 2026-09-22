package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestSimulationAPI(t *testing.T) {
	h := handler()
	call := func(input any) []bool {
		t.Helper()
		body, err := json.Marshal(input)
		if err != nil {
			t.Fatal(err)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/simulation", bytes.NewReader(body)))
		if w.Code != http.StatusOK {
			t.Fatalf("status %d: %s", w.Code, w.Body.String())
		}
		var state struct {
			Cells      []bool
			Population int
		}
		if err := json.Unmarshal(w.Body.Bytes(), &state); err != nil {
			t.Fatal(err)
		}
		if len(state.Cells) != 2500 || state.Population != 3 {
			t.Fatalf("unexpected state: length %d, population %d", len(state.Cells), state.Population)
		}
		return state.Cells
	}
	initial := call(map[string]any{"action": "reset", "pattern": "blinker"})
	next := call(map[string]any{"action": "step", "cells": initial})
	if reflect.DeepEqual(initial, next) {
		t.Fatal("blinker should change after one step")
	}
	final := call(map[string]any{"action": "step", "cells": next})
	if !reflect.DeepEqual(initial, final) {
		t.Fatal("blinker should repeat after two steps")
	}
}

func TestInvalidGrid(t *testing.T) {
	w := httptest.NewRecorder()
	handler().ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/simulation", bytes.NewBufferString(`{"action":"step","cells":[true]}`)))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestPage(t *testing.T) {
	w := httptest.NewRecorder()
	handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != http.StatusOK || !bytes.Contains(w.Body.Bytes(), []byte("<canvas")) {
		t.Fatal("page unavailable")
	}
}
