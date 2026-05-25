package handler

import (
	"net/http"
	"testing"
)

func TestURLHandler_handlePost(t *testing.T) {
	type fields struct {
		service URLService
	}
	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &URLHandler{
				service: tt.fields.service,
			}
			h.handlePost(tt.args.w, tt.args.r)
		})
	}
}
