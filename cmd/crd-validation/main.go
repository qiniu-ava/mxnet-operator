package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	crdvalidation "github.com/ant31/crd-validation/pkg"
	"github.com/ghodss/yaml"
	"github.com/qiniu-ava/mxnet-operator/pkg/apis/qiniu/v1alpha1"
	apiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

const (
	crdName = "github.com/qiniu-ava/mxnet-operator/pkg/apis/qiniu/v1alpha1.MXJob"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("mxjob-crdvalidation {in-template} {out-generated}")
		return
	}

	in := os.Args[1]
	out := os.Args[2]
	b, e := ioutil.ReadFile(in)
	if e != nil {
		log.Fatal("read template file error:", e)
	}

	inJSON, e := yaml.YAMLToJSON(b)
	if e != nil {
		log.Fatal("failed to convert yaml to json:", e)
	}

	crdTempl := apiext.CustomResourceDefinition{}
	if e := json.Unmarshal(inJSON, &crdTempl); e != nil {
		log.Fatal("invalid yaml template file:", e)
	}

	genValidation := crdvalidation.GetCustomResourceValidation(crdName, v1alpha1.GetOpenAPIDefinitions)
	// TODO: merge with the validation of template?
	crdTempl.Spec.Validation = genValidation

	outJSON, e := json.MarshalIndent(crdTempl, "", "    ")
	if e != nil {
		log.Fatal("failed to convert object to json", e)
	}

	outYAML, e := yaml.JSONToYAML(outJSON)
	if e != nil {
		log.Fatal("failed to convert json to yaml:", e)
	}

	if e := ioutil.WriteFile(out, outYAML, 0666); e != nil {
		log.Fatal("write generated file error:", e)
	}
}
