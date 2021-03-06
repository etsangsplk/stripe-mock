package param

import (
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/stripe/stripe-mock/param/form"
	"github.com/stripe/stripe-mock/param/nestedtypeassembler"
	"github.com/stripe/stripe-mock/param/parser"
)

// ParseParams extracts parameters from a request that an application can
// consume.
//
// Depending on the type of request, parameters may be extracted from either
// the query string, a form-encoded body, or a multipart form-encoded body (the
// latter being specific to only a very small number of endpoints).
//
// Regardless of origin, parameters are assumed to follow "Rack-style"
// conventions for encoding complex types like arrays and maps, which is how
// the Stripe API decodes data. These complex types are what makes the param
// package's implementation non-trivial. We rely on the nestedtypeassembler
// subpackage to do the heavy lifting for that.
func ParseParams(r *http.Request) (map[string]interface{}, error) {
	var values form.FormValues

	contentType := r.Header.Get("Content-Type")

	// Truncate content type parameters. For example, given:
	//
	//     application/json; charset=utf-8
	//
	// We want to chop off the `; charset=utf-8` at the end.
	contentType = strings.Split(contentType, ";")[0]

	if r.Method == "GET" {
		formString := r.URL.RawQuery

		var err error
		values, err = parser.ParseFormString(formString)
		if err != nil {
			return nil, err
		}
	} else if contentType == MultipartMediaType {
		err := r.ParseMultipartForm(MaxMemory)
		if err != nil {
			return nil, err
		}

		for key, keyValues := range r.MultipartForm.Value {
			for _, keyValue := range keyValues {
				values = append(values, form.FormPair{key, keyValue})
			}
		}

		for key, keyValues := range r.MultipartForm.File {
			for _, keyFileHeader := range keyValues {
				file, err := keyFileHeader.Open()
				if err != nil {
					return nil, err
				}

				keyFileBytes, err := ioutil.ReadAll(file)
				file.Close()
				if err != nil {
					return nil, err
				}

				values = append(values, form.FormPair{key, string(keyFileBytes)})
			}
		}
	} else {
		formBytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		r.Body.Close()

		formString := string(formBytes)

		values, err = parser.ParseFormString(formString)
		if err != nil {
			return nil, err
		}
	}

	return nestedtypeassembler.AssembleParams(values)
}

//
// Private constants
//

// The maximum amount of memory allowed when ingesting a multipart form.
//
// Set to 1 MB.
const MaxMemory = 1 * 1024 * 1024

const MultipartMediaType = "multipart/form-data"
