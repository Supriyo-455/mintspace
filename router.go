package main

import (
	"fmt"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

type routerFunc func(w http.ResponseWriter, r *http.Request, p httprouter.Params) error

func makeRouterHandleFunc(f routerFunc) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		if err := f(w, r, p); err != nil {
			// handle error
			// log the error
			print(err)
		}
	}
}

type Router struct {
	mux *httprouter.Router
}

func NewRouter() *Router {
	router := new(Router)
	router.mux = httprouter.New()

	router.mux.GET("/", makeRouterHandleFunc(index))

	return router
}

func index(res http.ResponseWriter, req *http.Request, _ httprouter.Params) error {
	fmt.Fprintf(res, "My blogging app!!")
	return nil
}
