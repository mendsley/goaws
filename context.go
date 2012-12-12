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
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// A context object holds the credentials needed
// to sign/verify AWS requests.
type Context struct {
	keyId string
	key   string
}

// Create a new context with a given AWS Access Key ID and
// Access Key.
func NewContext(accessKeyId, accessKey string) Context {
	return Context{
		keyId: accessKeyId,
		key:   accessKey,
	}
}

type signingContext int

const (
	defaultHTTPSigningContext = signingContext(iota)
	purchaseSigningContext
)

func (sc signingContext) getValues(c Context, r *http.Request) url.Values {
	params := r.URL.Query()
	switch sc {
	case defaultHTTPSigningContext:
		params.Set("Timestamp", time.Now().UTC().Format(time.RFC3339))
		params.Set("AWSAccessKeyId", c.keyId)
		params.Set("SignatureVersion", "2")
		params.Set("SignatureMethod", "HmacSHA256")
		return params

	case purchaseSigningContext:
		params.Set("accessKey", c.keyId)
		params.Set("signatureVersion", "2")
		params.Set("signatureMethod", "HmacSHA256")
		return params
	}

	panic("Unknown signing context")
}

func (sc signingContext) addSignature(v url.Values, signature string) {
	switch sc {
	case defaultHTTPSigningContext:
		v.Set("Signature", signature)

	case purchaseSigningContext:
		v.Set("signature", signature)

	default:
		panic("Unknown signing context")
	}
}

// Signs an HTTP request using SignatureVersion 2 and HmacSHA256.
func (c Context) SignRequest(r *http.Request) {
	c.sign(defaultHTTPSigningContext, r)
}

func (c Context) sign(sc signingContext, r *http.Request) {
	params := sc.getValues(c, r)

	values := strings.Split(params.Encode(), "&")
	sort.Strings(values)
	queryString := strings.Join(values, "&")
	queryString = strings.Replace(queryString, "+", "%20", -1)
	queryString = strings.Replace(queryString, "(", "%28", -1)
	queryString = strings.Replace(queryString, ")", "%29", -1)

	var signString bytes.Buffer
	signString.WriteString(r.Method)
	signString.WriteRune('\n')
	signString.WriteString(r.URL.Host)
	signString.WriteRune('\n')
	signString.WriteString(r.URL.Path)
	signString.WriteRune('\n')
	signString.WriteString(queryString)

	sign := hmac.New(sha256.New, []byte(c.key))
	sign.Write(signString.Bytes())

	signature := base64.StdEncoding.EncodeToString(sign.Sum(nil))
	sc.addSignature(params, signature)

	r.URL.RawQuery = params.Encode()
}
