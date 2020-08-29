package util

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"

	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
)

// EnsureDir ensures the given directory path exists, if not it will create the full path
func EnsureDir(dirPath string) error {
	if _, serr := os.Stat(dirPath); os.IsNotExist(serr) {
		merr := os.MkdirAll(dirPath, os.ModePerm)
		if merr != nil {
			return merr
		}
	}
	return nil
}

// LoadExternalFile helper for reading data from references to external files
var LoadExternalFile = func(path string) ([]byte, error) {
	realpath, err := homedir.Expand(path)
	if err != nil {
		return []byte{}, err
	}

	filedata, err := ioutil.ReadFile(realpath)
	if err != nil {
		return []byte{}, err
	}
	return filedata, nil
}

// Download a file from the URL to directory, specified as arguments.
func DownloadFile(uri, dir string) (err error) {
	if uri == "" {
		return fmt.Errorf("Empty URL, can't continue")
	}
	log.Debugf("Starting download URL \"%s\" to dir \"%s\"\n", uri, dir)
	urlObject, err := ResolveURL(uri)
	log.Debugf("Found URL object: %v\n", urlObject)
	if err != nil {
		return err
	}
	fname := path.Base(urlObject.Path)
	log.Debugf("Downloading file name \"%s\"\n", fname)
	response, err := http.Get(uri)
	log.Debugf("Got HTTP response from \"%s\": %s\n", uri, response.Status)
	if response.StatusCode > 399 {
		err = fmt.Errorf("Invalid server response for \"%s\": %s\n", uri, response.Status)
		return err
	}
	if err != nil {
		return err
	}
	defer response.Body.Close()

	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	fpath := path.Join(dir, fname)
	mode := os.FileMode(uint32(0644))
	log.Debugf("Starting write to file \"%s\"\n", fpath)
	err = ioutil.WriteFile(fpath, data, mode)
	if err != nil {
		return err
	}
	return
}

func ExtractTarball(f, p string) error {
	fp, err := homedir.Expand(f)
	if err != nil {
		return err
	}
	p, err = homedir.Expand(p)
	if err != nil {
		return err
	}
	cmd := exec.Command("tar", "xf", fp, "-C", p)
	log.Debugf("Perpared command for extracting: %v\n", cmd)
	if err = cmd.Run(); err != nil {
		return err
	}
	log.Printf("Extracted \"%s\" to dir \"%s\"\n", fp, p)
	return nil
}
