package releasesjson

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"gophers.dev/pkgs/ignore"
	"io"
	"log"
	"net/url"
	"strings"

	"golang.org/x/crypto/openpgp"
	"gophers.dev/cmds/hc-install/internal/httpclient"
)

type ChecksumDownloader struct {
	ProductVersion   *ProductVersion
	Logger           *log.Logger
	ArmoredPublicKey string

	BaseURL string
}

type ChecksumFileMap map[string]HashSum

type HashSum []byte

func (hs HashSum) Size() int {
	return len(hs)
}

func (hs HashSum) String() string {
	return hex.EncodeToString(hs)
}

func HashSumFromHexDigest(hexDigest string) (HashSum, error) {
	sumBytes, err := hex.DecodeString(hexDigest)
	if err != nil {
		return nil, err
	}
	return HashSum(sumBytes), nil
}

func (cd *ChecksumDownloader) DownloadAndVerifyChecksums() (ChecksumFileMap, error) {
	sigFilename, err := cd.findSigFilename(cd.ProductVersion)
	if err != nil {
		return nil, err
	}

	client := httpclient.NewHTTPClient()
	sigURL := fmt.Sprintf("%s/%s/%s/%s", cd.BaseURL,
		url.PathEscape(cd.ProductVersion.Name),
		url.PathEscape(cd.ProductVersion.RawVersion),
		url.PathEscape(sigFilename))
	cd.Logger.Printf("downloading signature from %s", sigURL)
	sigResp, err := client.Get(sigURL)
	if err != nil {
		return nil, err
	}

	if sigResp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to download signature from %q: %s", sigURL, sigResp.Status)
	}

	defer ignore.Close(sigResp.Body)

	shasumsURL := fmt.Sprintf("%s/%s/%s/%s", cd.BaseURL,
		url.PathEscape(cd.ProductVersion.Name),
		url.PathEscape(cd.ProductVersion.RawVersion),
		url.PathEscape(cd.ProductVersion.SHASUMS))
	cd.Logger.Printf("downloading checksums from %s", shasumsURL)
	sumsResp, err := client.Get(shasumsURL)
	if err != nil {
		return nil, err
	}

	if sumsResp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to download checksums from %q: %s", shasumsURL, sumsResp.Status)
	}

	defer ignore.Close(sumsResp.Body)

	var shaSums strings.Builder
	sumsReader := io.TeeReader(sumsResp.Body, &shaSums)

	err = cd.verifySumsSignature(sumsReader, sigResp.Body)
	if err != nil {
		return nil, err
	}

	return fileMapFromChecksums(shaSums)
}

func fileMapFromChecksums(checksums strings.Builder) (ChecksumFileMap, error) {
	csMap := make(ChecksumFileMap, 0)

	lines := strings.Split(checksums.String(), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) != 2 {
			return nil, fmt.Errorf("unexpected checksum line format: %q", line)
		}

		h, err := HashSumFromHexDigest(parts[0])
		if err != nil {
			return nil, err
		}

		if h.Size() != sha256.Size {
			return nil, fmt.Errorf("unexpected sha256 format (len: %d, expected: %d)",
				h.Size(), sha256.Size)
		}

		csMap[parts[1]] = h
	}
	return csMap, nil
}

func compareChecksum(logger *log.Logger, r io.Reader, verifiedHashSum HashSum) error {
	h := sha256.New()
	_, err := io.Copy(h, r)
	if err != nil {
		return err
	}

	calculatedSum := h.Sum(nil)
	if !bytes.Equal(calculatedSum, verifiedHashSum) {
		return fmt.Errorf("checksum mismatch (expected %q, calculated %q)",
			verifiedHashSum,
			hex.EncodeToString(calculatedSum))
	}

	logger.Printf("checksum matches: %q", hex.EncodeToString(calculatedSum))

	return nil
}

func (cd *ChecksumDownloader) verifySumsSignature(checksums, signature io.Reader) error {
	el, err := cd.keyEntityList()
	if err != nil {
		return err
	}

	_, err = openpgp.CheckDetachedSignature(el, checksums, signature)
	if err != nil {
		return fmt.Errorf("unable to verify checksums signature: %w", err)
	}

	cd.Logger.Printf("checksum signature is valid")

	return nil
}

func (cd *ChecksumDownloader) findSigFilename(pv *ProductVersion) (string, error) {
	sigFiles := pv.SHASUMSSigs
	if len(sigFiles) == 0 {
		sigFiles = []string{pv.SHASUMSSig}
	}

	keyIds, err := cd.pubKeyIds()
	if err != nil {
		return "", err
	}

	for _, filename := range sigFiles {
		for _, keyID := range keyIds {
			if strings.HasSuffix(filename, fmt.Sprintf("_SHA256SUMS.%s.sig", keyID)) {
				return filename, nil
			}
		}
		if strings.HasSuffix(filename, "_SHA256SUMS.sig") {
			return filename, nil
		}
	}

	return "", fmt.Errorf("no suitable sig file found")
}

func (cd *ChecksumDownloader) pubKeyIds() ([]string, error) {
	entityList, err := cd.keyEntityList()
	if err != nil {
		return nil, err
	}

	fingerprints := make([]string, 0)
	for _, entity := range entityList {
		fingerprints = append(fingerprints, entity.PrimaryKey.KeyIdShortString())
	}

	return fingerprints, nil
}

func (cd *ChecksumDownloader) keyEntityList() (openpgp.EntityList, error) {
	if cd.ArmoredPublicKey == "" {
		return nil, fmt.Errorf("no public key provided")
	}
	return openpgp.ReadArmoredKeyRing(strings.NewReader(cd.ArmoredPublicKey))
}
