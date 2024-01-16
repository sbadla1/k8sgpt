/*
Copyright 2023 The K8sGPT Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package analyzer

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/k8sgpt-ai/k8sgpt/pkg/common"
	"github.com/k8sgpt-ai/k8sgpt/pkg/util"
)

type SplunkAnalyzer struct {
}

func (SplunkAnalyzer) Analyze(a common.Analyzer) ([]common.Result, error) {

	kind := "Splunk"

	AnalyzerErrorsMetric.DeletePartialMatch(map[string]string{
		"analyzer_name": kind,
	})

	var preAnalysis = map[string]common.PreAnalysis{}
	var failures []common.Failure
	// fetch metatdata: cluster_name, namespace, splunk_rest_url, component_name, time range, auth etc
	// cluster := ""
	namespace := a.Namespace
	// component := ""
	splunk_user := "rest_k8sgpt"
	splunk_pass := "F9rOHNncHQtLXNwbHVuay1jbGl"
	splunk_rest_url := "https://splunk-rest-us-west-2.ppd2.cmn.sbg.a.intuit.com/services/search/jobs/export"

	// construct splunk query
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	var data = strings.NewReader(fmt.Sprintf("search index=iks kubernetes_namespace=%s earliest=-1m latest=now | head 3&output_mode=json", namespace))

	// create search request
	req, err := http.NewRequest("POST", splunk_rest_url, data)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(splunk_user, splunk_pass)

	// wait and get the search results
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		failures = append(failures, common.Failure{
			Text: fmt.Sprintf("Error: %s reading response from splunk instance %s", err.Error(), splunk_rest_url),
		})
	} else {
		failures = append(failures, common.Failure{
			Text: fmt.Sprintf("%s\n", bodyText),
		})
	}

	if len(failures) > 0 {
		preAnalysis[fmt.Sprintf("splunk/%s", splunk_rest_url)] = common.PreAnalysis{
			FailureDetails: failures,
		}
		AnalyzerErrorsMetric.WithLabelValues(kind, "component_a", splunk_rest_url).Set(float64(len(failures)))
	}

	for key, value := range preAnalysis {
		var currentAnalysis = common.Result{
			Kind:  kind,
			Name:  key,
			Error: value.FailureDetails,
		}

		parent, _ := util.GetParent(a.Client, value.Pod.ObjectMeta)
		currentAnalysis.ParentObject = parent
		a.Results = append(a.Results, currentAnalysis)
	}

	return a.Results, nil
}
