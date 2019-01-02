package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/henvic/productreview/reviews"
	log "github.com/sirupsen/logrus"
)

func h(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		ErrorHandler(w, r, http.StatusMethodNotAllowed)
		return
	}

	if t := r.Header.Get("Content-Type"); strings.Contains(t, "application/json") {
		ErrorHandler(w, r, http.StatusNotAcceptable)
		return
	}

	var pr reviews.Review

	if err := json.NewDecoder(r.Body).Decode(&pr); err != nil {
		ErrorHandler(w, r, http.StatusBadRequest)
		return
	}

	id, err := reviews.Create(r.Context(), pr)

	if err != nil {
		if _, ok := err.(reviews.ValidationError); !ok {
			ErrorHandler(w, r, http.StatusInternalServerError)
			log.Error(err)
			return
		}

		ErrorHandler(w, r, http.StatusBadRequest, err.Error())
		return
	}

	resp := reviews.Response{
		Success:  true,
		ReviewID: id,
	}

	if err = json.NewEncoder(w).Encode(resp); err != nil {
		ErrorHandler(w, r, http.StatusInternalServerError)
		return
	}
}
