// update.go - enables to self- or autoupdate a binary from github releases.
//
package main

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto"
	"crypto/sha256"
	//"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/inconshreveable/go-update"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	//"net/http/httputil"
	"os"
	"runtime"
	"strconv"
	"strings"
)

var err error // global error variable

// the following variables must be changed in order to meet your project details:
var github_user = "scusi"   // your github account name
var github_project = "mwdl" // your github project name
var binary_name = "mwdl"    // name of your binary within your release archives
var windows_ext = ".exe"    // windows extension for your binary within your release archives

// replace publicKey with your public ECDSA key in pem format
var publicKey = []byte(`
-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEDDEtBqbRWOGkYlJLONyuGSndiD+C
lApqBbwd5Rk97zGjaPNJcblIt55s48IxmQU7OA7TxH0zHNfIetjUfguXkA==
-----END PUBLIC KEY-----
`)

// DO NOT CHANGE ANYTHING BELOW THIS LINE, UNLESS YOU KNOW EXACTLY WHAT YOU ARE DOING.

// GetLatestRelease - will get the assetID and tag_name of the latest release.
// the tag_name can be used to see if the latest version is new than the one running.
// the assedID is used in subsequent calls to the github API in oder to get the
// right archive from the latest release.
func GetLatestRelease() (releaseID string, tag_name string, err error) {
	resp, err := http.Get("https://api.github.com/repos/" + github_user + "/" + github_project + "/releases/latest")
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return
	}
	releaseID_float := result["id"].(float64)
	releaseID_int := int(releaseID_float)
	//log.Printf("releaseID: %d (type: %T)\n", releaseID_int, releaseID_int)
	releaseID = strconv.Itoa(releaseID_int)

	tag_name = result["tag_name"].(string)
	return
}

// GetMatchingAssetDownloadURL - will take a releaseID and return the
// download URL for the asset that matches the GOOS and GOARCH
// of the running program.
func GetMatchingAssetDownloadURL(releaseID string) (downloadURL string, err error) {
	resp, err := http.Get("https://api.github.com/repos/" + github_user + "/" + github_project + "/releases/" + releaseID + "/assets")
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	var results []map[string]interface{}
	err = json.Unmarshal(body, &results)
	if err != nil {
		return
	}
	for _, result := range results {
		name := result["name"].(string)
		if strings.Contains(name, runtime.GOOS) && strings.Contains(name, runtime.GOARCH) {
			//if strings.Contains(name, "windows") && strings.Contains(name, runtime.GOARCH) { // for testing
			//log.Printf("[%02d]: %v\n", k, int(result["id"].(float64)))
			//log.Printf("[%02d]: %v\n", k, result["name"])
			//log.Printf("[%02d]: %v\n", k, result["browser_download_url"])
			downloadURL = result["browser_download_url"].(string)
		}
	}
	return
}

// DownloadAsset - downloads an asset from a given download URL
func DownloadAsset(downloadURL string) (asset []byte, err error) {
	resp, err := http.Get(downloadURL)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	//dumpResp, err := httputil.DumpResponse(resp, false)
	if err != nil {
		return
	}
	//log.Printf("Response:\n%s\n", dumpResp)

	asset, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	/*
		tempFile, err := ioutil.TempFile(os.TempDir(), github_project+"-download-")
		if err != nil {
			return
		}
		tempFile.Write(asset)
		tempFile.Sync()
		log.Printf("wrote downloaded content to '%s'\n", tempFile.Name())
	*/
	return
}

// UnpackAsset - will take an asset archive and unpack it, returns the binary
func UnpackAsset(asset []byte) (binary []byte, err error) {
	// check archiv format (tar.gz or zip)
	// tar.gz == 1f8b
	// zip == 504b
	magicSig := make([]byte, 2)
	magicSig = asset[0:2]
	//log.Printf("magic is: '%s'\n", hex.Dump(magicSig))
	//log.Printf("magic is: '%s'\n", hex.Dump(asset[0:4]))

	if bytes.Equal(magicSig, []byte{0x1F, 0x8B}) {
		if debug {
			log.Printf("detected gzip file")
		}
		// it is a gzipped file
		// binary is mwdl
		assetReader := bytes.NewReader(asset)
		gzf, err := gzip.NewReader(assetReader)
		if err != nil {
			return binary, err
		}
		tarReader := tar.NewReader(gzf)
		for {
			header, err := tarReader.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return binary, err
			}
			name := header.Name
			// if it is named right and is a regular file
			if name == binary_name && header.Typeflag == tar.TypeReg {
				var b bytes.Buffer
				bw := bufio.NewWriter(&b)
				n, err := io.Copy(bw, tarReader)
				if err != nil {
					return binary, err
				}
				bw.Flush()
				binary = b.Bytes()
				if debug {
					log.Printf("copied %d bytes from %s\n", n, name)
				}
				break
			} else {
				if debug {
					log.Printf("ignored '%s', continue...", name)
				}
				continue

			}
		}
		return binary, err
	} else if bytes.Equal(magicSig, []byte{0x50, 0x4B}) {
		if debug {
			log.Printf("detected zip file")
		}
		// it is a ZIP file
		// write asset into a temporary file
		zipTempFile, err := ioutil.TempFile(os.TempDir(), github_project+"-update-")
		if err != nil {
			return binary, err
		}
		_, err = zipTempFile.Write(asset)
		if err != nil {
			return binary, err
		}
		defer zipTempFile.Close()
		// unpack temporary file
		zipReader, err := zip.OpenReader(zipTempFile.Name())
		if err != nil {
			return binary, err
		}
		for _, f := range zipReader.File {
			// find the binary
			// binary filename is "mwdl.exe"
			if f.Name == binary_name+windows_ext {
				var b bytes.Buffer
				bw := bufio.NewWriter(&b)
				// open the binary file from the archive
				rc, err := f.Open()
				if err != nil {
					return binary, err
				}
				// copy the binary file content from the archive into 'binary' slice.
				//n, err := io.Copy(bw, rc)
				_, err = io.Copy(bw, rc)
				if err != nil {
					return binary, err
				}
				rc.Close()
				//log.Printf("copied %d bytes from %s\n", n, f.Name)
			} else {
				// if the filename is not what we look for, continue to next file in archive.
				continue
			}
		}
		err = os.Remove(zipTempFile.Name())
		if err != nil {
			log.Printf("removing temprary file '%s' failed: %s\n", zipTempFile.Name(), err.Error())
		}
		return binary, err
	} else {
		// it is something else
		// dump it and return error
		log.Printf("%s", hex.Dump(asset[0:10]))
		err := fmt.Errorf("no supported archive format found")
		return binary, err
	}
}

// Update - will take the binary, validate it and if OK updates itself
func Update(binary []byte) (err error) {
	// caculate checksum
	hash := sha256.New()
	//hash := sha1.New()
	hash.Write(binary)
	checksum := hash.Sum(nil)
	log.Printf("Checksum is: %x\n", checksum)
	// go get the signature for checksum from signature server
	/*
	resp, err := http.Get("http://127.0.0.1:9999/signature/" + fmt.Sprintf("%x", checksum))
	if err != nil {
		return
	}
	defer resp.Body.Close()
	signature, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	*/
	signatureA := "cf7cf5f67dcf832502956468ce9644bebc8f4f6684b6a497ba113f981a2a404a"
	signature, err := hex.DecodeString(signatureA)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Signature is: %x", signature)
	// TODO: prepare update options
	opts := update.Options{
		Checksum:  checksum,
		Signature: signature,
		Hash:      crypto.SHA256,             // this is the default, you don't need to specify it
		//Hash:      crypto.SHA1, 
		Verifier:  update.NewECDSAVerifier(), // this is the default, you don't need to specify it
	}
	err = opts.SetPublicKeyPEM(publicKey)
	if err != nil {
		log.Println("Set public key failed " + err.Error())
		//log.Println(err)
		return
	}
	// TODO: validate checksum and signature of the binary and apply update
	update_data_reader := bytes.NewReader(binary)
	//err = update.Apply(update_data_reader, opts)
	err = update.Apply(update_data_reader, update.Options{})
	if err != nil {
		// error handling
		log.Println(err)
		return
	}
	return
}
