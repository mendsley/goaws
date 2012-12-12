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
	"strings"
)

// Context for store-wide settings
type Store struct {
	Sandbox   bool
	ReturnURL string
}

// Defines a purchasable item.
type Purchase struct {
	Description string
	Price       string
	ReferenceId string
}

// Create a URL to purchase an item.
func (store Store) CreatePurchaseURL(c Context, item Purchase) (string, error) {

	if !strings.HasPrefix(item.Price, "USD ") {
		return "", errors.New("AWS only supports USD prices")
	}

	params := make(url.Values)
	params.Set("description", item.Description)
	params.Set("amount", item.Price)
	params.Set("cobrandingStyle", "logo")
	params.Set("immediateReturn", "1")
	params.Set("processImmediate", "0")
	if item.ReferenceId != "" {
		params.Set("referenceId", item.ReferenceId)
	}
	params.Add("returnURL", store.ReturnURL)

	host := "https://authorize.payments.amazon.com/pba/paypipeline?"
	if store.Sandbox {
		host = "https://authorize.payments-sandbox.amazon.com/pba/paypipeline?"
	}

	req, err := http.NewRequest("GET", host+params.Encode(), nil)
	if err != nil {
		return "", errors.New("Failed to build request: " + err.Error())
	}

	c.sign(purchaseSigningContext, req)

	return req.URL.String(), nil
}

// Get the status of a transaction by id
func (store Store) GetTransactionStatus(c Context, transactionId string) error {

	params := make(url.Values)
	params.Set("Action", "GetTransactionStatus")
	params.Set("TransactionId", transactionId)
	params.Set("Version", "2008-09-17")

	host := "https://fps.amazonaws.com/?"
	if store.Sandbox {
		host = "https://fps.sandbox.amazonaws.com/?"
	}

	req, err := http.NewRequest("GET", host+params.Encode(), nil)
	if err != nil {
		return errors.New("Failed to build request: " + err.Error())
	}

	c.SignRequest(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.New("Failed to contact Amazon: " + err.Error())
	}

	var response struct {
		GetTransactionStatusResult struct {
			TransactionId     string
			TransactionStatus string
			StatusCode        string
			StatusMessage     string
		}
	}

	err = xml.NewDecoder(resp.Body).Decode(&response)
	resp.Body.Close()
	if err != nil {
		return errors.New("Failed to parse Amazon response: " + err.Error())
	}

	if response.GetTransactionStatusResult.StatusCode != "Success" {
		return errors.New("Amazon returned an invalid status: (" + response.GetTransactionStatusResult.StatusCode + ") " + response.GetTransactionStatusResult.StatusMessage)
	}

	return nil
}

// Settle a transaction that has been reserved
func (store Store) SettleTransaction(c Context, transactionId, amount string) error {

	if !strings.HasPrefix(amount, "USD ") {
		return errors.New("Cannot settle a non-USD transaction")
	}

	params := make(url.Values)
	params.Set("Action", "Settle")
	params.Set("ReserveTransactionId", transactionId)
	params.Set("TransactionAmount.CurrencyCode", "USD")
	params.Set("TransactionAmount.Value", amount[4:])
	params.Set("Version", "2008-09-17")

	host := "https://fps.amazonaws.com/?"
	if store.Sandbox {
		host = "https://fps.sandbox.amazonaws.com/?"
	}

	req, err := http.NewRequest("GET", host+params.Encode(), nil)
	if err != nil {
		return errors.New("Failed to build request: " + err.Error())
	}

	c.SignRequest(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.New("Failed to contact Amazon: " + err.Error())
	}

	var response struct {
		SettleResult struct {
			TransactionId     string
			TransactionStatus string
		}
		Errors struct {
			Error []struct {
				Code    string
				Message string
			}
		}
	}

	err = xml.NewDecoder(resp.Body).Decode(&response)
	resp.Body.Close()
	if err != nil {
		return errors.New("Failed to decode response from Amazon: " + err.Error())
	}

	if len(response.Errors.Error) > 0 {
		return errors.New("Amazon returned an error: " + response.Errors.Error[0].Message)
	}

	return nil
}

// Verify the parameters for a set of FPS parameters
func (store Store) VerifyPaymentParams(c Context, v url.Values) error {

	params := make(url.Values)
	params.Set("Action", "VerifySignature")
	params.Set("UrlEndPoint", store.ReturnURL)
	params.Set("HttpParameters", v.Encode())
	params.Set("Version", "2008-09-17")

	host := "https://fps.amazonaws.com/?"
	if store.Sandbox {
		host = "https://fps.sandbox.amazonaws.com/?"
	}

	req, err := http.NewRequest("GET", host+params.Encode(), nil)
	if err != nil {
		return errors.New("Failed to build request: " + err.Error())
	}

	c.SignRequest(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.New("Failed to contact Amazon: " + err.Error())
	}

	var response struct {
		VerifySignatureResult struct {
			VerificationStatus string
		}
		Errors struct {
			Error []struct {
				Code    string
				Message string
			}
		}
	}

	err = xml.NewDecoder(resp.Body).Decode(&response)
	resp.Body.Close()
	if err != nil {
		return errors.New("Failed to decode response from Amazon: " + err.Error())
	}

	if len(response.Errors.Error) > 0 {
		return errors.New("Failed to validate signature: " + response.Errors.Error[0].Message)
	}

	if response.VerifySignatureResult.VerificationStatus != "Success" {
		return errors.New("Invalid signature verification: " + response.VerifySignatureResult.VerificationStatus)
	}

	return nil
}
