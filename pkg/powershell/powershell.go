package util

import (
	"encoding/base64"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
)

const PipeHasEnded = "The pipe has been ended."
const PipeIsBeingClosed = "The pipe is being closed."

func UploadCmd(path string) string {
	return EncodeCmd(`
		begin {
			$path = "` + path + `"
			Remove-Item $path -ErrorAction Ignore
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
}

func EncodeCmd(psCmd string) string {
	// 2 byte chars to make PowerShell happy
	wideCmd := ""
	for _, b := range []byte(psCmd) {
		wideCmd += string(b) + "\x00"
	}

	// Base64 encode the command
	input := []uint8(wideCmd)
	return base64.StdEncoding.EncodeToString(input)
}

// Powershell wraps a PowerShell script
// and prepares it for execution by the winrm client
func Cmd(psCmd string) string {
	encodedCmd := EncodeCmd(psCmd)

	log.Debugf("encoded powershell command: %s", psCmd)
	// Create the powershell.exe command line to execute the script
	return fmt.Sprintf("powershell.exe -NonInteractive -NoProfile -EncodedCommand %s", encodedCmd)
}

// from jbrekelmans/go-winrm/util.go PowerShellSingleQuotedStringLiteral
func SingleQuote(v string) string {
	var sb strings.Builder
	_, _ = sb.WriteRune('\'')
	for _, rune := range v {
		var esc string
		switch rune {
		case '\n':
			esc = "`n"
		case '\r':
			esc = "`r"
		case '\t':
			esc = "`t"
		case '\a':
			esc = "`a"
		case '\b':
			esc = "`b"
		case '\f':
			esc = "`f"
		case '\v':
			esc = "`v"
		case '"':
			esc = "`\""
		case '\'':
			esc = "`'"
		case '`':
			esc = "``"
		case '\x00':
			esc = "`0"
		default:
			_, _ = sb.WriteRune(rune)
			continue
		}
		_, _ = sb.WriteString(esc)
	}
	_, _ = sb.WriteRune('\'')
	return sb.String()
}

// from jbrekelmans/go-winrm/util.go PowerShellSingleQuotedStringLiteral
func DoubleQuote(v string) string {
	var sb strings.Builder
	_, _ = sb.WriteRune('"')
	for _, rune := range v {
		var esc string
		switch rune {
		case '"':
			esc = "`\""
		default:
			_, _ = sb.WriteRune(rune)
			continue
		}
		_, _ = sb.WriteString(esc)
	}
	_, _ = sb.WriteRune('"')
	return sb.String()
}
