/*
Copyright © 2021 Sniptt <support@sniptt.com>

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
package client

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/sniptt-official/ots/build"
	"github.com/spf13/viper"
)

type CreateOtsReq struct {
	EncryptedBytes string `json:"encryptedBytes"`
	ExpiresIn      uint32 `json:"expiresIn"`
}

type CreateOtsRes struct {
	Id        string `json:"id"`
	ExpiresAt int64  `json:"expiresAt"`
	ViewURL   *url.URL
}

func CreateOts(encryptedBytes []byte, expiresIn time.Duration, region string) (*CreateOtsRes, error) {
	baseUrl := viper.GetString("base_url")

	region, err := getRegion(region)
	if err != nil {
		return nil, err
	}

	reqUrl := url.URL{
		Scheme: "https",
		Host:   fmt.Sprintf("ots.%s.%s", region, baseUrl),
		Path:   "secrets",
	}

	reqBody := &CreateOtsReq{
		EncryptedBytes: base64.StdEncoding.EncodeToString(encryptedBytes),
		ExpiresIn:      uint32(expiresIn.Seconds()),
	}

	resBody := &CreateOtsRes{}

	payloadBuf := new(bytes.Buffer)
	err = json.NewEncoder(payloadBuf).Encode(reqBody)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}

	req, err := http.NewRequest("POST", reqUrl.String(), payloadBuf)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Client-Name", "ots-cli")
	req.Header.Add("X-Client-Version", build.Version)

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	err = decodeJSON(res, resBody)
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(res.Header.Get("X-View-Url"))
	if err != nil {
		return nil, err
	}

	resBody.ViewURL = u

	return resBody, nil
}

func getRegion(region string) (string, error) {
	switch region {
	case "us":
		return "us-east-1", nil
	case "eu":
		return "eu-central-1", nil
	default:
		return "", errors.New("invalid region")
	}
}

func decodeJSON(res *http.Response, target interface{}) error {
	statusOK := res.StatusCode >= 200 && res.StatusCode < 300

	if !statusOK {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}

		// TODO: Implement error struct response.
		return errors.New(string(body))
	}

	return json.NewDecoder(res.Body).Decode(target)
}
