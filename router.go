package main

import (
	"net/http"
	"text/template"

	"github.com/julienschmidt/httprouter"
)

type routerFunc func(res http.ResponseWriter, req *http.Request, params httprouter.Params) error

func makeRouterHandleFunc(f routerFunc) httprouter.Handle {
	return func(res http.ResponseWriter, req *http.Request, params httprouter.Params) {
		if err := f(res, req, params); err != nil {
			Error().Println(err.Error())
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
	templ, err := template.ParseFiles("templates/home.html")
	templ.Execute(res, "")
	return err
}
