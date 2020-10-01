package winrm

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jbrekelmans/winrm"
	log "github.com/sirupsen/logrus"
)

// Adapted from https://github.com/jbrekelmans/go-winrm/copier.go by Jasper Brekelmans

const pipeHasEnded = "The pipe has been ended."
const pipeIsBeingClosed = "The pipe is being closed."

// Upload uploads a file to a host
func Upload(src, dest string, c *Connection) error {
	psCmd := winrm.FormatPowerShellScriptCommandLine(`
		begin {
			$path = ` + winrm.PowerShellSingleQuotedStringLiteral(dest) + `
			$DebugPreference = "Continue"
			$ErrorActionPreference = "Stop"
			Set-StrictMode -Version 2
			$fd = [System.IO.File]::Create($path)
			$sha256 = [System.Security.Cryptography.SHA256CryptoServiceProvider]::Create()
			$bytes = @() #initialize for empty file case
		}
		process {
			$bytes = [System.Convert]::FromBase64String($input)
			$sha256.TransformBlock($bytes, 0, $bytes.Length, $bytes, 0) | Out-Null
			$fd.Write($bytes, 0, $bytes.Length)
		}
		end {
			$sha256.TransformFinalBlock($bytes, 0, 0) | Out-Null
			$hash = [System.BitConverter]::ToString($sha256.Hash).Replace("-", "").ToLowerInvariant()
			$fd.Close()
			Write-Output "{""sha256"":""$hash""}"
		}
	`)
	stat, err := os.Stat(src)
	if err != nil {
		return err
	}
	sha256DigestLocalObj := sha256.New()
	sha256DigestLocal := ""
	sha256DigestRemote := ""
	srcSize := uint64(stat.Size())
	log.Infof("%s: uploading %s to %s", c.Address, formatBytes(float64(srcSize)), dest)
	bytesSent := uint64(0)
	fdClosed := false
	fd, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if !fdClosed {
			_ = fd.Close()
			fdClosed = true
		}
	}()
	startTime := time.Now()
	lease := c.shellpool.Get()
	if lease == nil {
		return fmt.Errorf("%s: failed to create a shell", c.Address)
	}
	defer lease.Release()
	shell := lease.shell
	cmd, err := shell.StartCommand(psCmd[0], psCmd[1:], false, true)
	if err != nil {
		return err
	}
	// Since we are passing data over a powershell pipe, we encode the data as lines of base64 (each line is terminated by a carriage return +
	// line feed sequence, hence the -2)
	bufferCapacity := (shell.Client().SendInputMax() - 2) / 4 * 3
	base64LineBufferCapacity := bufferCapacity/3*4 + 2
	base64LineBuffer := make([]byte, base64LineBufferCapacity)
	base64LineBuffer[base64LineBufferCapacity-2] = '\r'
	base64LineBuffer[base64LineBufferCapacity-1] = '\n'
	buffer := make([]byte, bufferCapacity)
	bufferLength := 0
	ended := false
	for {
		var n int
		n, err = fd.Read(buffer)
		bufferLength += n
		if err != nil {
			break
		}
		if bufferLength == bufferCapacity {
			base64.StdEncoding.Encode(base64LineBuffer, buffer)
			bytesSent += uint64(bufferLength)
			_, _ = sha256DigestLocalObj.Write(buffer)
			if bytesSent >= srcSize {
				ended = true
				sha256DigestLocal = hex.EncodeToString(sha256DigestLocalObj.Sum(nil))
			}
			err := cmd.SendInput(base64LineBuffer, ended)
			bufferLength = 0
			if err != nil {
				return err
			}
		}
	}
	_ = fd.Close()
	fdClosed = true
	if err == io.EOF {
		err = nil
	}
	if err != nil {
		cmd.Signal()
		return err
	}
	if !ended {
		_, _ = sha256DigestLocalObj.Write(buffer[:bufferLength])
		sha256DigestLocal = hex.EncodeToString(sha256DigestLocalObj.Sum(nil))
		base64.StdEncoding.Encode(base64LineBuffer, buffer[:bufferLength])
		i := base64.StdEncoding.EncodedLen(bufferLength)
		base64LineBuffer[i] = '\r'
		base64LineBuffer[i+1] = '\n'
		err = cmd.SendInput(base64LineBuffer[:i+2], true)
		if err != nil {
			if !strings.Contains(err.Error(), pipeHasEnded) && !strings.Contains(err.Error(), pipeIsBeingClosed) {
				cmd.Signal()
				return err
			}
			// ignore pipe errors that results from passing true to cmd.SendInput
		}
		ended = true
		bytesSent += uint64(bufferLength)
		bufferLength = 0
	}
	var wg sync.WaitGroup
	wg.Add(2)
	var stderr bytes.Buffer
	var stdout bytes.Buffer
	go func() {
		defer wg.Done()
		_, err = io.Copy(&stderr, cmd.Stderr)
		if err != nil {
			stderr.Reset()
		}
	}()
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(cmd.Stdout)
		for scanner.Scan() {
			var output struct {
				Sha256 string `json:"sha256"`
			}
			if json.Unmarshal(scanner.Bytes(), &output) == nil {
				sha256DigestRemote = output.Sha256
			} else {
				_, _ = stdout.Write(scanner.Bytes())
				_, _ = stdout.WriteString("\n")
			}
		}
		if err := scanner.Err(); err != nil {
			stdout.Reset()
		}
	}()
	cmd.Wait()
	wg.Wait()
	duration := time.Since(startTime).Seconds()
	speed := float64(bytesSent) / duration
	log.Debugf("transfered %d bytes in %f seconds (%s/s)", bytesSent, duration, formatBytes(speed))

	if cmd.ExitCode() != 0 {
		log.WithFields(log.Fields{
			"stdout":    stdout.String(),
			"stderr":    stderr.String(),
			"exit_code": cmd.ExitCode(),
		}).Errorf("%s: non-zero exit code", c.Address)
		return fmt.Errorf("exit code non-zero")
	}
	if sha256DigestRemote == "" {
		return fmt.Errorf("copy file command did not output the expected JSON to stdout but exited with code 0")
	} else if sha256DigestRemote != sha256DigestLocal {
		return fmt.Errorf("copy file checksum mismatch (local = %s, remote = %s)", sha256DigestLocal, sha256DigestRemote)
	}

	return nil
}

func formatBytes(bytes float64) string {
	units := []string{
		"bytes",
		"KiB",
		"MiB",
		"GiB",
	}
	logBase1024 := 0
	for bytes > 1024.0 && logBase1024 < len(units) {
		bytes /= 1024.0
		logBase1024++
	}
	return fmt.Sprintf("%.3f %s", bytes, units[logBase1024])
}
