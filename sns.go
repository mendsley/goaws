// Copyright 2012 Matthew Endsley
// All rights reserved
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted providing that the following conditions
// are met:
// 1. Redistributions of source code must retain the above copyright
//    notice, this list of conditions and the following disclaimer.
// 2. Redistributions in binary form must reproduce the above copyright
//    notice, this list of conditions and the following disclaimer in the
//    documentation and/or other materials provided with the distribution.
//
// THIS SOFTWARE IS PROVIDED BY THE AUTHOR ``AS IS'' AND ANY EXPRESS OR
// IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
// ARE DISCLAIMED.  IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR ANY
// DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
// DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS
// OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION)
// HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT,
// STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING
// IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
// POSSIBILITY OF SUCH DAMAGE.

package goaws

import (
	"encoding/xml"
	"errors"
	"net/http"
	"net/url"
)

// A context holding the ARN/host pair for an SNS topic.
type Topic struct {
	host string
	arn  string
}

// Create an SNS Topic context for a specific host/ARN combination.
func NewTopic(host, arn string) Topic {
	return Topic{
		host: host,
		arn:  arn,
	}
}

// Publish a message to the SNS topic using the specified Context to
// sign the request.
func (t Topic) Publish(c Context, body string) (messageId, requestId string, err error) {

	params := make(url.Values)
	params.Set("TopicArn", t.arn)
	params.Set("Message", body)
	params.Set("Action", "Publish")

	req, err := http.NewRequest("GET", "https://"+t.host+"/?"+params.Encode(), nil)
	if err != nil {
		return "", "", errors.New("Failed to create request: " + err.Error())
	}

	c.SignRequest(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", errors.New("Failed to do request: " + err.Error())
	}

	var response struct {
		PublishResult struct {
			MessageId string
		}
		ResponseMetadata struct {
			RequestId string
		}
	}

	defer resp.Body.Close()
	if err := xml.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", "", errors.New("Malformed response: " + err.Error())
	}

	return response.PublishResult.MessageId, response.ResponseMetadata.RequestId, nil
}
