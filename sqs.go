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
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// A context holder the data for an SQS queue.
type Queue struct {
	url string
}

// Create a SQS queue given it's URL.
func NewQueue(url string) Queue {
	return Queue{
		url: url,
	}
}

type SQSMessage struct {
	ReceiptHandle string
	Body          string
}

// Recieves messages from the SQS queue using the specified context to
// sign the reques. Retreives at most `max` messages waiting at most
// the duration specified by `wait`.
func (q Queue) ReceiveMessages(c Context, max int, wait time.Duration) (messages []SQSMessage, err error) {

	seconds := int(wait.Seconds())
	if seconds < 0 || seconds > 20 {
		return nil, fmt.Errorf("Wait time must be no longer than 20 seconds. Got: %d", seconds)
	}

	if max < 0 || max > 10 {
		return nil, fmt.Errorf("Max messages must be no larger than 10. Got: %d", max)
	}

	params := make(url.Values)
	params.Set("Action", "ReceiveMessage")
	params.Set("MaxNumberOfMessages", strconv.FormatInt(int64(max), 10))
	params.Set("VisibilityTimeout", "5")
	params.Set("WaitTimeSeconds", strconv.FormatInt(int64(seconds), 10))
	params.Set("Version", "2009-02-01")

	req, err := http.NewRequest("GET", q.url+"/?"+params.Encode(), nil)
	if err != nil {
		return nil, errors.New("Failed to create request: " + err.Error())
	}

	c.SignRequest(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.New("Failed to do request: " + err.Error())
	}

	defer resp.Body.Close()

	var response struct {
		ReceiveMessageResult struct {
			Message []struct {
				MessageId     string
				ReceiptHandle string
				MD5OfBody     string
				Body          string
				Attribute     []struct {
					Name  string
					Value string
				}
			}
		}
		ResponseMetadata struct {
			RequestId string
		}
	}

	if err := xml.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, errors.New("Malformed response: " + err.Error())
	}

	count := len(response.ReceiveMessageResult.Message)
	if count > 0 {
		messages = make([]SQSMessage, count)
		for ii, msg := range response.ReceiveMessageResult.Message {
			messages[ii].ReceiptHandle = msg.ReceiptHandle
			messages[ii].Body = msg.Body
		}
	}

	err = nil
	return
}

// Delete a message from the queue.
func (q Queue) DeleteMessage(c Context, receiptHandle string) error {

	params := make(url.Values)
	params.Set("Action", "DeleteMessage")
	params.Set("ReceiptHandle", receiptHandle)
	params.Set("Version", "2009-02-01")

	req, err := http.NewRequest("GET", q.url+"/?"+params.Encode(), nil)
	if err != nil {
		return errors.New("Failed to create request: " + err.Error())
	}

	c.SignRequest(req)

	_, err = http.DefaultClient.Do(req)
	if err != nil {
		return errors.New("Failed to do request: " + err.Error())
	}

	return nil
}
